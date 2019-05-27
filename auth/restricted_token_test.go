package auth_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/tidepool-org/tide-whisperer/auth"
)

func testRestrictedTokenSetup(url string) (*http.Request, *auth.RestrictedToken) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	restrictedToken := &auth.RestrictedToken{
		Paths:          &[]string{"/alfa/bravo", "/charlie/delta"},
		ExpirationTime: time.Now().Add(time.Hour),
	}
	return req, restrictedToken
}

func Test_RestrictedToken_Authenticates_MissingRequest(t *testing.T) {
	req, restrictedToken := testRestrictedTokenSetup("http://localhost/charlie/delta")
	req = nil
	if restrictedToken.Authenticates(req) {
		t.Error("RestrictedToken.Authenticates fails to not authenticate missing request")
	}
}

func Test_RestrictedToken_Authenticates_MissingRequestURL(t *testing.T) {
	req, restrictedToken := testRestrictedTokenSetup("http://localhost/charlie/delta")
	req.URL = nil
	if restrictedToken.Authenticates(req) {
		t.Error("RestrictedToken.Authenticates fails to not authenticate missing request url")
	}
}

func Test_RestrictedToken_Authenticates_ExpiredRestrictedToken(t *testing.T) {
	req, restrictedToken := testRestrictedTokenSetup("http://localhost/charlie/delta")
	restrictedToken.ExpirationTime = time.Now().Add(-time.Hour)
	if restrictedToken.Authenticates(req) {
		t.Error("RestrictedToken.Authenticates fails to not authenticate expired restricted token")
	}
}

func Test_RestrictedToken_Authenticates_PathsEmpty(t *testing.T) {
	req, restrictedToken := testRestrictedTokenSetup("http://localhost/charlie/delta")
	restrictedToken.Paths = &[]string{}
	if restrictedToken.Authenticates(req) {
		t.Error("RestrictedToken.Authenticates fails to not authenticate paths empty")
	}
}

func Test_RestrictedToken_Authenticates_PathNotFound(t *testing.T) {
	req, restrictedToken := testRestrictedTokenSetup("http://localhost/yankee/zulu")
	if restrictedToken.Authenticates(req) {
		t.Error("RestrictedToken.Authenticates fails to not authenticate path not found")
	}
}

func Test_RestrictedToken_Authenticates_PartialPrefix(t *testing.T) {
	req, restrictedToken := testRestrictedTokenSetup("http://localhost/charlie/deltaecho/foxtrot")
	if restrictedToken.Authenticates(req) {
		t.Error("RestrictedToken.Authenticates fails to not authenticate partial prefix")
	}
}

func Test_RestrictedToken_Authenticates_Valid(t *testing.T) {
	req, restrictedToken := testRestrictedTokenSetup("http://localhost/charlie/delta")
	if !restrictedToken.Authenticates(req) {
		t.Error("RestrictedToken.Authenticates fails to authenticate valid")
	}
}

func Test_RestrictedToken_Authenticates_ValidTrailingSlash(t *testing.T) {
	req, restrictedToken := testRestrictedTokenSetup("http://localhost/charlie/delta/")
	if !restrictedToken.Authenticates(req) {
		t.Error("RestrictedToken.Authenticates fails to authenticate valid trailing slash")
	}
}

func Test_RestrictedToken_Authenticates_ValidPrefix(t *testing.T) {
	req, restrictedToken := testRestrictedTokenSetup("http://localhost/charlie/delta/echo/foxtrot")
	if !restrictedToken.Authenticates(req) {
		t.Error("RestrictedToken.Authenticates fails to authenticate valid prefix")
	}
}

func Test_RestrictedToken_Authenticates_ValidWithoutPaths(t *testing.T) {
	req, restrictedToken := testRestrictedTokenSetup("http://localhost/yankee/zulu")
	restrictedToken.Paths = nil
	if !restrictedToken.Authenticates(req) {
		t.Error("RestrictedToken.Authenticates fails to authenticate valid without paths")
	}
}
