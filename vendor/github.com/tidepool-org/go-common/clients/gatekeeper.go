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
	"github.com/tidepool-org/go-common/tokens"
)

type (
	//Inteface so that we can mock gatekeeperClient for tests
	Gatekeeper interface {
		//userID  -- the Tidepool-assigned userID
		//groupID  -- the Tidepool-assigned groupID
		//
		// returns the Permissions
		UserInGroup(userID string, groupID string) (Permissions, error)

		//groupID  -- the Tidepool-assigned groupID
		//
		// returns the map of user id to Permissions
		UsersInGroup(groupID string) (UsersPermissions, error)

		//userID  -- the Tidepool-assigned userID
		//groupID  -- the Tidepool-assigned groupID
		//permissions -- the permisson we want to give the user for the group
		SetPermissions(userID string, groupID string, permissions Permissions) (Permissions, error)
	}

	gatekeeperClient struct {
		httpClient   *http.Client    // store a reference to the http client so we can reuse it
		hostGetter   disc.HostGetter // The getter that provides the host to talk to for the client
		serverSecret string          // An object that provides tokens for communicating with gatekeeper
	}

	gatekeeperClientBuilder struct {
		httpClient   *http.Client    // store a reference to the http client so we can reuse it
		hostGetter   disc.HostGetter // The getter that provides the host to talk to for the client
		serverSecret string          // An object that provides tokens for communicating with gatekeeper
	}

	Permission       map[string]interface{}
	Permissions      map[string]Permission
	UsersPermissions map[string]Permissions
)

var (
	Allowed Permission = Permission{}
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

func (b *gatekeeperClientBuilder) WithSecret(serverSecret string) *gatekeeperClientBuilder {
	b.serverSecret = serverSecret
	return b
}

func (b *gatekeeperClientBuilder) Build() *gatekeeperClient {
	if b.hostGetter == nil {
		panic("gatekeeperClient requires a hostGetter to be set")
	}
	if b.serverSecret == "" {
		panic("gatekeeperClient requires a serverSecret to be set")
	}

	if b.httpClient == nil {
		b.httpClient = http.DefaultClient
	}

	return &gatekeeperClient{
		httpClient:   b.httpClient,
		hostGetter:   b.hostGetter,
		serverSecret: b.serverSecret,
	}
}

func (client *gatekeeperClient) UserInGroup(userID, groupID string) (Permissions, error) {
	host := client.getHost()
	if host == nil {
		return nil, errors.New("No known gatekeeper hosts")
	}
	host.Path += fmt.Sprintf("access/%s/%s", groupID, userID)

	req, _ := http.NewRequest("GET", host.String(), nil)
	req.Header.Add(tokens.TidepoolLegacyServiceSecretHeaderKey, client.serverSecret)

	res, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		retVal := make(Permissions)
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

func (client *gatekeeperClient) UsersInGroup(groupID string) (UsersPermissions, error) {
	host := client.getHost()
	if host == nil {
		return nil, errors.New("No known gatekeeper hosts")
	}
	host.Path += fmt.Sprintf("access/%s", groupID)

	req, _ := http.NewRequest("GET", host.String(), nil)
	req.Header.Add(tokens.TidepoolLegacyServiceSecretHeaderKey, client.serverSecret)

	res, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		retVal := make(UsersPermissions)
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

func (client *gatekeeperClient) SetPermissions(userID, groupID string, permissions Permissions) (Permissions, error) {
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
		req.Header.Add(tokens.TidepoolLegacyServiceSecretHeaderKey, client.serverSecret)

		res, err := client.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		if res.StatusCode == 200 {
			retVal := make(Permissions)
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
