package data

import (
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
	handlerLogFunc := tidewhisperer.middlewareV1(tidewhisperer.getRangeV1, "userID")

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
	handlerLogFunc := tidewhisperer.middlewareV1(tidewhisperer.getDataV1, "userID")

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
