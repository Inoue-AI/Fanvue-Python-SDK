package fanvue

import (
	"errors"
	"fmt"
	"net/http"
)

// Error represents a non-2xx response from the Fanvue API. Fanvue uses real
// HTTP status codes, so StatusCode is the primary signal; Message carries the
// human-readable description extracted from the response body (which may use a
// "message", "error", or "errors" field depending on the endpoint).
type Error struct {
	StatusCode int      // HTTP status code returned by Fanvue.
	Message    string   // Human-readable description extracted from the body.
	Errors     []string // Field-level validation errors (422-style responses), if any.
	RequestID  string   // Value of the x-request-id response header, for tracing.
	Body       []byte   // Raw response body, retained for diagnostics.
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("fanvue: %d: %s (request_id=%s)", e.StatusCode, e.Message, e.RequestID)
	}
	return fmt.Sprintf("fanvue: %d: %s", e.StatusCode, e.Message)
}

// IsUnauthorized reports whether the request failed authentication (HTTP 401).
// The caller must supply a fresh OAuth access token to recover.
func (e *Error) IsUnauthorized() bool { return e.StatusCode == http.StatusUnauthorized }

// IsForbidden reports whether the token lacks the required scope (HTTP 403).
func (e *Error) IsForbidden() bool { return e.StatusCode == http.StatusForbidden }

// IsNotFound reports whether the addressed resource does not exist (HTTP 404).
func (e *Error) IsNotFound() bool { return e.StatusCode == http.StatusNotFound }

// IsGone reports whether the API version has been sunset (HTTP 410). The caller
// should upgrade the X-Fanvue-API-Version they send.
func (e *Error) IsGone() bool { return e.StatusCode == http.StatusGone }

// IsRateLimited reports whether Fanvue throttled the request (HTTP 429).
func (e *Error) IsRateLimited() bool { return e.StatusCode == http.StatusTooManyRequests }

// IsServerError reports whether Fanvue returned a 5xx failure that may be
// transient and worth retrying with backoff.
func (e *Error) IsServerError() bool { return e.StatusCode >= 500 }

// AsError unwraps any error returned by this package into a *Error if it is
// one, returning the typed error and true. Otherwise it returns nil and false.
func AsError(err error) (*Error, bool) {
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}
