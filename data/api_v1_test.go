package data

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func TestAPI_GetRangeV1(t *testing.T) {
	traceID := uuid.New().String()
	userID := "abcdef"
	urlParams := map[string]string{
		"userID": userID,
	}

	resetOPAMockRouteV1(true, "/v1/range", userID)
	storage.DataRangeV1 = []string{"2021-01-01T00:00:00.000Z", "2021-01-03T00:00:00.000Z"}
	t.Cleanup(func() {
		storage.DataRangeV1 = nil
	})
	expectedValue := "[\"" + storage.DataRangeV1[0] + "\",\"" + storage.DataRangeV1[1] + "\"]"
	handlerLogFunc := tidewhisperer.middlewareV1(tidewhisperer.getRangeV1, true, "userID")

	request, _ := http.NewRequest("GET", "/v1/range/"+userID, nil)
	request.Header.Set("x-tidepool-trace-session", traceID)
	request.Header.Set("x-tidepool-session-token", userID)
	request = mux.SetURLVars(request, urlParams)
	response := httptest.NewRecorder()

	handlerLogFunc(response, request)
	result := response.Result()
	if result.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d to equal %d", response.Code, http.StatusOK)
	}

	body := make([]byte, 1024)
	defer result.Body.Close()
	n, _ := result.Body.Read(body)
	bodyStr := string(body[:n])

	if bodyStr != expectedValue {
		t.Errorf("Expected '%s' to equal '%s'", bodyStr, expectedValue)
	}
}

func TestAPI_GetDataV1(t *testing.T) {
	traceID := uuid.New().String()
	userID := "abcdef"
	urlParams := map[string]string{
		"userID": userID,
	}

	storage.DataV1 = []string{
		"{\"id\":\"01\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:00.000Z\",\"type\":\"cbg\",\"value\":10,\"units\":\"mmol/L\"}",
		"{\"id\":\"02\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:01.000Z\",\"type\":\"cbg\",\"value\":11,\"units\":\"mmol/L\"}",
		"{\"id\":\"03\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:02.000Z\",\"type\":\"cbg\",\"value\":12,\"units\":\"mmol/L\"}",
		"{\"id\":\"04\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:03.000Z\",\"type\":\"cbg\",\"value\":13,\"units\":\"mmol/L\"}",
		"{\"id\":\"05\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:04.000Z\",\"type\":\"cbg\",\"value\":14,\"units\":\"mmol/L\"}",
	}
	storage.DataIDV1 = []string{
		"{\"id\":\"00\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:00.000Z\",\"type\":\"upload\"}",
	}
	t.Cleanup(func() {
		storage.DataV1 = nil
		storage.DataIDV1 = nil
	})

	resetOPAMockRouteV1(true, "/v1/data", userID)
	handlerLogFunc := tidewhisperer.middlewareV1(tidewhisperer.getDataV1, true, "userID")

	request, _ := http.NewRequest("GET", "/v1/data/"+userID, nil)
	request.Header.Set("x-tidepool-trace-session", traceID)
	request.Header.Set("x-tidepool-session-token", userID)
	request = mux.SetURLVars(request, urlParams)
	response := httptest.NewRecorder()

	handlerLogFunc(response, request)
	result := response.Result()
	if result.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d to equal %d", response.Code, http.StatusOK)
	}

	body := make([]byte, 1024)
	defer result.Body.Close()
	n, _ := result.Body.Read(body)
	bodyStr := string(body[:n])

	expectedBody := `[
{"id":"01","time":"2021-01-10T00:00:00.000Z","type":"cbg","units":"mmol/L","uploadId":"00","value":10},
{"id":"02","time":"2021-01-10T00:00:01.000Z","type":"cbg","units":"mmol/L","uploadId":"00","value":11},
{"id":"03","time":"2021-01-10T00:00:02.000Z","type":"cbg","units":"mmol/L","uploadId":"00","value":12},
{"id":"04","time":"2021-01-10T00:00:03.000Z","type":"cbg","units":"mmol/L","uploadId":"00","value":13},
{"id":"05","time":"2021-01-10T00:00:04.000Z","type":"cbg","units":"mmol/L","uploadId":"00","value":14},
{"id":"00","time":"2021-01-10T00:00:00.000Z","type":"upload","uploadId":"00"}]
`

	if bodyStr != expectedBody {
		t.Fatalf("Expected '%s' to equal '%s'", bodyStr, expectedBody)
	}
}

