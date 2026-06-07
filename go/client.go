package fanvue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the production Fanvue API root.
const DefaultBaseURL = "https://api.fanvue.com"

// DefaultTimeout is the per-request timeout used when ClientOptions.Timeout is
// zero. 30s matches the Python SDK default.
const DefaultTimeout = 30 * time.Second

// apiVersionHeader is the header Fanvue uses to pin the API contract version.
const apiVersionHeader = "X-Fanvue-API-Version"

// AuthHeaderProvider returns the full Authorization header value
// (e.g. "Bearer <token>"). It is invoked on every request, enabling token
// rotation without reconstructing the client. The supplied context allows the
// provider to honour cancellation while fetching/refreshing a token.
type AuthHeaderProvider func(ctx context.Context) (string, error)

// ClientOptions configures a *Client created via New.
//
// Exactly one authentication mode must be supplied: AccessToken,
// AuthorizationHeader, or AuthHeaderProvider. APIVersion is always required.
type ClientOptions struct {
	// APIVersion is sent as the X-Fanvue-API-Version header on every request.
	// Required.
	APIVersion string

	// AccessToken is the OAuth access token. When set, the client sends
	// "Authorization: Bearer <AccessToken>".
	AccessToken string

	// AuthorizationHeader is the full Authorization header value to send
	// verbatim (e.g. "Bearer <token>"). Use when the prefix is not "Bearer".
	AuthorizationHeader string

	// AuthHeaderProvider supplies the Authorization header dynamically per
	// request, enabling token rotation.
	AuthHeaderProvider AuthHeaderProvider

	// BaseURL overrides the production Fanvue host. Useful for tests.
	BaseURL string

	// Timeout sets the per-request total timeout. Defaults to DefaultTimeout.
	Timeout time.Duration

	// HTTPClient lets callers inject a fully configured *http.Client. When nil,
	// New constructs one with bounded idle connections, an explicit Timeout,
	// and an IdleConnTimeout. The package never falls back to
	// http.DefaultClient.
	HTTPClient *http.Client

	// UserAgent overrides the User-Agent header. Defaults to
	// "inoue-fanvue-sdk-go/1".
	UserAgent string
}

// Client is the Fanvue API client. It is safe for concurrent use by multiple
// goroutines. Callers must invoke Close (or defer client.Close()) when done so
// idle TCP connections are released.
type Client struct {
	httpClient   *http.Client
	baseURL      string
	apiVersion   string
	accessToken  string
	authHeader   string
	authProvider AuthHeaderProvider
	userAgent    string
	ownsHTTP     bool
}

// New constructs a *Client with the supplied options. It returns an error when
// APIVersion is empty or when the number of auth modes supplied is not exactly
// one.
func New(opts ClientOptions) (*Client, error) {
	if strings.TrimSpace(opts.APIVersion) == "" {
		return nil, errors.New("fanvue: APIVersion is required")
	}

	authModes := 0
	if opts.AccessToken != "" {
		authModes++
	}
	if opts.AuthorizationHeader != "" {
		authModes++
	}
	if opts.AuthHeaderProvider != nil {
		authModes++
	}
	if authModes != 1 {
		return nil, errors.New(
			"fanvue: provide exactly one auth mode: AccessToken, AuthorizationHeader, or AuthHeaderProvider",
		)
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	baseURL := strings.TrimRight(opts.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	ua := opts.UserAgent
	if ua == "" {
		ua = "inoue-fanvue-sdk-go/1"
	}

	c := &Client{
		baseURL:      baseURL,
		apiVersion:   opts.APIVersion,
		accessToken:  opts.AccessToken,
		authHeader:   opts.AuthorizationHeader,
		authProvider: opts.AuthHeaderProvider,
		userAgent:    ua,
	}

	if opts.HTTPClient != nil {
		c.httpClient = opts.HTTPClient
		c.ownsHTTP = false
	} else {
		c.httpClient = &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   20,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				ResponseHeaderTimeout: timeout,
			},
		}
		c.ownsHTTP = true
	}
	return c, nil
}

