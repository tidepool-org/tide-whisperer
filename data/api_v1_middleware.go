package data

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type (
	// httpResponseWriter used for middleware api functions.
	//
	// Use a string builder, so we can send back a valid error response
	// even if an error occurred after the first write
	//
	// - Uppercase fields are for the the functions handlers functions
	//
	// - Lowercase for the middleware (~private)
	httpResponseWriter struct {
		URL         *url.URL
		VARS        map[string]string
		TraceID     string
		Header      http.Header
		writeBuffer strings.Builder
		statusCode  int
		err         *detailedError
		size        int
	}

	// HandlerLoggerFunc expose our httpResponseWriter API
	HandlerLoggerFunc func(context.Context, *httpResponseWriter) error

	// RequestLoggerFunc type to simplify func signatures
	RequestLoggerFunc func(HandlerLoggerFunc) HandlerLoggerFunc
)

var emptyUserIDs = []string{}

func (res *httpResponseWriter) Grow(n int) {
	if n > 0 { // Avoid Grow panic()
		res.writeBuffer.Grow(n)
	} else {
		res.err = &detailedError{
			Status:          http.StatusInternalServerError,
			Code:            "write_error",
			Message:         "Internal Server Error",
			InternalMessage: "Grow(): writeBuffer is nil",
			ID:              res.TraceID,
		}
	}
}

func (res *httpResponseWriter) Write(v []byte) error {
	size, err := res.writeBuffer.Write(v)
	res.size += size
	return err
}

func (res *httpResponseWriter) WriteString(s string) error {
	size, err := res.writeBuffer.WriteString(s)
	res.size += size
	return err
}

// WriteError final writing to the response
func (res *httpResponseWriter) WriteError(err *detailedError) error {
	if err == nil {
		err = &detailedError{
			Status:          http.StatusInternalServerError,
			Code:            "unknown_error",
			Message:         "Unknown error",
			InternalMessage: "WriteError() with nil error",
		}
	}

	res.err = err
	res.err.ID = res.TraceID

	// Discard the previous content write, so we ends up with
	// a valid json returned to the client
	res.writeBuffer.Reset()

	jsonErr, _ := json.Marshal(err)
	res.WriteHeader(err.Status)
	return res.Write(jsonErr)
}

func (res *httpResponseWriter) WriteHeader(statusCode int) {
	res.statusCode = statusCode
}

// middlewareV1 middleware to log received requests
func (a *API) middlewareV1(fn HandlerLoggerFunc, checkPermissions bool, params ...string) http.HandlerFunc {
	// The mux handler func:
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		start := time.Now().UTC()

		// It is recommended by go to get the request information before writing
		// So get theses now

		logErrors := make([]string, 0, 5)
		logRequest := fmt.Sprintf("%s - %s %s HTTP/%d.%d", r.RemoteAddr, r.Method, r.URL.String(), r.ProtoMajor, r.ProtoMinor)

		// TODO: use x-client-trace-id ?
		// https://docs.solo.io/gloo-edge/latest/guides/observability/tracing/
		traceID := r.Header.Get("x-tidepool-trace-session")
		if !isValidUUID(traceID) {
			// We want a trace id, but for now we do not enforce it
			logErrors = append(logErrors, fmt.Sprintf("no-trace:\"%s\"", traceID))
			traceID = uuid.New().String()
		}

		// Make our context
		ctx := timeItContext(r.Context())

		res := httpResponseWriter{
			Header:     r.Header.Clone(), // Clone the header, to be sure
			URL:        r.URL,
			VARS:       nil,
			TraceID:    traceID,
			statusCode: http.StatusOK, // Default status
			err:        nil,
		}

		userIDs := emptyUserIDs
		// The handler have parameters, get them
		if len(params) > 0 {
			res.VARS = mux.Vars(r) // Decode route parameter

			if contains(params, "userID") {
				// userID is a commonly used parameter
				// See if we can view the data
				userID := res.VARS["userID"]
				userIDs = []string{userID}

				if len(userID) > 64 {
					// Quick verification on the userID for security reason
					// Partial but may help without beeing a burden
					// 64 characters is probably a good compromise
					res.WriteError(&detailedError{
						Status:          http.StatusBadRequest,
						Code:            "invalid_userid",
						Message:         "Invalid parameter userId",
						InternalMessage: "userID do not match the regex",
					})
				}
			}
		}

		if checkPermissions && !a.isAuthorized(r, userIDs) {
			err = res.WriteError(&errorNoViewPermission)
		}

		// Mainteners: No read from the request below this point!

		// Make the call to the API function if we can:
		if res.err == nil {
			err = fn(ctx, &res)
			if err != nil {
				logErrors = append(logErrors, fmt.Sprintf("efn:\"%s\"", err))
			}
		}

		// We will send a JSON, so advertise it for all of our requests
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(res.statusCode)
		_, err = w.Write([]byte(res.writeBuffer.String()))
		if err != nil {
			logErrors = append(logErrors, fmt.Sprintf("eww:\"%s\"", err))
		}

		// Log errors management
		if res.err != nil {
			if res.err.Code != "" {
				logErrors = append(logErrors, fmt.Sprintf("code:\"%s\"", res.err.Code))
			}
			if res.err.InternalMessage != "" {
				logErrors = append(logErrors, fmt.Sprintf("err:\"%s\"", res.err.InternalMessage))
			}
		}

		// Get the time spent on it
		end := time.Now().UTC()
		dur := end.Sub(start).Milliseconds()
		// Log the message
		var logError string
		if len(logErrors) > 0 {
			logError = fmt.Sprintf("{%s} - ", strings.Join(logErrors, ","))
		}

		timerResults := timeResults(ctx)
		if len(timerResults) > 0 {
			timerResults = fmt.Sprintf("{%s} %d ms", timerResults, dur)
		} else {
			timerResults = fmt.Sprintf("%d ms", dur)
		}
		a.logger.Printf("{%s} %s %d - %s%s - %d bytes", traceID, logRequest, res.statusCode, logError, timerResults, res.size)
	}
}
