package data

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func getDefaultResponseWriter(t *testing.T) *httpResponseWriter {
	testURL, err := url.Parse("https://localhost/")
	if err != nil {
		t.Fatal("Invalid test URL")
	}
	res := &httpResponseWriter{
		Header:     http.Header{},
		URL:        testURL,
		VARS:       nil,
		TraceID:    uuid.New().String(),
		statusCode: http.StatusOK, // Default status
		err:        nil,
	}
	return res
}

func resetOPAMockRouteV1(authorized bool, route string, userID string) {
	mockShoreline.UserID = userID
	mockShoreline.IsServer = false
	mockShoreline.Unauthorized = !authorized
	auth := mockPerms.GetMockedAuth(authorized, map[string]interface{}{}, "tidewhisperer-v1")
	mockPerms.SetMockOpaAuth(route, &auth, nil)
}

func TestApiV1MiddlewareHrwWrite(t *testing.T) {
	res := getDefaultResponseWriter(t)

	value := "OK"
	res.Write([]byte(value))
	if res.size != 2 {
		t.Fatalf("Expected %d to equal 2", res.size)
	}
	result := res.writeBuffer.String()
	if result != value {
		t.Fatalf("Expected `%s` to equal `%s`", result, value)
	}
}

func TestApiV1MiddlewareHrwWriteString(t *testing.T) {
	res := getDefaultResponseWriter(t)

	value := "OK"
	res.WriteString(value)
	if res.size != 2 {
		t.Fatalf("Expected %d to equal 2", res.size)
	}
	result := res.writeBuffer.String()
	if result != value {
		t.Fatalf("Expected `%s` to equal `%s`", result, value)
	}
}

func TestApiV1MiddlewareWriteHeader(t *testing.T) {
	res := getDefaultResponseWriter(t)

	code := http.StatusNotFound
	res.WriteHeader(code)
	if res.statusCode != code {
		t.Fatalf("Expected %d to equal %d", res.statusCode, code)
	}
}

func TestApiV1MiddlewareHrwWriteError(t *testing.T) {
	res := getDefaultResponseWriter(t)
	value := &detailedError{Status: http.StatusNotFound, Code: "data_not_found", Message: "no data for specified user"}
	res.WriteError(value)
	if res.err == nil {
		t.Fatalf("Expected err to be not nil")
	}
	if res.err.Code != value.Code {
		t.Fatalf("Expected `%s` to equal `%s`", res.err.Code, value.Code)
	}
	if res.statusCode != value.Status {
		t.Fatalf("Expected %d to equal %d", res.statusCode, value.Status)
	}
	result := res.writeBuffer.String()
	valueString := fmt.Sprintf("{\"status\":%d,\"id\":\"%s\",\"code\":\"data_not_found\",\"message\":\"no data for specified user\"}", value.Status, res.TraceID)
	if result != valueString {
		t.Fatalf("Expected `%s` to equal `%s`", result, valueString)
	}
}

func TestApiV1MiddlewareNoError(t *testing.T) {
	value := "[\"OK\"]"
	handlerFunc := func(ctx context.Context, res *httpResponseWriter) error {
		res.WriteString(value)
		return nil
	}

	handlerLogFunc := tidewhisperer.middlewareV1(handlerFunc)

	traceID := uuid.New().String()
	request, _ := http.NewRequest("GET", "/test", nil)
	request.Header.Set("x-tidepool-trace-session", traceID)
	response := httptest.NewRecorder()
	handlerLogFunc(response, request)

	result := response.Result()
	if result.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d to equal %d", response.Code, http.StatusOK)
	}

	contentType := result.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("Expected `%s` to equal `application/json`", contentType)
	}

	body := make([]byte, 1024)
	defer result.Body.Close()
	n, _ := result.Body.Read(body)
	bodyStr := string(body[:n])
	if bodyStr != value {
		t.Fatalf("Expected `%s` to equal `%s`", bodyStr, value)
	}
}

