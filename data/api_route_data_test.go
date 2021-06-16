package data

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/tidepool-org/tide-whisperer/store"
	"go.mongodb.org/mongo-driver/bson"
)

// Testing GetData route
// Utility function to check which params are passed to store.GetDeviceData
func getDataCheckStoreParams(expectedParams store.Params, t *testing.T) {
	params := storage.GetDeviceDataCall
	if !reflect.DeepEqual(*params.SchemaVersion, *expectedParams.SchemaVersion) {
		t.Fatalf("store.GetDeviceData params.SchemaVersion given [%v] expected [%v] ", *params.SchemaVersion, *expectedParams.SchemaVersion)
	}
	params.SchemaVersion = nil
	expectedParams.SchemaVersion = nil
	if !reflect.DeepEqual(params, expectedParams) {
		t.Fatalf("store.GetDeviceData params given [%v] expected [%v] ", params, expectedParams)
	}
}

// Utility function to prepare request on GetData route
func getDataPrepareRequest(token string, urlParams map[string]string) (*http.Request, *httptest.ResponseRecorder) {
	return prepareGetTestRequest("/patient", token, urlParams)
}

// Utility function to get default params passed to store.GetDeviceData
func getDataStoreDefaultParams() store.Params {
	return store.Params{
		UserID:        "patient",
		Types:         []string{""},
		SubTypes:      []string{""},
		SchemaVersion: &schemaVersions,
		LevelFilter:   []int{0, 1},
		Carelink:      true,
		Medtronic:     true,
	}
}

// TestGetData_NoToken calling GetData route without token should be unauthorized
func TestGetData_NoToken(t *testing.T) {
	resetMocks()

	urlParams := make(map[string]string)
	request, response := getDataPrepareRequest("", urlParams)
	tidewhisperer.GetData(response, request, defaultGetDataURLVars)

	checkUnAuthorized(response, t)
}

// TestGetData_WrongToken calling GetData route with an authorized token should be unauthorized
func TestGetData_WrongToken(t *testing.T) {
	resetMocks()
	mockShoreline.Unauthorized = true

	urlParams := make(map[string]string)
	request, response := getDataPrepareRequest("mytoken", urlParams)
	tidewhisperer.GetData(response, request, defaultGetDataURLVars)

	checkUnAuthorized(response, t)
}

// TestGetData_GoodTokenSameUser calling GetData route for the user owning the token should be authorized
func TestGetData_GoodTokenSameUser(t *testing.T) {
	resetMocks()

	urlParams := make(map[string]string)
	request, response := getDataPrepareRequest("mytoken", urlParams)
	tidewhisperer.GetData(response, request, defaultGetDataURLVars)

	checkAuthorized(response, t)
	getDataCheckStoreParams(defaultGetDataStoreParams, t)
}

// TestGetData_GoodTokenGuestUserNotInvited calling GetData route for a user who didn't invite the user owning the token should be unauthorized
func TestGetData_GoodTokenGuestUserNotInvited(t *testing.T) {
	resetMocks()
	mockShoreline.UserID = "guestUninvited"
	auth := mockPerms.GetMockedAuth(false, map[string]interface{}{}, "tidewhisperer-get")
	mockPerms.SetMockOpaAuth("/patient", &auth, nil)

	urlParams := make(map[string]string)
	request, response := getDataPrepareRequest("mytoken", urlParams)
	tidewhisperer.GetData(response, request, defaultGetDataURLVars)

	checkUnAuthorized(response, t)
}

// TestGetData_GoodTokenGuestUserNotInvited calling GetData route for a user who invited the user owning the token should be authorized
func TestGetData_GoodTokenGuestUserInvited(t *testing.T) {
	resetMocks()
	mockShoreline.UserID = "guest"

	urlParams := make(map[string]string)
	request, response := getDataPrepareRequest("mytoken", urlParams)
	tidewhisperer.GetData(response, request, defaultGetDataURLVars)

	checkAuthorized(response, t)
	getDataCheckStoreParams(defaultGetDataStoreParams, t)
}

