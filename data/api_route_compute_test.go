package data

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/mdblp/shoreline/token"
	"github.com/stretchr/testify/mock"
	"github.com/tidepool-org/tide-whisperer/store"
)

// Testing GetTimeInRange route
// Utility function to prepare request on GetTimeInRange route
func getTimeInRangePrepareRequest(token string, urlParams map[string]string) (*http.Request, *httptest.ResponseRecorder) {
	return prepareGetTestRequest("/compute/tir", token, urlParams)
}

// Utility function to check which params are passed to store.GetTimeInRangeData
func getTimeInRangeCheckStoreParams(expectedParams store.AggParams, t *testing.T) {
	params := storage.GetTimeInRangeDataCall
	if !reflect.DeepEqual(*params.SchemaVersion, *expectedParams.SchemaVersion) {
		t.Fatalf("store.GetDeviceData params.SchemaVersion given [%v] expected [%v] ", *params.SchemaVersion, *expectedParams.SchemaVersion)
	}
	params.SchemaVersion = nil
	expectedParams.SchemaVersion = nil
	if !reflect.DeepEqual(params, expectedParams) {
		t.Fatalf("store.GetDeviceData params given [%v] expected [%v] ", params, expectedParams)
	}
}
func getTimeInRangeStoreDefaultParams() store.AggParams {
	endTime := time.Now()
	startTime := endTime.Add(-(time.Hour * 24))
	return store.AggParams{
		UserIDs: []string{"patient"},
		Date: store.Date{
			Start: startTime.Format(time.RFC3339),
			End:   endTime.Format(time.RFC3339)},
		SchemaVersion: &schemaVersions,
	}
}

// TestGetTimeInRange_NoToken calling GetTimeInRange route without token should be unauthorized
func TestGetTimeInRange_NoToken(t *testing.T) {
	resetMocks()
	mockAuth.On("Authenticate", mock.Anything).Return(nil)
	urlParams := make(map[string]string)
	urlParams["userIds"] = "patient"
	request, response := getTimeInRangePrepareRequest("", urlParams)
	tidewhisperer.GetTimeInRange(response, request)

	checkUnAuthorized(response, t)
}

// TestGetTimeInRange_WrongToken calling GetTimeInRange route with an authorized token should be unauthorized
func TestGetTimeInRange_WrongToken(t *testing.T) {
	resetMocks()
	mockAuth.On("Authenticate", mock.Anything).Return(nil)
	urlParams := make(map[string]string)
	urlParams["userIds"] = "patient"
	request, response := getTimeInRangePrepareRequest("mytoken", urlParams)
	tidewhisperer.GetTimeInRange(response, request)

	checkUnAuthorized(response, t)
}

// TestGetTimeInRange_GoodTokenGuestUserNotInvited calling GetTimeInRange route for a user who didn't invite the user owning the token should be unauthorized
func TestGetTimeInRange_GoodTokenGuestUserNotInvited(t *testing.T) {
	resetMocks()
	mockAuth.On("Authenticate", mock.Anything).Return(&token.TokenData{UserId: "guestUninvited", IsServer: false})
	auth := mockPerms.GetMockedAuth(false, map[string]interface{}{}, "tidewhisperer-compute")
	mockPerms.SetMockOpaAuth("/compute/tir", &auth, nil)

	urlParams := make(map[string]string)
	urlParams["userIds"] = "patient"
	request, response := getTimeInRangePrepareRequest("mytoken", urlParams)
	tidewhisperer.GetTimeInRange(response, request)

	checkUnAuthorized(response, t)
}

// TestGetTimeInRange_GoodTokenGuestUserInvited calling GetTimeInRange route for a user who invited the user owning the token should be authorized
func TestGetTimeInRange_GoodTokenGuestUserInvited(t *testing.T) {
	resetMocks()
	mockAuth.On("Authenticate", mock.Anything).Return(&token.TokenData{UserId: "guest", IsServer: false})
	urlParams := make(map[string]string)
	urlParams["userIds"] = "patient"
	request, response := getTimeInRangePrepareRequest("mytoken", urlParams)
	storeParams := getTimeInRangeStoreDefaultParams()
	tidewhisperer.GetTimeInRange(response, request)

	checkAuthorized(response, t)
	getTimeInRangeCheckStoreParams(storeParams, t)
}

