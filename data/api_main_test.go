package data

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/gorilla/mux"
	"github.com/mdblp/go-common/clients/auth"
	"github.com/mdblp/shoreline/token"
	twV2Client "github.com/mdblp/tide-whisperer-v2/v2/client/tidewhisperer"
	"github.com/stretchr/testify/mock"
	"github.com/tidepool-org/go-common/clients/opa"
	"github.com/tidepool-org/go-common/clients/status"
	"github.com/tidepool-org/go-common/clients/version"
	"github.com/tidepool-org/tide-whisperer/store"
)

var (
	schemaVersions = store.SchemaVersion{
		Maximum: 99,
		Minimum: 1,
	}
	logger        = log.New(os.Stdout, "api-test", log.LstdFlags|log.Lshortfile)
	storage       = store.NewMockStoreClient()
	mockAuth      = auth.NewMock()
	mockPerms     = opa.NewMock()
	mockTideV2    = twV2Client.NewMock()
	tidewhisperer = InitAPI(storage, mockAuth, mockPerms, schemaVersions, logger, mockTideV2)
	rtr           = mux.NewRouter()
)

// Utility function to reset all mocks to default value
func resetMocks() {
	mockAuth.ExpectedCalls = nil
	auth := mockPerms.GetMockedAuth(true, map[string]interface{}{}, "tidewhisperer-get")
	mockPerms.SetMockOpaAuth("/patient", &auth, nil)
	auth2 := mockPerms.GetMockedAuth(true, map[string]interface{}{}, "tidewhisperer-compute")
	mockPerms.SetMockOpaAuth("/compute/tir", &auth2, nil)
}

// Utility function to check authorized responses
func checkAuthorized(response *httptest.ResponseRecorder, t *testing.T) {
	if response.Code != http.StatusOK {
		t.Fatalf("Resp given [%d] expected [%d] ", response.Code, http.StatusOK)
	}
}

// Generic Utility function to check error responses
func checkResponseError(status int, code string, message string, response *httptest.ResponseRecorder, t *testing.T) {
	if response.Code != status {
		t.Fatalf("Resp given [%d] expected [%d] ", response.Code, status)
	}
	body, _ := ioutil.ReadAll(response.Body)
	var dataBody detailedError
	json.Unmarshal([]byte(string(body)), &dataBody)
	if dataBody.Status != status {
		t.Fatalf("Body status given [%d] expected [%d] ", dataBody.Status, status)
	}
	if dataBody.Code != code {
		t.Fatalf("Body code given [%s] expected [%s] ", dataBody.Code, code)
	}
	if dataBody.Message != message {
		t.Fatalf("Body message given [%s] expected [%s] ", dataBody.Message, message)
	}
}

// Utility function to check unauthorized responses
func checkUnAuthorized(response *httptest.ResponseRecorder, t *testing.T) {
	checkResponseError(http.StatusForbidden, "data_cant_view",
		"user is not authorized to view data",
		response, t)
}

// Utility function to check invalid parameters error responses
func checkInvalidParams(response *httptest.ResponseRecorder, t *testing.T) {
	checkResponseError(http.StatusBadRequest, "invalid_parameters",
		"one or more parameters are invalid",
		response, t)
}

// Utility function to parse response body as an array
func parseArrayResponse(response *httptest.ResponseRecorder) []map[string]interface{} {
	body, _ := ioutil.ReadAll(response.Body)
	var dataBody []map[string]interface{}
	json.Unmarshal([]byte(string(body)), &dataBody)
	return dataBody
}
func prepareGetTestRequest(route string, token string, urlParams map[string]string) (*http.Request, *httptest.ResponseRecorder) {
	tidewhisperer.SetHandlers("", rtr)
	request, _ := http.NewRequest("GET", route, nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if len(urlParams) > 0 {
		q := request.URL.Query()
		for key, element := range urlParams {
			q.Add(key, element)
		}
		request.URL.RawQuery = q.Encode()
	}
	response := httptest.NewRecorder()
	return request, response
}

// Utility function to prepare request on GetStatus route
func getStatusPrepareRequest() (*http.Request, *httptest.ResponseRecorder) {
	version.ReleaseNumber = "1.2.3"
	version.FullCommit = "e0c73b95646559e9a3696d41711e918398d557fb"
	tidewhisperer.SetHandlers("", rtr)
	request, _ := http.NewRequest("GET", "/status", nil)
	response := httptest.NewRecorder()
	return request, response
}

// Utility function to prepare resposnes on GetStatus route
func getStatusParseResponse(response *httptest.ResponseRecorder) status.ApiStatus {
	body, _ := ioutil.ReadAll(response.Body)
	// Checking body content
	dataBody := status.ApiStatus{}
	json.Unmarshal([]byte(string(body)), &dataBody)
	return dataBody
}

// Testing GetStatus route
// TestGetStatus_StatusOk calling GetStatus route with an enabled storage
func TestGetStatus_StatusOk(t *testing.T) {
	resetMocks()
	mockAuth.On("Authenticate", mock.Anything).Return(&token.TokenData{UserId: "patient", IsServer: false})
	request, response := getStatusPrepareRequest()
	tidewhisperer.getStatus(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("Resp given [%d] expected [%d] ", response.Code, http.StatusOK)
	}
	// Checking body content
	dataBody := getStatusParseResponse(response)
	expectedStatus := status.ApiStatus{
		Status:  status.Status{Code: 200, Reason: "OK"},
		Version: version.ReleaseNumber + "+" + version.FullCommit,
	}
	if !reflect.DeepEqual(dataBody, expectedStatus) {
		t.Fatalf("store.GetStatus given [%v] expected [%v] ", dataBody, expectedStatus)
	}

}

// TestGetStatus_StatusKo calling GetStatus route with a disabled storage
func TestGetStatus_StatusKo(t *testing.T) {
	resetMocks()
	mockAuth.On("Authenticate", mock.Anything).Return(&token.TokenData{UserId: "patient", IsServer: false})
	storage.EnablePingError()

	request, response := getStatusPrepareRequest()
	tidewhisperer.getStatus(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("Resp given [%d] expected [%d] ", response.Code, http.StatusInternalServerError)
	}
	// Checking body content
	dataBody := getStatusParseResponse(response)
	expectedStatus := status.ApiStatus{
		Status:  status.Status{Code: 500, Reason: "Mock Ping Error"},
		Version: version.ReleaseNumber + "+" + version.FullCommit,
	}
	if !reflect.DeepEqual(dataBody, expectedStatus) {
		t.Fatalf("store.GetStatus given [%v] expected [%v] ", dataBody, expectedStatus)
	}
}

// Testing Get501 route
// TestGet501 calling Get501 route to check route is not authorized
func TestGet501(t *testing.T) {
	request, _ := http.NewRequest("GET", "/swagger", nil)
	response := httptest.NewRecorder()
	tidewhisperer.SetHandlers("", rtr)
	tidewhisperer.get501(response, request)
	if response.Code != http.StatusNotImplemented {
		t.Fatalf("Resp given [%d] expected [%d] ", response.Code, http.StatusNotImplemented)
	}
}
