package fanvue

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	testAPIVersion = "2025-06-26"
	testToken      = "test-access-token"
)

// newTestClient wires a *Client to an in-process httptest.Server so no real
// network calls are ever made.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := New(ClientOptions{
		APIVersion:  testAPIVersion,
		AccessToken: testToken,
		BaseURL:     srv.URL,
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c, srv
}

func ptrString(s string) *string { return &s }
func ptrInt(i int) *int          { return &i }

func TestNew_RequiresAPIVersion(t *testing.T) {
	if _, err := New(ClientOptions{AccessToken: "x"}); err == nil {
		t.Fatal("expected error when APIVersion is empty")
	}
}

func TestNew_RequiresExactlyOneAuthMode(t *testing.T) {
	if _, err := New(ClientOptions{APIVersion: testAPIVersion}); err == nil {
		t.Fatal("expected error when no auth mode is supplied")
	}
	_, err := New(ClientOptions{
		APIVersion:          testAPIVersion,
		AccessToken:         "a",
		AuthorizationHeader: "Bearer b",
	})
	if err == nil {
		t.Fatal("expected error when two auth modes are supplied")
	}
}

func TestNew_DefaultsHTTPClient_NoDefaultClient(t *testing.T) {
	c, err := New(ClientOptions{APIVersion: testAPIVersion, AccessToken: "x"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()
	if c.HTTPClient() == http.DefaultClient {
		t.Fatal("New must NOT reuse http.DefaultClient")
	}
	if c.HTTPClient().Timeout == 0 {
		t.Fatal("New must set an explicit Timeout on the http.Client")
	}
	tr, ok := c.HTTPClient().Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", c.HTTPClient().Transport)
	}
	if tr.MaxIdleConnsPerHost == 0 {
		t.Fatal("Transport must set MaxIdleConnsPerHost")
	}
	if tr.IdleConnTimeout == 0 {
		t.Fatal("Transport must set IdleConnTimeout")
	}
}

func TestNew_CustomHTTPClientReusedAsIs(t *testing.T) {
	custom := &http.Client{Timeout: time.Second}
	c, err := New(ClientOptions{APIVersion: testAPIVersion, AccessToken: "x", HTTPClient: custom})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()
	if c.HTTPClient() != custom {
		t.Fatal("custom HTTPClient should be reused as-is")
	}
}

func TestRequiredHeadersAreInjected(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+testToken {
			t.Errorf("unexpected Authorization: %q", got)
		}
		if got := r.Header.Get("X-Fanvue-API-Version"); got != testAPIVersion {
			t.Errorf("unexpected API version header: %q", got)
		}
		_, _ = io.WriteString(w, currentUserJSON)
	})
	if _, err := c.GetCurrentUser(context.Background()); err != nil {
		t.Fatalf("GetCurrentUser: %v", err)
	}
}

func TestAuthorizationHeaderMode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer direct" {
			t.Errorf("unexpected Authorization: %q", got)
		}
		_, _ = io.WriteString(w, currentUserJSON)
	}))
	defer srv.Close()
	c, err := New(ClientOptions{
		APIVersion:          testAPIVersion,
		AuthorizationHeader: "Bearer direct",
		BaseURL:             srv.URL,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()
	if _, err := c.GetCurrentUser(context.Background()); err != nil {
		t.Fatalf("GetCurrentUser: %v", err)
	}
}

func TestAuthHeaderProviderMode(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer rotated" {
			t.Errorf("unexpected Authorization: %q", got)
		}
		_, _ = io.WriteString(w, currentUserJSON)
	}))
	defer srv.Close()
	c, err := New(ClientOptions{
		APIVersion: testAPIVersion,
		BaseURL:    srv.URL,
		AuthHeaderProvider: func(ctx context.Context) (string, error) {
			calls++
			return "Bearer rotated", nil
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer c.Close()
	if _, err := c.GetCurrentUser(context.Background()); err != nil {
		t.Fatalf("GetCurrentUser: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected provider to be called once, got %d", calls)
	}
}

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		status int
		body   string
		check  func(*Error) bool
		want   string
	}{
		{http.StatusUnauthorized, `{"error":"unauthorized"}`, (*Error).IsUnauthorized, "unauthorized"},
		{http.StatusForbidden, `{"message":"forbidden scope"}`, (*Error).IsForbidden, "forbidden scope"},
		{http.StatusNotFound, `{"message":"not found"}`, (*Error).IsNotFound, "not found"},
		{http.StatusGone, `{"error":"sunset","message":"upgrade"}`, (*Error).IsGone, "upgrade"},
		{http.StatusTooManyRequests, `{"error":"rate limited"}`, (*Error).IsRateLimited, "rate limited"},
		{http.StatusInternalServerError, `{}`, (*Error).IsServerError, "Internal Server Error"},
	}
	for _, tc := range cases {
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("x-request-id", "req-xyz")
			w.WriteHeader(tc.status)
			_, _ = io.WriteString(w, tc.body)
		})
		_, err := c.GetCurrentUser(context.Background())
		apiErr, ok := AsError(err)
		if !ok {
			t.Fatalf("status %d: expected *Error, got %T: %v", tc.status, err, err)
		}
		if apiErr.StatusCode != tc.status {
			t.Errorf("status %d: got StatusCode %d", tc.status, apiErr.StatusCode)
		}
		if !tc.check(apiErr) {
			t.Errorf("status %d: predicate returned false (%+v)", tc.status, apiErr)
		}
		if apiErr.Message != tc.want {
			t.Errorf("status %d: got message %q want %q", tc.status, apiErr.Message, tc.want)
		}
		if apiErr.RequestID != "req-xyz" {
			t.Errorf("status %d: request id not propagated: %q", tc.status, apiErr.RequestID)
		}
	}
}

func TestValidationErrorMessageJoinsErrors(t *testing.T) {
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"errors":["text required","audience invalid"]}`)
	})
	_, err := c.GetCurrentUser(context.Background())
	apiErr, ok := AsError(err)
	if !ok {
		t.Fatalf("expected *Error, got %v", err)
	}
	if len(apiErr.Errors) != 2 {
		t.Fatalf("expected 2 field errors, got %v", apiErr.Errors)
	}
	if !strings.Contains(apiErr.Message, "text required") {
		t.Fatalf("expected joined message, got %q", apiErr.Message)
	}
}

func TestEncodeQueryOmitsNilAndSetsValues(t *testing.T) {
	q := encodeQuery(map[string]any{
		"page": ptrInt(2),
		"size": (*int)(nil),
		"flag": ptrString("on"),
	})
	if q.Get("page") != "2" {
		t.Errorf("expected page=2, got %q", q.Get("page"))
	}
	if _, ok := q["size"]; ok {
		t.Error("nil pointer param must be omitted")
	}
	if q.Get("flag") != "on" {
		t.Errorf("expected flag=on, got %q", q.Get("flag"))
	}
}

func decodeBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

const currentUserJSON = `{
  "uuid": "u-1",
  "email": "creator@example.com",
  "handle": "creator",
  "bio": "",
  "displayName": "Creator",
  "isCreator": true,
  "createdAt": "2026-01-01T00:00:00.000Z",
  "updatedAt": null,
  "avatarUrl": null,
  "bannerUrl": null
}`