func TestAPI_GetDataV1_Parameters(t *testing.T) {
	traceID := uuid.New().String()
	userID := "abcdef"
	urlParams := map[string]string{
		"userID": userID,
	}

	storage.DataV1 = []string{
		"{\"id\":\"01\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:01.000Z\",\"type\":\"deviceEvent\",\"subType\":\"deviceParameter\",\"level\":\"1\"}",
		"{\"id\":\"02\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:02.000Z\",\"type\":\"deviceEvent\",\"subType\":\"deviceParameter\",\"level\":\"2\"}",
		"{\"id\":\"03\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:03.000Z\",\"type\":\"deviceEvent\",\"subType\":\"deviceParameter\",\"level\":\"3\"}",
	}
	storage.DataIDV1 = []string{
		"{\"id\":\"00\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:00.000Z\",\"type\":\"upload\"}",
	}
	t.Cleanup(func() {
		storage.DataV1 = nil
		storage.DataIDV1 = nil
	})

	resetOPAMockRouteV1(true, "/v1/data", userID)
	handlerLogFunc := tidewhisperer.middlewareV1(tidewhisperer.getDataV1, true, "userID")

	request, _ := http.NewRequest("GET", "/v1/data/"+userID, nil)
	request.Header.Set("x-tidepool-trace-session", traceID)
	request.Header.Set("x-tidepool-session-token", userID)
	request = mux.SetURLVars(request, urlParams)
	response := httptest.NewRecorder()

	handlerLogFunc(response, request)
	result := response.Result()
	if result.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d to equal %d", response.Code, http.StatusOK)
	}

	body := make([]byte, 1024)
	defer result.Body.Close()
	n, _ := result.Body.Read(body)
	bodyStr := string(body[:n])

	expectedBody := `[
{"id":"01","level":"1","subType":"deviceParameter","time":"2021-01-10T00:00:01.000Z","type":"deviceEvent","uploadId":"00"},
{"id":"02","level":"2","subType":"deviceParameter","time":"2021-01-10T00:00:02.000Z","type":"deviceEvent","uploadId":"00"},
{"id":"00","time":"2021-01-10T00:00:00.000Z","type":"upload","uploadId":"00"}]
`

	if bodyStr != expectedBody {
		t.Fatalf("Expected '%s' to equal '%s'", bodyStr, expectedBody)
	}
}

func TestAPI_GetDataSummaryV1(t *testing.T) {
	userID := "abcdef"
	urlParams := map[string]string{
		"userID": userID,
	}
	pumpSettings := &PumpSettings{
		ID:   "1",
		Type: "pumpSettings",
		Time: "2021-01-02T20:00:00.000Z",
		Payload: pumpSettingsPayload{
			Parameters: []deviceParameter{
				{
					Level: 1,
					Name:  "PATIENT_GLY_HYPO_LIMIT",
					Value: "70",
					Unit:  "mg/dL",
				}, {
					Level: 1,
					Name:  "PATIENT_GLY_HYPER_LIMIT",
					Value: "180",
					Unit:  "mg/dL",
				}, {
					Level: 1,
					Name:  "WEIGHT",
					Value: "80",
					Unit:  "kg",
				},
			},
		},
	}
	pumpSettingsJSON, err := json.Marshal(pumpSettings)
	if err != nil {
		t.Fatalf("Marshal pump settings: %e", err)
	}
	dataPS := string(pumpSettingsJSON)

	resetOPAMockRouteV1(true, "/v1/summary", userID)
	handlerLogFunc := tidewhisperer.middlewareV1(tidewhisperer.getDataSummaryV1, true, "userID")

	storage.DataRangeV1 = []string{"2021-01-01T00:00:00.000Z", "2021-01-03T00:00:00.000Z"}
	storage.DataPSV1 = &dataPS
	storage.DataBGV1 = []string{
		"{\"value\":60,\"units\":\"mg/dL\"}",
		"{\"value\":80,\"units\":\"mg/dL\"}",
		"{\"value\":84,\"units\":\"mg/dL\"}",
		"{\"value\":200,\"units\":\"mg/dL\"}",
	}
	t.Cleanup(func() {
		storage.DataRangeV1 = nil
		storage.DataPSV1 = nil
		storage.DataBGV1 = nil
	})

	request, _ := http.NewRequest("GET", "/v1/summary/"+userID, nil)
	request.Header.Set("x-tidepool-session-token", userID)
	request = mux.SetURLVars(request, urlParams)
	response := httptest.NewRecorder()

	handlerLogFunc(response, request)
	result := response.Result()
	if result.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d to equal %d", response.Code, http.StatusOK)
	}

	body := make([]byte, 256)
	defer result.Body.Close()
	n, _ := result.Body.Read(body)
	bodyStr := string(body[:n])

	expectedBody := "{\"userId\":\"abcdef\",\"rangeStart\":\"2021-01-01T00:00:00.000Z\",\"rangeEnd\":\"2021-01-03T00:00:00.000Z\",\"computeDays\":1,\"percentTimeInRange\":50,\"percentTimeBelowRange\":25,\"numBgValues\":4,\"glyHypoLimit\":70,\"glyHyperLimit\":180,\"glyUnit\":\"mg/dL\"}"
	if bodyStr != expectedBody {
		t.Fatalf("Expected '%s' to equal '%s'", bodyStr, expectedBody)
	}
}
