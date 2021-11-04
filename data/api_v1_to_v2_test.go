package data

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/mdblp/tide-whisperer-v2/schema"
)

const dayTimeFormat = "2006-01-02"

func TestAPI_GetDataV2(t *testing.T) {
	traceID := uuid.New().String()
	userID := "abcdef"
	urlParams := map[string]string{
		"userID": userID,
	}

	storage.DataV1 = []string{
		"{\"id\":\"01\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:00.000Z\",\"type\":\"basal\",\"value\":10}",
		"{\"id\":\"02\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:01.000Z\",\"type\":\"basal\",\"value\":11}",
		"{\"id\":\"03\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:02.000Z\",\"type\":\"basal\",\"value\":12}",
		"{\"id\":\"04\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:03.000Z\",\"type\":\"basal\",\"value\":13}",
		"{\"id\":\"05\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:04.000Z\",\"type\":\"basal\",\"value\":14}",
	}
	storage.DataIDV1 = []string{
		"{\"id\":\"00\",\"uploadId\":\"00\",\"time\":\"2021-01-10T00:00:00.000Z\",\"type\":\"upload\"}",
	}

	creationTime1, _ := time.Parse(time.RFC3339, "2021-01-01T08:00:00Z")
	day1, _ := time.Parse(dayTimeFormat, "2021-01-01")

	creationTime2, _ := time.Parse(time.RFC3339, "2021-01-02T08:00:00Z")
	day2, _ := time.Parse(dayTimeFormat, "2021-01-02")

	mockTideV2.MockedData = []schema.CbgBucket{
		{
			Id:                "bucket1",
			CreationTimestamp: creationTime1,
			UserId:            userID,
			Day:               day1,
			Samples: []schema.CbgSample{
				{
					Value:          10.0,
					Units:          "mmol/L",
					Timestamp:      day1.Add(time.Minute * 5),
					Timezone:       "GMT",
					TimezoneOffset: 0,
				},
				{
					Value:          10.2,
					Units:          "mmol/L",
					Timestamp:      day1.Add(time.Minute * 10),
					Timezone:       "GMT",
					TimezoneOffset: 0,
				},
				{
					Value:          10.8,
					Units:          "mmol/L",
					Timestamp:      day1.Add(time.Minute * 15),
					Timezone:       "GMT",
					TimezoneOffset: 0,
				},
			},
		},
		{
			Id:                "bucket2",
			CreationTimestamp: creationTime2,
			UserId:            userID,
			Day:               day2,
			Samples: []schema.CbgSample{
				{
					Value:          11.0,
					Units:          "mmol/L",
					Timestamp:      day2.Add(time.Minute * 5),
					Timezone:       "GMT",
					TimezoneOffset: 0,
				},
				{
					Value:          11.2,
					Units:          "mmol/L",
					Timestamp:      day2.Add(time.Minute * 10),
					Timezone:       "GMT",
					TimezoneOffset: 0,
				},
				{
					Value:          11.8,
					Units:          "mmol/L",
					Timestamp:      day2.Add(time.Minute * 15),
					Timezone:       "GMT",
					TimezoneOffset: 0,
				},
			},
		},
	}
	t.Cleanup(func() {
		storage.DataV1 = nil
		storage.DataIDV1 = nil
		mockTideV2.MockedData = []schema.CbgBucket{}
	})

	resetOPAMockRouteV1(true, "/v1/dataV2", userID)
	handlerLogFunc := tidewhisperer.middlewareV1(tidewhisperer.getDataV2, true, "userID")

	request, _ := http.NewRequest("GET", "/v1/dataV2/"+userID, nil)
	request.Header.Set("x-tidepool-trace-session", traceID)
	request.Header.Set("x-tidepool-session-token", userID)
	request = mux.SetURLVars(request, urlParams)
	response := httptest.NewRecorder()

	handlerLogFunc(response, request)
	result := response.Result()
	if result.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d to equal %d", response.Code, http.StatusOK)
	}

	body := make([]byte, 2048)
	defer result.Body.Close()
	n, _ := result.Body.Read(body)
	bodyStr := string(body[:n])

	expectedBody := `[
{"id":"01","time":"2021-01-10T00:00:00.000Z","type":"basal","uploadId":"00","value":10},
{"id":"02","time":"2021-01-10T00:00:01.000Z","type":"basal","uploadId":"00","value":11},
{"id":"03","time":"2021-01-10T00:00:02.000Z","type":"basal","uploadId":"00","value":12},
{"id":"04","time":"2021-01-10T00:00:03.000Z","type":"basal","uploadId":"00","value":13},
{"id":"05","time":"2021-01-10T00:00:04.000Z","type":"basal","uploadId":"00","value":14},
{"id":"bucket1_0","time":"2021-01-01T00:05:00Z","timezone":"GMT","type":"cbg","units":"mmol/L","value":10},
{"id":"bucket1_1","time":"2021-01-01T00:10:00Z","timezone":"GMT","type":"cbg","units":"mmol/L","value":10.2},
{"id":"bucket1_2","time":"2021-01-01T00:15:00Z","timezone":"GMT","type":"cbg","units":"mmol/L","value":10.8},
{"id":"bucket2_0","time":"2021-01-02T00:05:00Z","timezone":"GMT","type":"cbg","units":"mmol/L","value":11},
{"id":"bucket2_1","time":"2021-01-02T00:10:00Z","timezone":"GMT","type":"cbg","units":"mmol/L","value":11.2},
{"id":"bucket2_2","time":"2021-01-02T00:15:00Z","timezone":"GMT","type":"cbg","units":"mmol/L","value":11.8},
{"id":"00","time":"2021-01-10T00:00:00.000Z","type":"upload","uploadId":"00"}]
`

	if bodyStr != expectedBody {
		t.Fatalf("Expected '%s' to equal '%s'", bodyStr, expectedBody)
	}
}
