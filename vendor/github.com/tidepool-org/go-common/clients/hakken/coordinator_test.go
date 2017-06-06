package hakken

import (
	"fmt"
	common "github.com/tidepool-org/go-common"
	"github.com/tidepool-org/go-common/atomics"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPoll(t *testing.T) {
	makeServer := func(as *atomics.AtomicString) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/v1/coordinator":
				retVal := as.Get()

				if strings.Contains(retVal, "FAIL") {
					t.Error(retVal)
					return
				} else if strings.Contains(retVal, "RETURN:") {
					value, _ := strconv.Atoi(strings.Split(retVal, ":")[1])
					res.WriteHeader(value)
					return
				}

				fmt.Fprint(res, retVal)
			default:
				t.Errorf("Uknown url[%s]", req.URL.Path)
			}
		}))
	}

	coordOneReturn := &atomics.AtomicString{}
	coordTwoReturn := &atomics.AtomicString{}
	resyncReturn := &atomics.AtomicString{}

	coordOne := makeServer(coordOneReturn)
	defer coordOne.Close()
	coordTwo := makeServer(coordTwoReturn)
	defer coordTwo.Close()
	resyncServer := makeServer(resyncReturn)
	defer resyncServer.Close()

	coordOneURL := common.URLOrPanic(coordOne.URL)
	coordTwoURL := common.URLOrPanic(coordTwo.URL)

	initialVal := fmt.Sprintf(`[{"host": "%s"}, {"host": "%s"}]`, coordOneURL.Host, coordTwoURL.Host)
	coordOneReturn.Set(initialVal)
	coordTwoReturn.Set(initialVal)
	resyncReturn.Set(initialVal)

	resyncClient := coordinatorClient{Coordinator{*common.URLOrPanic(resyncServer.URL)}}

	resyncTicker := make(chan time.Time)
	pollTicker := make(chan time.Time)
	dropCooChan := make(chan *coordinatorClient)

	manager := &coordinatorManager{
		resyncClient: resyncClient,
		resyncTicker: &time.Ticker{C: resyncTicker},
		pollTicker:   &time.Ticker{C: pollTicker},
		dropCooChan:  dropCooChan,
	}

	if c := manager.getClient(); c != nil {
		t.Errorf("Expected client to NOT exist, got[%+v]", c)
	}
	manager.start()

	if c := manager.getClient(); c == nil {
		t.Errorf("Expected to get a client, didn't")
	} else if c.String() != coordOne.URL {
		t.Errorf("Expected to get client for coordOne, got[%s]", c.String())
	}
	if c := manager.getClients(); c == nil || len(c) != 2 {
		t.Errorf("Expected two clients, got [%d]", len(c))
	}

	// Remove coordOne client
	manager.reportBadClient(manager.getClient())
	<-time.After(10 * time.Millisecond)
	if c := manager.getClient(); c == nil {
		t.Errorf("Expected to get a client, didn't")
	} else if c.String() != coordTwo.URL {
		t.Errorf("Expected to get client for coordTwo, got[%s]", c.String())
	}

	coordOneReturn.Set("FAIL - coordOne should not be called")
	pollTicker <- time.Now()

	for count := 0; count < 10; count++ {
		<-time.After(10 * time.Millisecond)
		if len(manager.getClients()) == 2 {
			break
		}
	}
	if c := manager.getClient(); c == nil {
		t.Errorf("Expected to get a client, didn't")
	} else if c.String() != coordTwo.URL {
		t.Errorf("Expected to get client for coordTwo, got[%s]", c.String())
	}
	if c := manager.getClients(); c == nil || len(c) != 2 {
		t.Errorf("Expected two clients, got [%d]", len(c))
	}

	coordOneReturn.Set("RETURN:500")
	pollTicker <- time.Now()

	for count := 0; count < 10; count++ {
		<-time.After(10 * time.Millisecond)
		if len(manager.getClients()) == 1 {
			break
		}
	}
	if c := manager.getClient(); c == nil {
		t.Errorf("Expected to get a client, didn't")
	} else if c.String() != coordTwo.URL {
		t.Errorf("Expected to get client for coordTwo, got[%s]", c.String())
	}
	if c := manager.getClients(); c == nil || len(c) != 1 {
		t.Errorf("Expected one client, got [%d]", len(c))
	}

	coordTwoReturn.Set("RETURN:500")
	pollTicker <- time.Now()
	for count := 0; count < 10; count++ {
		<-time.After(10 * time.Millisecond)
		if len(manager.getClients()) == 1 {
			break
		}
	}
	if c := manager.getClient(); c != nil {
		t.Errorf("Expected to NOT get a client")
	}
	if c := manager.getClients(); c == nil || len(c) != 0 {
		t.Errorf("Expected zero clients, got [%d]", len(c))
	}

	resyncTicker <- time.Now()
	for count := 0; count < 10; count++ {
		<-time.After(10 * time.Millisecond)
		if len(manager.getClients()) == 2 {
			break
		}
	}
	if c := manager.getClient(); c == nil {
		t.Errorf("Expected to get a client, didn't")
	} else if c.String() != coordOne.URL {
		t.Errorf("Expected to get client for coordTwo, got[%s]", c.String())
	}
	if c := manager.getClients(); c == nil || len(c) != 2 {
		t.Errorf("Expected two clients, got [%d]", len(c))
	}

	if err := manager.Close(); err != nil {
		t.Errorf("Error when closing manager: %s", err)
	}
}