func TestApiV1MiddlewareErrorResponse(t *testing.T) {
	value := &detailedError{Status: http.StatusNotFound, Code: "data_not_found", Message: "no data for specified user"}
	handlerFunc := func(ctx context.Context, res *httpResponseWriter) error {
		res.WriteError(value)
		return nil
	}

	handlerLogFunc := tidewhisperer.middlewareV1(handlerFunc)

	traceID := uuid.New().String()
	request, _ := http.NewRequest("GET", "/test", nil)
	request.Header.Set("x-tidepool-trace-session", traceID)
	response := httptest.NewRecorder()
	handlerLogFunc(response, request)

	result := response.Result()
	if result.StatusCode != value.Status {
		t.Fatalf("Expected %d to equal %d", response.Code, value.Status)
	}

	contentType := result.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("Expected `%s` to equal `application/json`", contentType)
	}

	valueString := fmt.Sprintf("{\"status\":%d,\"id\":\"%s\",\"code\":\"data_not_found\",\"message\":\"no data for specified user\"}", http.StatusNotFound, traceID)
	body := make([]byte, 1024)
	defer result.Body.Close()
	n, _ := result.Body.Read(body)
	bodyStr := string(body[:n])
	if bodyStr != valueString {
		t.Fatalf("Expected `%s` to equal `%s`", bodyStr, valueString)
	}
}

func TestApiV1MiddlewareNoErrorWithUserID(t *testing.T) {
	value := "[\"OK\"]"
	traceID := uuid.New().String()
	urlParams := map[string]string{
		"userID": "abcdef",
	}
	handlerFuncCalled := false

	handlerFunc := func(ctx context.Context, res *httpResponseWriter) error {
		res.WriteString(value)
		handlerFuncCalled = true
		return nil
	}

	handlerLogFunc := tidewhisperer.middlewareV1(handlerFunc, "userID")

	request, _ := http.NewRequest("GET", "/test/abcdef", nil)
	request.Header.Set("x-tidepool-trace-session", traceID)
	request.Header.Set("x-tidepool-session-token", "123456")
	request = mux.SetURLVars(request, urlParams)
	response := httptest.NewRecorder()

	resetOPAMockRouteV1(true, "/test", urlParams["userID"])
	handlerLogFunc(response, request)

	if !handlerFuncCalled {
		t.Fatalf("Handle func should have been called")
	}

	result := response.Result()
	if result.StatusCode != http.StatusOK {
		t.Fatalf("Expected %d to equal %d", response.Code, http.StatusOK)
	}

	contentType := result.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("Expected `%s` to equal `application/json`", contentType)
	}

	body := make([]byte, 1024)
	defer result.Body.Close()
	n, _ := result.Body.Read(body)
	bodyStr := string(body[:n])
	if bodyStr != value {
		t.Fatalf("Expected `%s` to equal `%s`", bodyStr, value)
	}
}

func TestApiV1MiddlewareNotAuthorizedWithUserID(t *testing.T) {
	value := "[\"OK\"]"
	traceID := uuid.New().String()
	urlParams := map[string]string{
		"userID": "abcdef",
	}
	handlerFuncCalled := false

	handlerFunc := func(ctx context.Context, res *httpResponseWriter) error {
		fmt.Println("You should not see me")
		res.WriteString(value)
		handlerFuncCalled = true
		return nil
	}

	handlerLogFunc := tidewhisperer.middlewareV1(handlerFunc, "userID")

	request, _ := http.NewRequest("GET", "/test/abcdef", nil)
	request.Header.Set("x-tidepool-trace-session", traceID)
	request.Header.Set("x-tidepool-session-token", "123456")
	request = mux.SetURLVars(request, urlParams)
	response := httptest.NewRecorder()

	resetOPAMockRouteV1(false, "/test", "abcdef")
	handlerLogFunc(response, request)

	if handlerFuncCalled {
		t.Fatalf("Handle func should not have been called")
	}

	result := response.Result()
	if result.StatusCode != errorNoViewPermission.Status {
		t.Fatalf("Expected %d to equal %d", response.Code, errorNoViewPermission.Status)
	}

	contentType := result.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Fatalf("Expected `%s` to equal `application/json`", contentType)
	}

	body := make([]byte, 1024)
	defer result.Body.Close()
	n, _ := result.Body.Read(body)
	bodyStr := string(body[:n])

	errorText := fmt.Sprintf("{\"status\":%d,\"id\":\"%s\",\"code\":\"%s\",\"message\":\"%s\"}", errorNoViewPermission.Status, traceID, errorNoViewPermission.Code, errorNoViewPermission.Message)
	if bodyStr != errorText {
		t.Fatalf("Expected `%s` to equal `%s`", bodyStr, errorText)
	}
}
