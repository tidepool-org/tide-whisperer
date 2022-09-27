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
	handlerLogFunc := tidewhisperer.middlewareV1(tidewhisperer.getRangeV1, true, "userID")

	request, _ := http.NewRequest("GET", "/v1/range/"+userID, nil)
	request.Header.Set("x-tidepool-trace-session", traceID)
	request.Header.Set("Authorization", "Bearer "+userID)
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
