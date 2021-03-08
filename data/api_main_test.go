package data

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gorilla/mux"
	"github.com/tidepool-org/go-common/clients"
	"github.com/tidepool-org/go-common/clients/opa"
	"github.com/tidepool-org/go-common/clients/shoreline"
	"github.com/tidepool-org/go-common/clients/status"
	"github.com/tidepool-org/go-common/clients/version"
	"github.com/tidepool-org/tide-whisperer/auth"
	"github.com/tidepool-org/tide-whisperer/store"
)

var (
	schemaVersions = store.SchemaVersion{
		Maximum: 99,
		Minimum: 1,
	}
	serverToken               = "token"
	storage                   = store.NewMockStoreClient()
	mockShoreline             = shoreline.NewMock("token")
	mockAuth                  = auth.NewMock()
	perms                     = clients.Permissions{"view": clients.Allowed, "root": clients.Allowed}
	mockPerms                 = opa.NewMock()
	tidewhisperer             = InitApi(storage, mockShoreline, mockAuth, mockPerms, schemaVersions)
	defaultGetDataURLVars     = map[string]string{"userID": "patient"}
	defaultGetDataStoreParams = getDataStoreDefaultParams()
	rtr                       = mux.NewRouter()
)

// Utility function to reset all mocks to default value
func resetMocks() {
	mockShoreline.UserID = "patient"
	mockShoreline.Unauthorized = false
	mockShoreline.IsServer = false

	auth := mockPerms.GetMockedAuth(true, map[string]interface{}{}, "tidewhisperer-get")
	mockPerms.SetMockOpaAuth("/patient", &auth, nil)
	auth2 := mockPerms.GetMockedAuth(true, map[string]interface{}{}, "tidewhisperer-compute")
	mockPerms.SetMockOpaAuth("/compute/tir", &auth2, nil)
	storage.DeviceData = make([]string, 1)
	storage.DeviceData[0] = `{"type":"A","value":"B"}`

	storage.TimeInRangeData = make([]string, 1)
	storage.TimeInRangeData[0] = `{"userId":"patient","lastCbgTime":"2020-04-15T08:00:00Z"}`
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
	checkResponseError(http.StatusInternalServerError, "invalid_parameters",
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
		request.Header.Set("x-tidepool-session-token", token)
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

	request, response := getStatusPrepareRequest()
	tidewhisperer.GetStatus(response, request)

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
	storage.EnablePingError()

	request, response := getStatusPrepareRequest()
	tidewhisperer.GetStatus(response, request)

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
	tidewhisperer.Get501(response, request)
	if response.Code != http.StatusNotImplemented {
		t.Fatalf("Resp given [%d] expected [%d] ", response.Code, http.StatusNotImplemented)
	}
}
