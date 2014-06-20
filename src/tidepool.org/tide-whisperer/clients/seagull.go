package clients

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type seagullClient struct {
	httpClient *http.Client // store a reference to the http client so we can reuse it
	hostGetter HostGetter   // The getter that provides the host to talk to for the client
}

type seagullClientBuilder struct {
	httpClient *http.Client
	hostGetter HostGetter
}

func NewSeagullClientBuilder() *seagullClientBuilder {
	return &seagullClientBuilder{}
}

func (b *seagullClientBuilder) WithHttpClient(httpClient *http.Client) *seagullClientBuilder {
	b.httpClient = httpClient
	return b
}

func (b *seagullClientBuilder) WithHostGetter(hostGetter HostGetter) *seagullClientBuilder {
	b.hostGetter = hostGetter
	return b
}

func (b *seagullClientBuilder) Build() *seagullClient {
	if b.httpClient == nil {
		panic("seagullClient requires an httpClient to be set")
	}
	if b.hostGetter == nil {
		panic("seagullClient requires a hostGetter to be set")
	}
	return &seagullClient{
		httpClient: b.httpClient,
		hostGetter: b.hostGetter,
	}
}

type PrivatePair struct {
	ID    string
	Value string
}

func (client *seagullClient) GetPrivatePair(userID, hashName, token string) *PrivatePair {
	host := client.hostGetter.HostGet()
	host.Path += fmt.Sprintf("%s/private/%s", userID, hashName)

	req, _ := http.NewRequest("GET", host.String(), nil)
	req.Header.Add("x-tidepool-session-token", token)

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