// Close releases idle TCP connections held by the underlying *http.Client. It
// is safe to call multiple times. When the caller injected their own
// *http.Client, Close is a no-op for that client.
func (c *Client) Close() error {
	if c == nil || c.httpClient == nil {
		return nil
	}
	if c.ownsHTTP {
		if t, ok := c.httpClient.Transport.(*http.Transport); ok {
			t.CloseIdleConnections()
		}
	}
	return nil
}

// HTTPClient returns the underlying *http.Client. Exposed for advanced
// instrumentation only; callers should not mutate the returned client.
func (c *Client) HTTPClient() *http.Client { return c.httpClient }

// authorizationHeader resolves the Authorization header value for a request,
// honouring whichever auth mode the client was constructed with.
func (c *Client) authorizationHeader(ctx context.Context) (string, error) {
	switch {
	case c.accessToken != "":
		return "Bearer " + c.accessToken, nil
	case c.authHeader != "":
		return c.authHeader, nil
	case c.authProvider != nil:
		header, err := c.authProvider(ctx)
		if err != nil {
			return "", fmt.Errorf("fanvue: auth header provider: %w", err)
		}
		if strings.TrimSpace(header) == "" {
			return "", errors.New("fanvue: auth header provider returned an empty string")
		}
		return header, nil
	default:
		return "", errors.New("fanvue: no auth mode configured")
	}
}

// doJSON performs an authenticated request and decodes a JSON response body
// into out (which may be nil for 204/empty responses). It returns a *Error for
// any non-2xx status.
func (c *Client) doJSON(
	ctx context.Context,
	method, path string,
	query url.Values,
	body any,
	out any,
) error {
	authHeader, err := c.authorizationHeader(ctx)
	if err != nil {
		return err
	}

	fullURL := c.baseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		buf, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return fmt.Errorf("fanvue: marshal request body: %w", marshalErr)
		}
		bodyReader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("fanvue: build request: %w", err)
	}
	req.Header.Set("Authorization", authHeader)
	req.Header.Set(apiVersionHeader, c.apiVersion)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fanvue: %s %s: %w", method, fullURL, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("fanvue: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return parseAPIError(resp, raw)
	}

	if out == nil || len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return &Error{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("decode response: %v", err),
			RequestID:  resp.Header.Get("x-request-id"),
			Body:       raw,
		}
	}
	return nil
}

// parseAPIError converts a non-2xx response into a *Error, extracting the
// human-readable message from whichever field the endpoint used.
func parseAPIError(resp *http.Response, raw []byte) *Error {
	apiErr := &Error{
		StatusCode: resp.StatusCode,
		RequestID:  resp.Header.Get("x-request-id"),
		Body:       raw,
	}

	var payload struct {
		Message string   `json:"message"`
		Error   string   `json:"error"`
		Detail  string   `json:"detail"`
		Title   string   `json:"title"`
		Errors  []string `json:"errors"`
	}
	if err := json.Unmarshal(raw, &payload); err == nil {
		apiErr.Errors = payload.Errors
		switch {
		case payload.Message != "":
			apiErr.Message = payload.Message
		case payload.Error != "":
			apiErr.Message = payload.Error
		case payload.Detail != "":
			apiErr.Message = payload.Detail
		case payload.Title != "":
			apiErr.Message = payload.Title
		case len(payload.Errors) > 0:
			apiErr.Message = strings.Join(payload.Errors, "; ")
		}
	}

	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(resp.StatusCode)
		if apiErr.Message == "" {
			apiErr.Message = "request failed"
		}
	}
	return apiErr
}

// encodeQuery builds a url.Values, omitting any entries whose value is nil so
// that optional parameters are not sent. Integer pointers are rendered without
// scientific notation.
func encodeQuery(params map[string]any) url.Values {
	q := url.Values{}
	for key, value := range params {
		switch v := value.(type) {
		case nil:
			continue
		case string:
			if v != "" {
				q.Set(key, v)
			}
		case *string:
			if v != nil {
				q.Set(key, *v)
			}
		case *int:
			if v != nil {
				q.Set(key, fmt.Sprintf("%d", *v))
			}
		case *int64:
			if v != nil {
				q.Set(key, fmt.Sprintf("%d", *v))
			}
		case *bool:
			if v != nil {
				q.Set(key, fmt.Sprintf("%t", *v))
			}
		default:
			q.Set(key, fmt.Sprintf("%v", v))
		}
	}
	return q
}
