package clients

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type gatekeeperClient struct {
	httpClient    *http.Client  // store a reference to the http client so we can reuse it
	hostGetter    HostGetter    // The getter that provides the host to talk to for the client
	tokenProvider TokenProvider // An object that provides tokens for communicating with gatekeeper
}

type gatekeeperClientBuilder struct {
	httpClient    *http.Client  // store a reference to the http client so we can reuse it
	hostGetter    HostGetter    // The getter that provides the host to talk to for the client
	tokenProvider TokenProvider // An object that provides tokens for communicating with gatekeeper
}

func NewGatekeeperClientBuilder() *gatekeeperClientBuilder {
	return &gatekeeperClientBuilder{}
}

func (b *gatekeeperClientBuilder) WithHttpClient(httpClient *http.Client) *gatekeeperClientBuilder {
	b.httpClient = httpClient
	return b
}

func (b *gatekeeperClientBuilder) WithHostGetter(hostGetter HostGetter) *gatekeeperClientBuilder {
	b.hostGetter = hostGetter
	return b
}

func (b *gatekeeperClientBuilder) WithTokenProvider(tokenProvider TokenProvider) *gatekeeperClientBuilder {
	b.tokenProvider = tokenProvider
	return b
}

func (b *gatekeeperClientBuilder) Build() *gatekeeperClient {
	if b.httpClient == nil {
		panic("gatekeeperClient requires an httpClient to be set")
	}
	if b.hostGetter == nil {
		panic("gatekeeperClient requires a hostGetter to be set")
	}
	if b.tokenProvider == nil {
		panic("gatekeeperClient requires a tokenProvider to be set")
	}
	return &gatekeeperClient{
		httpClient:    b.httpClient,
		hostGetter:    b.hostGetter,
		tokenProvider: b.tokenProvider,
	}
}

type Permissions map[string]interface{}

func (client *gatekeeperClient) UserInGroup(userID, groupID string) (map[string]Permissions, error) {
	host := client.hostGetter.HostGet()
	host.Path += fmt.Sprintf("private/%s/%s", groupID, userID)

	req, _ := http.NewRequest("GET", host.String(), nil)
	req.Header.Add("x-tidepool-session-token", client.tokenProvider.TokenProvide())

	log.Println(req)
	res, err := client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, &StatusError{NewStatusf(res.StatusCode, "Unknown response code from service[%s]", req.URL)}
	}

	retVal := make(map[string]Permissions)
	if err := json.NewDecoder(res.Body).Decode(&retVal); err != nil {
		log.Println(err)
		return nil, &StatusError{NewStatus(500, "Unable to parse response.")}
	}
	return retVal, nil
}