// TestGetTimeInRange_ServerToken calling GetTimeInRange route for any user with a server token should be authorized
func TestGetTimeInRange_ServerToken(t *testing.T) {
	resetMocks()
	mockAuth.On("Authenticate", mock.Anything).Return(&token.TokenData{UserId: "server", IsServer: true})
	auth := mockPerms.GetMockedAuth(false, map[string]interface{}{}, "tidewhisperer-compute")
	mockPerms.SetMockOpaAuth("/compute/tir", &auth, nil)

	urlParams := make(map[string]string)
	urlParams["userIds"] = "patient"
	request, response := getTimeInRangePrepareRequest("mytoken", urlParams)
	storeParams := getTimeInRangeStoreDefaultParams()
	tidewhisperer.GetTimeInRange(response, request)

	checkAuthorized(response, t)
	getTimeInRangeCheckStoreParams(storeParams, t)
}

// TestGetTimeInRange_ValueOutput calling GetTimeInRange route with valid authorization should return the json values from storage
func TestGetTimeInRange_ValueOutput(t *testing.T) {
	resetMocks()
	mockAuth.On("Authenticate", mock.Anything).Return(&token.TokenData{UserId: "patient", IsServer: false})
	urlParams := make(map[string]string)
	urlParams["userIds"] = "patient"
	request, response := getTimeInRangePrepareRequest("mytoken", urlParams)
	storeParams := getTimeInRangeStoreDefaultParams()
	tidewhisperer.GetTimeInRange(response, request)

	checkAuthorized(response, t)
	getTimeInRangeCheckStoreParams(storeParams, t)
	// Checking body content
	dataBody := parseArrayResponse(response)
	if dataBody[0]["userId"] != "patient" {
		t.Fatalf("Body data first element userId [%s] expected [%s] ", dataBody[0]["userId"], "patient")
	}
	if dataBody[0]["lastCbgTime"] != "2020-04-15T08:00:00Z" {
		t.Fatalf("Body data first element lastCbgTime [%s] expected [%s] ", dataBody[0]["lastCbgTime"], "2020-04-15T08:00:00Z")
	}
}
func TestGetTimeInRange_UrlParameters(t *testing.T) {
	resetMocks()
	mockAuth.On("Authenticate", mock.Anything).Return(&token.TokenData{UserId: "patient", IsServer: false})
	// Testing date query parameters
	for _, dateField := range []string{"endDate"} {
		urlParams := make(map[string]string)
		dateValue := "2020-05-11T08:00:00Z"
		urlParams[dateField] = dateValue
		urlParams["userIds"] = "patient"
		request, response := getTimeInRangePrepareRequest("mytoken", urlParams)
		tidewhisperer.GetTimeInRange(response, request)
		checkAuthorized(response, t)
		storeParams := getTimeInRangeStoreDefaultParams()
		switch dateField {
		case "endDate":
			storeParams.Date = store.Date{Start: "2020-05-10T08:00:00Z", End: dateValue}
		}
		getTimeInRangeCheckStoreParams(storeParams, t)

		storage.GetTimeInRangeDataCalled = false
		urlParams[dateField] = "invalid_date"
		urlParams["userIds"] = "patient"
		request, response = getTimeInRangePrepareRequest("mytoken", urlParams)
		tidewhisperer.GetTimeInRange(response, request)
		checkInvalidParams(response, t)
		if storage.GetTimeInRangeDataCalled != false {
			t.Fatalf("store.GetTimeInRange called  [%v] expected [%v] ", storage.GetTimeInRangeDataCalled, false)
		}
	}

	// Testing string array query parameters
	for _, stringArrayField := range []string{"userIds"} {
		urlParams := make(map[string]string)
		arrValue := "p1,p2,p3"
		urlParams[stringArrayField] = arrValue
		request, response := getTimeInRangePrepareRequest("mytoken", urlParams)
		tidewhisperer.GetTimeInRange(response, request)
		checkAuthorized(response, t)
		storeParams := getTimeInRangeStoreDefaultParams()
		switch stringArrayField {
		case "userIds":
			storeParams.UserIDs = []string{"p1", "p2", "p3"}
		}
		getTimeInRangeCheckStoreParams(storeParams, t)
	}
}
