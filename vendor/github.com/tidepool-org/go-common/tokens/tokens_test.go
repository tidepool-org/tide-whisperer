package tokens

import (
	"net/http"
	"testing"
)

const testToken = "eyJ0eXAiOiJKV1QiLCJhbGciOiJS.iIjoiYXV0aDB8ODIxNjU0NDYz.sCOkfBAzP73Mrkk2pY1-s"

func TestGetHeaderTokenForBearer(t *testing.T) {
	request := &http.Request{Header: http.Header{}}
	request.Header.Set("Authorization", "Bearer "+testToken)

	if IsBearerToken(request) == false {
		t.Log("expected be a bearer token")
		t.Fail()
	}

	bearerToken := GetHeaderToken(request)
	if bearerToken != testToken {
		t.Logf("expected '%s' found '%s'", testToken, bearerToken)
		t.Fail()
	}
}

func TestGetHeaderTokenForSessionToken(t *testing.T) {
	request := &http.Request{Header: http.Header{}}
	request.Header.Set(TidepoolSessionTokenName, testToken)

	if IsSessionToken(request) == false {
		t.Log("expected to be a session token")
		t.Fail()
	}

	bearerToken := GetHeaderToken(request)
	if bearerToken != testToken {
		t.Logf("expected '%s' found '%s'", testToken, bearerToken)
		t.Fail()
	}
}

func TestGetHeaderTokenForNoToken(t *testing.T) {
	request := &http.Request{Header: http.Header{}}
	bearerToken := GetHeaderToken(request)
	if IsBearerToken(request) == true {
		t.Log("expected NOT be a bearer token")
		t.Fail()
	}
	if IsSessionToken(request) == true {
		t.Log("expected NOT be a session token")
		t.Fail()
	}
	if bearerToken != "" {
		t.Logf("expected '%s' found '%s'", "", bearerToken)
		t.Fail()
	}
}

func TestGetHeaderTokenBearerIsDefault(t *testing.T) {
	request := &http.Request{Header: http.Header{}}
	request.Header.Set("Authorization", "Bearer "+testToken)
	request.Header.Set(TidepoolSessionTokenName, "another.token.here")

	if IsBearerToken(request) == false {
		t.Log("expected to be a bearer token")
		t.Fail()
	}

	bearerToken := GetHeaderToken(request)
	if bearerToken != testToken {
		t.Logf("expected '%s' found '%s'", testToken, bearerToken)
		t.Fail()
	}
}

func TestTidepoolSessionTokenName(t *testing.T) {
	if TidepoolSessionTokenName != "x-tidepool-session-token" {
		t.Logf("expected '%s' found '%s'", "x-tidepool-session-token", TidepoolSessionTokenName)
		t.Fail()
	}
}

func TestTidepoolInternalScope(t *testing.T) {
	if TidepoolInternalScope != "tidepool:internal" {
		t.Logf("expected '%s' found '%s'", "tidepool:internal", TidepoolInternalScope)
		t.Fail()
	}
}

func TestTidepoolPublicScope(t *testing.T) {
	if TidepoolPublicScope != "tidepool:public" {
		t.Logf("expected '%s' found '%s'", "tidepool:public", TidepoolPublicScope)
		t.Fail()
	}
}
