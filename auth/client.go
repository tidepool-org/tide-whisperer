package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// Config holds the configuration for the Auth Client
type Config struct {
	Address       string `json:"address"`
	ServiceSecret string `json:"serviceSecret"`
	UserAgent     string `json:"userAgent"`
}

// ClientInterface interface that we will implement and mock
type ClientInterface interface {
	GetRestrictedToken(ctx context.Context, id string) (*RestrictedToken, error)
}

// Client holds the state of the Auth Client
type Client struct {
	config     *Config
	httpClient *http.Client
}

// NewClient creates a new Auth Client
func NewClient(config *Config, httpClient *http.Client) (*Client, error) {
	if config == nil {
		return nil, errors.New("config is missing")
	}
	if httpClient == nil {
		return nil, errors.New("http client is missing")
	}

	return &Client{
		config:     config,
		httpClient: httpClient,
	}, nil
}

// GetRestrictedToken fetches a restricted token from the `auth` service
func (c *Client) GetRestrictedToken(ctx context.Context, id string) (*RestrictedToken, error) {
	if ctx == nil {
		return nil, errors.New("context is missing")
	}
	if id == "" {
		return nil, errors.New("id is missing")
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/v1/restricted_tokens/%s", c.config.Address, id), nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	req.Header.Add("X-Tidepool-Service-Secret", c.config.ServiceSecret)
	req.Header.Add("User-Agent", c.config.UserAgent)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		io.Copy(ioutil.Discard, res.Body)
		res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected status code")
	}

	restrictedToken := &RestrictedToken{}
	if err = json.NewDecoder(res.Body).Decode(restrictedToken); err != nil {
		return nil, err
	}

	return restrictedToken, nil
}
