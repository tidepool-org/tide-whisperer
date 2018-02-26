package clients

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/tidepool-org/go-common/tokens"

	"github.com/tidepool-org/go-common/clients/disc"
	"github.com/tidepool-org/go-common/clients/status"
)

type (
	Seagull interface {
		// Retrieves arbitrary collection information from metadata
		//
		// userID -- the Tidepool-assigned userId
		// hashName -- the name of what we are trying to get
		GetPrivatePair(userID string, hashName string) *PrivatePair
		// Retrieves arbitrary collection information from metadata
		//
		// userID -- the Tidepool-assigned userId
		// collectionName -- the collection being retrieved
		// v - the interface to return the value in
		GetCollection(userID string, collectionName string, v interface{}) error
	}

	seagullClient struct {
		httpClient   *http.Client    // store a reference to the http client so we can reuse it
		hostGetter   disc.HostGetter // The getter that provides the host to talk to for the client
		serverSecret string          // An object that provides tokens for communicating with gatekeeper
	}

	seagullClientBuilder struct {
		httpClient   *http.Client
		hostGetter   disc.HostGetter
		serverSecret string
	}

	PrivatePair struct {
		ID    string
		Value string
	}
)

func NewSeagullClientBuilder() *seagullClientBuilder {
	return &seagullClientBuilder{}
}

func (b *seagullClientBuilder) WithHttpClient(httpClient *http.Client) *seagullClientBuilder {
	b.httpClient = httpClient
	return b
}

func (b *seagullClientBuilder) WithHostGetter(hostGetter disc.HostGetter) *seagullClientBuilder {
	b.hostGetter = hostGetter
	return b
}

func (b *seagullClientBuilder) WithSecret(serverSecret string) *seagullClientBuilder {
	b.serverSecret = serverSecret
	return b
}

func (b *seagullClientBuilder) Build() *seagullClient {
	if b.httpClient == nil {
		panic("seagullClient requires an httpClient to be set")
	}
	if b.hostGetter == nil {
		panic("seagullClient requires a hostGetter to be set")
	}
	if b.serverSecret == "" {
		panic("seagullClient requires a serverSecret to be set")
	}
	return &seagullClient{
		httpClient:   b.httpClient,
		hostGetter:   b.hostGetter,
		serverSecret: b.serverSecret,
	}
}

func (client *seagullClient) GetPrivatePair(userID string, hashName string) *PrivatePair {
	host := client.getHost()
	if host == nil {
		return nil
	}
	host.Path += fmt.Sprintf("%s/private/%s", userID, hashName)

	req, _ := http.NewRequest("GET", host.String(), nil)
	req.Header.Add(tokens.TidepoolLegacyServiceSecretHeaderKey, client.serverSecret)

	log.Println(req)
	res, err := client.httpClient.Do(req)
	if err != nil {
		log.Printf("Problem when looking up private pair for userID[%s]. %s", userID, err)
		return nil
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Printf("Unknown response code[%s] from service[%s]", res.StatusCode, req.URL)
		return nil
	}

	var retVal PrivatePair
	if err := json.NewDecoder(res.Body).Decode(&retVal); err != nil {
		log.Println("Error parsing JSON results", err)
		return nil
	}
	return &retVal
}

func (client *seagullClient) GetCollection(userID string, collectionName string, v interface{}) error {
	host := client.getHost()
	if host == nil {
		return nil
	}
	host.Path += fmt.Sprintf("%s/%s", userID, collectionName)

	req, _ := http.NewRequest("GET", host.String(), nil)
	req.Header.Add(tokens.TidepoolLegacyServiceSecretHeaderKey, client.serverSecret)

	log.Println(req)
	res, err := client.httpClient.Do(req)
	if err != nil {
		log.Printf("Problem when looking up collection for userID[%s]. %s", userID, err)
		return err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
			log.Println("Error parsing JSON results", err)
			return err
		}
		return nil
	case http.StatusNotFound:
		log.Printf("No [%s] collection found for [%s]", collectionName, userID)
		return nil
	default:
		return &status.StatusError{status.NewStatusf(res.StatusCode, "Unknown response code from service[%s]", req.URL)}
	}

}

func (client *seagullClient) getHost() *url.URL {
	if hostArr := client.hostGetter.HostGet(); len(hostArr) > 0 {
		cpy := new(url.URL)
		*cpy = hostArr[0]
		return cpy
	}
	return nil
}
