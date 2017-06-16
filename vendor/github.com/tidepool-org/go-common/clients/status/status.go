package status

import (
	"fmt"
	"net/http"
)

// Type status holds the status return from an http request.
type Status struct {
	Code   int    `json:"code"`
	Reason string `json:"reason"`
}

// NewStatus constructs a Status object; if no reason is provided, it uses the
// standard one.
func NewStatus(code int, reason string) Status {
	s := Status{Code: code, Reason: reason}
	if s.Reason == "" {
		s.Reason = http.StatusText(code)
	}
	return s
}

// NewStatus constructs a Status object; if no reason is provided, it uses the
// standard one.
func NewStatusf(code int, reason string, args ...interface{}) Status {
	return Status{Code: code, Reason: fmt.Sprintf(reason, args...)}
}

func StatusFromResponse(res *http.Response) Status {
	return Status{Code: res.StatusCode, Reason: res.Status}
}

// String() converts a status to a printable string.
func (s Status) String() string {
	return fmt.Sprintf("%d %s", s.Code, s.Reason)
}

// StatusError represents a Status as an error object.
type StatusError struct {
	Status
}

// Error() renders a StatusError.
func (s *StatusError) Error() string {
	return s.Status.String()
}