// TestGetData_ServerToken calling GetData route for any user with a server token should be authorized
func TestGetData_ServerToken(t *testing.T) {
	resetMocks()
	mockShoreline.UserID = "server"
	mockShoreline.IsServer = true
	auth := mockPerms.GetMockedAuth(false, map[string]interface{}{}, "tidewhisperer-get")
	mockPerms.SetMockOpaAuth("/patient", &auth, nil)

	urlParams := make(map[string]string)
	request, response := getDataPrepareRequest("mytoken", urlParams)
	tidewhisperer.GetData(response, request, defaultGetDataURLVars)

	checkAuthorized(response, t)
	getDataCheckStoreParams(defaultGetDataStoreParams, t)
}

// TestGetData_ValueOutput calling GetData route with valid authorization should return the json values from storage
func TestGetData_ValueOutput(t *testing.T) {
	resetMocks()

	urlParams := make(map[string]string)
	request, response := getDataPrepareRequest("mytoken", urlParams)
	tidewhisperer.GetData(response, request, defaultGetDataURLVars)

	checkAuthorized(response, t)
	getDataCheckStoreParams(defaultGetDataStoreParams, t)
	// Checking body content
	dataBody := parseArrayResponse(response)
	if dataBody[0]["type"] != "A" {
		t.Fatalf("Body data first element type [%s] expected [%s] ", dataBody[0]["type"], "A")
	}
	if dataBody[0]["value"] != "B" {
		t.Fatalf("Body data first element value [%s] expected [%s] ", dataBody[0]["value"], "B")
	}
}

// TestGetData_ValueOutputParametersHistory calling GetData route with valid authorization
// should return the json values from storage with history of parameters injected in "pumpSettings" typed objects
func TestGetData_ValueOutputParametersHistory(t *testing.T) {
	resetMocks()
	storage.DeviceData = make([]string, 1)
	storage.DeviceData[0] = `{"type":"pumpSettings","payload":{"pump":"testPumpData"}}`
	storage.ParametersHistory = bson.M{"history": "testHistoryData"}

	urlParams := make(map[string]string)
	request, response := getDataPrepareRequest("mytoken", urlParams)
	tidewhisperer.GetData(response, request, defaultGetDataURLVars)

	checkAuthorized(response, t)
	getDataCheckStoreParams(defaultGetDataStoreParams, t)
	// Checking body content
	dataBody := parseArrayResponse(response)
	if dataBody[0]["type"] != "pumpSettings" {
		t.Fatalf("Body data first element type [%s] expected [%s] ", dataBody[0]["type"], "pumpSettings")
	}
	payload := dataBody[0]["payload"].(map[string]interface{})
	if payload["pump"] != "testPumpData" {
		t.Fatalf("Body data first element payload->pump  [%s] expected [%s] ", payload["pump"], "testPumpData")
	}
	if payload["history"] != "testHistoryData" {
		t.Fatalf("Body data first element payload->history  [%s] expected [%s] ", payload["history"], "testHistoryData")
	}
}

