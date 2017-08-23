package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/tidepool-org/go-common/clients/disc"
	"github.com/tidepool-org/go-common/clients/status"
	"github.com/tidepool-org/go-common/errors"
)

type (
	//Inteface so that we can mock gatekeeperClient for tests
	Gatekeeper interface {
		//userID  -- the Tidepool-assigned userID
		//groupID  -- the Tidepool-assigned groupID
		//
		// returns the Permissions
		UserInGroup(userID, groupID string) (map[string]Permissions, error)
		//userID  -- the Tidepool-assigned userID
		//groupID  -- the Tidepool-assigned groupID
		//permissions -- the permisson we want to give the user for the group
		SetPermissions(userID, groupID string, permissions Permissions) (map[string]Permissions, error)
	}

	gatekeeperClient struct {
		httpClient    *http.Client    // store a reference to the http client so we can reuse it
		hostGetter    disc.HostGetter // The getter that provides the host to talk to for the client
		tokenProvider TokenProvider   // An object that provides tokens for communicating with gatekeeper
	}

	gatekeeperClientBuilder struct {
		httpClient    *http.Client    // store a reference to the http client so we can reuse it
		hostGetter    disc.HostGetter // The getter that provides the host to talk to for the client
		tokenProvider TokenProvider   // An object that provides tokens for communicating with gatekeeper
	}

	Permissions map[string]interface{}
)

func NewGatekeeperClientBuilder() *gatekeeperClientBuilder {
	return &gatekeeperClientBuilder{}
}

func (b *gatekeeperClientBuilder) WithHttpClient(httpClient *http.Client) *gatekeeperClientBuilder {
	b.httpClient = httpClient
	return b
}

func (b *gatekeeperClientBuilder) WithHostGetter(hostGetter disc.HostGetter) *gatekeeperClientBuilder {
	b.hostGetter = hostGetter
	return b
}

func (b *gatekeeperClientBuilder) WithTokenProvider(tokenProvider TokenProvider) *gatekeeperClientBuilder {
	b.tokenProvider = tokenProvider
	return b
}

func (b *gatekeeperClientBuilder) Build() *gatekeeperClient {
	if b.hostGetter == nil {
		panic("gatekeeperClient requires a hostGetter to be set")
	}
	if b.tokenProvider == nil {
		panic("gatekeeperClient requires a tokenProvider to be set")
	}

	if b.httpClient == nil {
		b.httpClient = http.DefaultClient
	}

	return &gatekeeperClient{
		httpClient:    b.httpClient,
		hostGetter:    b.hostGetter,
		tokenProvider: b.tokenProvider,
	}
}

func (client *gatekeeperClient) UserInGroup(userID, groupID string) (map[string]Permissions, error) {
	host := client.getHost()
	if host == nil {
		return nil, errors.New("No known gatekeeper hosts")
	}
	host.Path += fmt.Sprintf("access/%s/%s", groupID, userID)

	req, _ := http.NewRequest("GET", host.String(), nil)
	req.Header.Add("x-tidepool-session-token", client.tokenProvider.TokenProvide())

	log.Println(req)
	res, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		retVal := make(map[string]Permissions)
		log.Printf(" [%v]")
		if err := json.NewDecoder(res.Body).Decode(&retVal); err != nil {
			log.Println(err)
			return nil, &status.StatusError{status.NewStatus(500, "UserInGroup Unable to parse response.")}
		}
		return retVal, nil
	} else if res.StatusCode == 404 {
		return nil, nil
	} else {
		return nil, &status.StatusError{status.NewStatusf(res.StatusCode, "Unknown response code from service[%s]", req.URL)}
	}

}

func (client *gatekeeperClient) SetPermissions(userID, groupID string, permissions Permissions) (map[string]Permissions, error) {
	host := client.getHost()
	if host == nil {
		return nil, errors.New("No known gatekeeper hosts")
	}
	host.Path += fmt.Sprintf("access/%s/%s", groupID, userID)

	if jsonPerms, err := json.Marshal(permissions); err != nil {
		log.Println(err)
		return nil, &status.StatusError{status.NewStatusf(http.StatusInternalServerError, "Error marshaling the permissons [%s]", err)}
	} else {
		req, _ := http.NewRequest("POST", host.String(), bytes.NewBuffer(jsonPerms))
		req.Header.Set("content-type", "application/json")
		req.Header.Add("x-tidepool-session-token", client.tokenProvider.TokenProvide())

		res, err := client.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		if res.StatusCode == 200 {

			retVal := make(map[string]Permissions)
			if err := json.NewDecoder(res.Body).Decode(&retVal); err != nil {
				log.Printf("SetPermissions: Unable to parse response: [%s]", err.Error())
				return nil, &status.StatusError{status.NewStatus(500, "SetPermissions: Unable to parse response:")}
			}
			return retVal, nil
		} else if res.StatusCode == 404 {
			return nil, nil
		} else {
			return nil, &status.StatusError{status.NewStatusf(res.StatusCode, "Unknown response code from service[%s]", req.URL)}
		}
	}
}

func (client *gatekeeperClient) getHost() *url.URL {
	if hostArr := client.hostGetter.HostGet(); len(hostArr) > 0 {
		cpy := new(url.URL)
		*cpy = hostArr[0]
		return cpy
	} else {
		return nil
	}
}
