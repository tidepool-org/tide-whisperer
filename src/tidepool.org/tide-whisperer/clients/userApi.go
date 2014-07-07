// This is a client module to support server-side use of the Tidepool
// service called user-api.
package clients

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"tidepool.org/common/errors"
	"tidepool.org/common/jepson"
	"tidepool.org/tide-whisperer/clients/disc"
	"time"
)

// UserApiClient manages the local data for a client. A client is intended to be shared among multiple
// goroutines so it's OK to treat it as a singleton (and probably a good idea).
type UserApiClient struct {
	httpClient *http.Client         // store a reference to the http client so we can reuse it
	hostGetter disc.HostGetter      // The getter that provides the host to talk to for the client
	config     *UserApiClientConfig // Configuration for the client

	serverToken string         // stores the most recently received server token
	closed      chan chan bool // Channel to communicate that the object has been closed
}

type UserApiClientConfig struct {
	Name                 string          `json:"name"`                 // The name of this server for use in obtaining a server token
	Secret               string          `json:"secret"`               // The secret used along with the name to obtain a server token
	TokenRefreshInterval jepson.Duration `json:"tokenRefreshInterval"` // The amount of time between refreshes of the server token
}

// UserData is the data structure returned from a successful Login query.
type UserData struct {
	UserID   string   // the tidepool-assigned user ID
	UserName string   // the user-assigned name for the login (usually an email address)
	Emails   []string // the array of email addresses associated with this account
}

// TokenData is the data structure returned from a successful CheckToken query.
type TokenData struct {
	UserID   string // the UserID stored in the token
	IsServer bool   // true or false depending on whether the token was a servertoken
}

// NewApiClient constructs an api client object.
func NewApiClient(hostGetter disc.HostGetter, config *UserApiClientConfig, httpClient *http.Client) *UserApiClient {
	return &UserApiClient{
		hostGetter: hostGetter,
		config:     config,
		httpClient: httpClient,
	}
}

func NewUserApiConfig(name, secret string, tokenRefreshInterval jepson.Duration) *UserApiClientConfig {
	return &UserApiClientConfig{
		Name:                 name,
		Secret:               secret,
		TokenRefreshInterval: tokenRefreshInterval,
	}
}

// Start starts the client and makes it ready for us.  This must be done before using any of the functionality
// that requires a server token
func (client *UserApiClient) Start() error {
	if err := client.serverLogin(); err != nil {
		return err
	}

	go func() {
		for {
			timer := time.After(time.Duration(client.config.TokenRefreshInterval) * time.Millisecond)
			select {
			case twoWay := <-client.closed:
				twoWay <- true
				break
			case <-timer:
				if err := client.serverLogin(); err != nil {
					log.Print("Error when refreshing server login", err)
				}
			}
		}
	}()
	return nil
}

func (client *UserApiClient) Close() {
	twoWay := make(chan bool)
	client.closed <- twoWay
	<-twoWay

	client.serverToken = ""
}

// serverLogin issues a request to the server for a login, using the stored
// secret that was passed in on the creation of the client object. If
// successful, it stores the returned token in ServerToken.
func (client *UserApiClient) serverLogin() error {
	host := client.getHost()
	if host == nil {
		return errors.New("No known user-api hosts.")
	}

	host.Path += "/serverlogin"

	req, _ := http.NewRequest("POST", host.String(), nil)
	req.Header.Add("x-tidepool-server-name", client.config.Name)
	req.Header.Add("x-tidepool-server-secret", client.config.Secret)

	res, err := client.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "Failure to obtain a server token")
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return &StatusError{NewStatusf(res.StatusCode, "Unknown response code from service[%s]", req.URL)}
	}

	client.serverToken = res.Header.Get("x-tidepool-session-token")
	return nil
}

func extractUserData(r io.Reader) (*UserData, error) {
	var ud UserData
	if err := json.NewDecoder(r).Decode(&ud); err != nil {
		return nil, err
	}
	return &ud, nil
}

// Login logs in a user with a username and password. Returns a UserData object if successful
// and also stores the returned login token into ClientToken.
func (client *UserApiClient) Login(username, password string) (*UserData, string, error) {
	host := client.getHost()
	if host == nil {
		return nil, "", errors.New("No known user-api hosts.")
	}

	host.Path += "/login"

	req, _ := http.NewRequest("POST", host.String(), nil)
	req.SetBasicAuth(username, password)

	res, err := client.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case 200:
		ud, err := extractUserData(res.Body)
		if err != nil {
			return nil, "", err
		}

		return ud, res.Header.Get("x-tidepool-session-token"), nil
	case 404:
		return nil, "", nil
	default:
		return nil, "", &StatusError{NewStatusf(res.StatusCode, "Unknown response code from service[%s]", req.URL)}
	}
}

// CheckToken tests a token with the user-api to make sure it's current;
// if so, it returns the data encoded in the token.
func (client *UserApiClient) CheckToken(token string) *TokenData {
	host := client.getHost()
	if host == nil {
		return nil
	}

	host.Path += "/token/" + token

	req, _ := http.NewRequest("GET", host.String(), nil)
	req.Header.Add("x-tidepool-session-token", client.serverToken)

	res, err := client.httpClient.Do(req)
	if err != nil {
		log.Println("Error checking token", err)
		return nil
	}

	switch res.StatusCode {
	case 200:
		var td TokenData
		if err = json.NewDecoder(res.Body).Decode(&td); err != nil {
			log.Println("Error parsing JSON results", err)
			return nil
		}
		return &td
	case 404:
		return nil
	default:
		log.Printf("Unknown response code[%d] from service[%s]", res.StatusCode, req.URL)
		return nil
	}
}

func (client *UserApiClient) TokenProvide() string {
	return client.serverToken
}

func (client *UserApiClient) getHost() *url.URL {
	if hostArr := client.hostGetter.HostGet(); len(hostArr) > 0 {
		cpy := new(url.URL)
		*cpy = hostArr[0]
		return cpy
	} else {
		return nil
	}
}
