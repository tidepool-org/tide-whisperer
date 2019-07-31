package auth_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tidepool-org/tide-whisperer/auth"
)

func testClientSetup(address string) (*auth.Config, *http.Client) {
	config := &auth.Config{
		Address:       address,
		ServiceSecret: "ThisIsASecret!",
		UserAgent:     "Test-Agent",
	}
	httpClient := &http.Client{}
	return config, httpClient
}

func testServerClientSetup(handlerFunc http.HandlerFunc) (*httptest.Server, *auth.Client, context.Context) {
	server := httptest.NewServer(handlerFunc)
	client, _ := auth.NewClient(testClientSetup(server.URL))
	ctx := context.Background()
	return server, client, ctx
}

func Test_NewClient_ConfigMissing(t *testing.T) {
	config, httpClient := testClientSetup("http://localhost/alfa/bravo")
	config = nil
	client, err := auth.NewClient(config, httpClient)
	if err == nil || err.Error() != "config is missing" || client != nil {
		t.Error(err, "NewClient fails to return expected error and no client config missing")
	}
}

func Test_NewClient_HttpClientMissing(t *testing.T) {
	config, httpClient := testClientSetup("http://localhost/alfa/bravo")
	httpClient = nil
	client, err := auth.NewClient(config, httpClient)
	if err == nil || err.Error() != "http client is missing" || client != nil {
		t.Error(err, "NewClient fails to return expected error and no client http client missing")
	}
}

func Test_NewClient_Valid(t *testing.T) {
	config, httpClient := testClientSetup("http://localhost/alfa/bravo")
	client, err := auth.NewClient(config, httpClient)
	if err != nil || client == nil {
		t.Error(err, "NewClient fails to return no error and client")
	}
}

func Test_Client_GetRestrictedToken_ContextMissing(t *testing.T) {
	id := "1234567890"
	server, client, ctx := testServerClientSetup(func(res http.ResponseWriter, req *http.Request) {})
	defer server.Close()
	ctx = nil
	restrictedToken, err := client.GetRestrictedToken(ctx, id)
	if err == nil || err.Error() != "context is missing" || restrictedToken != nil {
		t.Error(err, "Client.GetRestrictedToken fails to return expected error and no restricted token context missing")
	}
}

func Test_Client_GetRestrictedToken_IDMissing(t *testing.T) {
	id := ""
	server, client, ctx := testServerClientSetup(func(res http.ResponseWriter, req *http.Request) {})
	defer server.Close()
	restrictedToken, err := client.GetRestrictedToken(ctx, id)
	if err == nil || err.Error() != "id is missing" || restrictedToken != nil {
		t.Error(err, "Client.GetRestrictedToken fails to return expected error and no restricted token id missing")
	}
}

func Test_Client_GetRestrictedToken_UnexpectedStatusCode(t *testing.T) {
	id := "1234567890"
	server, client, ctx := testServerClientSetup(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusBadRequest)
	})
	defer server.Close()
	restrictedToken, err := client.GetRestrictedToken(ctx, id)
	if err == nil || err.Error() != "unexpected status code" || restrictedToken != nil {
		t.Error(err, "Client.GetRestrictedToken fails to return expected error and no restricted token unexpected status code")
	}
}

func Test_Client_GetRestrictedToken_MalformedJSON(t *testing.T) {
	id := "1234567890"
	server, client, ctx := testServerClientSetup(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		res.Write([]byte("{"))
	})
	defer server.Close()
	restrictedToken, err := client.GetRestrictedToken(ctx, id)
	if err == nil || err.Error() != "unexpected EOF" || restrictedToken != nil {
		t.Error(err, "Client.GetRestrictedToken fails to return expected error and no restricted token malformed JSON")
	}
}

func Test_Client_GetRestrictedToken_Valid(t *testing.T) {
	id := "1234567890"
	userID := "9876543210"
	server, client, ctx := testServerClientSetup(func(res http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			t.Error("Client.GetRestrictedToken fails to use expected method")
		}
		if req.URL.Path != fmt.Sprintf("/v1/restricted_tokens/%s", id) {
			t.Error("Client.GetRestrictedToken fails to use expected path")
		}
		if req.Header.Get("X-Tidepool-Service-Secret") != "ThisIsASecret!" {
			t.Error("Client.GetRestrictedToken fails to add expected header with service secret")
		}
		if req.Header.Get("User-Agent") != "Test-Agent" {
			t.Error("Client.GetRestrictedToken fails to add expected header with user agent")
		}
		res.WriteHeader(http.StatusOK)
		res.Write([]byte(fmt.Sprintf(`{"id": "%s", "userId": "%s"}`, id, userID)))
	})
	defer server.Close()
	restrictedToken, err := client.GetRestrictedToken(ctx, id)
	if err != nil || restrictedToken == nil {
		t.Error(err, "Client.GetRestrictedToken fails to return no error and restricted token")
	}
	if restrictedToken.UserID != userID {
		t.Error("Client.GetRestrictedToken fails to return expected restricted token")
	}
}
