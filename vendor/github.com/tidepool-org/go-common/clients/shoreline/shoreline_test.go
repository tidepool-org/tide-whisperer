package shoreline

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tidepool-org/go-common/clients/disc"
)

const name = "test"
const secret = "howdy ho, neighbor joe"
const token = "three.part.token"

func TestStart(t *testing.T) {
	srvr := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/serverlogin":
			if nam := req.Header.Get("x-tidepool-server-name"); nam != name {
				t.Errorf("Bad value for name[%v]", nam)
			}

			if sec := req.Header.Get("x-tidepool-server-secret"); sec != secret {
				t.Errorf("Bad secret value[%v]", sec)
			}

			res.Header().Set("x-tidepool-session-token", token)
		default:
			t.Errorf("Unknown path[%s]", req.URL.Path)
		}
	}))
	defer srvr.Close()

	shorelineClient := NewShorelineClientBuilder().
		WithHostGetter(disc.NewStaticHostGetterFromString(srvr.URL)).
		WithName("test").
		WithSecret("howdy ho, neighbor joe").
		WithTokenRefreshInterval(10 * time.Millisecond).
		Build()

	err := shorelineClient.Start()

	if err != nil {
		t.Errorf("Failed start with error[%v]", err)
	}
	if tok := shorelineClient.SecretProvide(); tok != secret {
		t.Errorf("Unexpected secret[%s]", tok)
	}

	<-time.After(100 * time.Millisecond)
	shorelineClient.Close()
}

func TestLogin(t *testing.T) {
	srvr := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/serverlogin":
			res.Header().Set("x-tidepool-session-token", token)
		case "/login":
			if auth := req.Header.Get("Authorization"); auth != "Basic YmlsbHk6aG93ZHk=" {
				t.Errorf("Bad Authorization Header[%v]", auth)
			}

			res.Header().Set("x-tidepool-session-token", token)
			fmt.Fprint(res, `{"userid": "1234abc", "username": "billy", "emails": ["billy@1234.abc"]}`)
		default:
			t.Errorf("Unknown path[%s]", req.URL.Path)
		}
	}))
	defer srvr.Close()

	shorelineClient := NewShorelineClientBuilder().
		WithHostGetter(disc.NewStaticHostGetterFromString(srvr.URL)).
		WithName("test").
		WithSecret("howdy ho, neighbor joe").
		Build()

	err := shorelineClient.Start()
	if err != nil {
		t.Errorf("Failed start with error[%v]", err)
	}
	defer shorelineClient.Close()

	ud, tok, err := shorelineClient.Login("billy", "howdy")
	if err != nil {
		t.Errorf("Error on login[%v]", err)
	}
	if tok != token {
		t.Errorf("Unexpected token[%s]", tok)
	}
	if ud.UserID != "1234abc" || ud.Username != "billy" || len(ud.Emails) != 1 || ud.Emails[0] != "billy@1234.abc" {
		t.Errorf("Bad userData object[%+v]", ud)
	}
}

func TestSignup(t *testing.T) {
	srvr := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/serverlogin":
			res.Header().Set("x-tidepool-session-token", token)
		case "/user":
			res.WriteHeader(http.StatusCreated)
			fmt.Fprint(res, `{"userid": "1234abc", "username": "new me", "emails": ["new.me@1234.abc"]}`)
		default:
			t.Errorf("Unknown path[%s]", req.URL.Path)
		}
	}))
	defer srvr.Close()

	client := NewShorelineClientBuilder().
		WithHostGetter(disc.NewStaticHostGetterFromString(srvr.URL)).
		WithName("test").
		WithSecret("howdy ho, neighbor joe").
		Build()

	err := client.Start()
	if err != nil {
		t.Errorf("Failed start with error[%v]", err)
	}
	defer client.Close()

	ud, err := client.Signup("new me", "howdy", "new.me@1234.abc")
	if err != nil {
		t.Errorf("Error on signup [%s]", err.Error())
	}
	if ud.UserID != "1234abc" || ud.Username != "new me" || len(ud.Emails) != 1 || ud.Emails[0] != "new.me@1234.abc" {
		t.Errorf("Bad userData object[%+v]", ud)
	}

}
