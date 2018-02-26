package tokens

import (
	"net/http"
	"testing"
)

const testToken = "eyJ0eXAiOiJKV1QiLCJhbGciOiJS.iIjoiYXV0aDB8ODIxNjU0NDYz.sCOkfBAzP73Mrkk2pY1-s"

func TestGetBearerToken(t *testing.T) {
	const bearerHeaderValue = "Bearer " + testToken

	request := &http.Request{Header: http.Header{}}
	request.Header.Set(AuthorizationHeaderKey, bearerHeaderValue)

	bearerToken := GetBearerToken(request)
	if bearerToken == "" {
		t.Logf("expected '%s' found '%s'", bearerHeaderValue, bearerToken)
		t.Fail()
	}
	if bearerToken != bearerHeaderValue {
		t.Logf("expected '%s' found '%s'", bearerHeaderValue, bearerToken)
		t.Fail()
	}
}

func TestGetServerSecret(t *testing.T) {
	request := &http.Request{Header: http.Header{}}
	request.Header.Set(TidepoolLegacyServiceSecretHeaderKey, testToken)

	serverToken := GetServerSecret(request)
	if serverToken == "" {
		t.Logf("expected '%s' found '%s'", testToken, serverToken)
		t.Fail()
	}
	if serverToken != testToken {
		t.Logf("expected '%s' found '%s'", testToken, serverToken)
		t.Fail()
	}
}

func TestTidepoolLegacyServiceSecretHeaderKey(t *testing.T) {
	if TidepoolLegacyServiceSecretHeaderKey != "X-Tidepool-Legacy-Service-Secret" {
		t.Logf("expected '%s' found '%s'", "X-Tidepool-Legacy-Service-Secret", TidepoolLegacyServiceSecretHeaderKey)
		t.Fail()
	}
}

func TestTidepoolSessionTokenName(t *testing.T) {
	if TidepoolSessionTokenName != "X-Tidepool-Session-Token" {
		t.Logf("expected '%s' found '%s'", "X-Tidepool-Session-Token", TidepoolSessionTokenName)
		t.Fail()
	}
}

func TestAuthorizationHeaderKey(t *testing.T) {
	if AuthorizationHeaderKey != "Authorization" {
		t.Logf("expected '%s' found '%s'", "Authorization", AuthorizationHeaderKey)
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