func TestGetData_UrlParameters(t *testing.T) {
	resetMocks()

	// Testing boolean query paramters
	for _, boolField := range []string{"carelink", "dexcom", "latest", "medtronic"} {
		storage.DeviceData = make([]string, 1)
		if boolField == "latest" {
			storage.DeviceData[0] = `{"latest_doc":{"type":"A","value":"B"}}`
		} else {
			storage.DeviceData[0] = `{"type":"A","value":"B"}`
		}
		urlParams := make(map[string]string)
		urlParams[boolField] = "1"
		request, response := getDataPrepareRequest("mytoken", urlParams)
		tidewhisperer.GetData(response, request, defaultGetDataURLVars)

		checkAuthorized(response, t)
		dataStoreParams := getDataStoreDefaultParams()
		switch boolField {
		case "carelink":
			dataStoreParams.Carelink = true
		case "dexcom":
			dataStoreParams.Dexcom = true
		case "latest":
			dataStoreParams.Latest = true
		case "medtronic":
			dataStoreParams.Medtronic = true
		}
		getDataCheckStoreParams(dataStoreParams, t)

		storage.GetDeviceDataCalled = false
		urlParams[boolField] = ""
		request, response = getDataPrepareRequest("mytoken", urlParams)
		tidewhisperer.GetData(response, request, defaultGetDataURLVars)
		checkInvalidParams(response, t)
		if storage.GetDeviceDataCalled != false {
			t.Fatalf("store.GetDeviceData called  [%v] expected [%v] ", storage.GetDeviceDataCalled, false)
		}
	}

	storage.DeviceData = make([]string, 1)
	storage.DeviceData[0] = `{"type":"A","value":"B"}`
	// Testing date query parameters
	for _, dateField := range []string{"startDate", "endDate"} {
		urlParams := make(map[string]string)
		dateValue := "2020-05-11T08:00:00Z"
		urlParams[dateField] = dateValue
		request, response := getDataPrepareRequest("mytoken", urlParams)
		tidewhisperer.GetData(response, request, defaultGetDataURLVars)
		checkAuthorized(response, t)
		dataStoreParams := getDataStoreDefaultParams()
		switch dateField {
		case "startDate":
			dataStoreParams.Date = store.Date{Start: dateValue, End: ""}
		case "endDate":
			dataStoreParams.Date = store.Date{Start: "", End: dateValue}
		}
		getDataCheckStoreParams(dataStoreParams, t)

		storage.GetDeviceDataCalled = false
		urlParams[dateField] = "invalid_date"
		request, response = getDataPrepareRequest("mytoken", urlParams)
		tidewhisperer.GetData(response, request, defaultGetDataURLVars)
		checkInvalidParams(response, t)
		if storage.GetDeviceDataCalled != false {
			t.Fatalf("store.GetDeviceData called  [%v] expected [%v] ", storage.GetDeviceDataCalled, false)
		}
	}

	// Testing string array query parameters
	for _, stringArrayField := range []string{"type", "subType"} {
		urlParams := make(map[string]string)
		arrValue := "1,2,3"
		urlParams[stringArrayField] = arrValue
		request, response := getDataPrepareRequest("mytoken", urlParams)
		tidewhisperer.GetData(response, request, defaultGetDataURLVars)
		checkAuthorized(response, t)
		dataStoreParams := getDataStoreDefaultParams()
		switch stringArrayField {
		case "type":
			dataStoreParams.Types = []string{"1", "2", "3"}
		case "subType":
			dataStoreParams.SubTypes = []string{"1", "2", "3"}
		}
		getDataCheckStoreParams(dataStoreParams, t)
	}

	// Testing string query parameters
	for _, stringField := range []string{"deviceId", "uploadId"} {
		urlParams := make(map[string]string)
		strValue := "test"
		urlParams[stringField] = strValue
		request, response := getDataPrepareRequest("mytoken", urlParams)
		tidewhisperer.GetData(response, request, defaultGetDataURLVars)
		checkAuthorized(response, t)
		dataStoreParams := getDataStoreDefaultParams()
		switch stringField {
		case "deviceId":
			dataStoreParams.DeviceID = strValue
		case "uploadId":
			dataStoreParams.UploadID = strValue
		}
		getDataCheckStoreParams(dataStoreParams, t)
	}

	// Testing LevelFilter
	storage.DeviceModel = "DBLHU"
	urlParams := make(map[string]string)
	request, response := getDataPrepareRequest("mytoken", urlParams)
	tidewhisperer.GetData(response, request, defaultGetDataURLVars)
	checkAuthorized(response, t)
	dataStoreParams := getDataStoreDefaultParams()
	dataStoreParams.LevelFilter = []int{0, 1, 2, 3}
	getDataCheckStoreParams(dataStoreParams, t)
}
