// Copyright 2026 The semctl authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

// Client is a Semaphore UI API client.
type Client struct {
	baseURL     string
	token       string
	tokenSource string
	httpClient  *http.Client
	maxRetries  int
	backoff     []time.Duration
	debug       bool
	debugOut    io.Writer
	verbose     bool
	verboseOut  io.Writer
}

// NewClient creates a new API client.
func NewClient(baseURL, token string) *Client {
	return NewClientWithSource(baseURL, token, "bearer")
}

// NewClientWithSource creates a new API client with a specific token source.
func NewClientWithSource(baseURL, token, tokenSource string) *Client {
	return &Client{
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		token:       token,
		tokenSource: tokenSource,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxRetries: 3,
		backoff:    []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second},
	}
}

// WithHTTPClient overrides the default HTTP client.
func (c *Client) WithHTTPClient(hc *http.Client) *Client {
	c.httpClient = hc
	return c
}

// WithRetryPolicy overrides the default retry settings.
func (c *Client) WithRetryPolicy(maxRetries int, backoff []time.Duration) *Client {
	c.maxRetries = maxRetries
	c.backoff = backoff
	return c
}

// WithDebug enables debug logging to the given writer.
func (c *Client) WithDebug(w io.Writer) *Client {
	c.debug = true
	c.debugOut = w
	return c
}

func (c *Client) debugf(format string, args ...any) {
	if c.debug && c.debugOut != nil {
		_, _ = fmt.Fprintf(c.debugOut, "[debug] "+format+"\n", args...)
	}
}

// WithVerbose enables verbose logging to the given writer.
func (c *Client) WithVerbose(w io.Writer) *Client {
	c.verbose = true
	c.verboseOut = w
	return c
}

func (c *Client) verbosef(format string, args ...any) {
	if c.verbose && c.verboseOut != nil {
		_, _ = fmt.Fprintf(c.verboseOut, "[verbose] "+format+"\n", args...)
	}
}

// WithBaseURL overrides the default base URL.
func (c *Client) WithBaseURL(baseURL string) *Client {
	c.baseURL = strings.TrimSuffix(baseURL, "/")
	return c
}

// WithToken overrides the default token.
func (c *Client) WithToken(token string) *Client {
	c.token = token
	return c
}

// Token returns the configured token.
func (c *Client) Token() string {
	return c.token
}

// BaseURL returns the configured base URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// IsAuthenticated returns true if the client has a non-empty token.
func (c *Client) IsAuthenticated() bool {
	return c.token != ""
}

// Get performs an HTTP GET request against the API.
func (c *Client) Get(ctx context.Context, path string) (*http.Response, error) {
	return c.Do(ctx, http.MethodGet, path, nil)
}

// Post performs an HTTP POST request against the API.
func (c *Client) Post(ctx context.Context, path string, body any) (*http.Response, error) {
	return c.Do(ctx, http.MethodPost, path, body)
}

// Put performs an HTTP PUT request against the API.
func (c *Client) Put(ctx context.Context, path string, body any) (*http.Response, error) {
	return c.Do(ctx, http.MethodPut, path, body)
}

// Delete performs an HTTP DELETE request against the API.
func (c *Client) Delete(ctx context.Context, path string) (*http.Response, error) {
	return c.Do(ctx, http.MethodDelete, path, nil)
}

// Do performs an HTTP request against the API.
func (c *Client) Do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	return c.DoWithHeaders(ctx, method, path, body, nil)
}

// FetchAllPages performs a paginated request and returns all pages by
// automatically following Link headers and appending to dest.
func (c *Client) FetchAllPages(ctx context.Context, path string, dest any) error {
	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Pointer || rv.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to a slice")
	}
	elemType := rv.Elem().Type().Elem()
	page := 1
	for {
		resp, err := c.Do(ctx, http.MethodGet, fmt.Sprintf("%s?page=%d", path, page), nil)
		if err != nil {
			return err
		}
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return newError(resp, body)
		}
		pageSlice := reflect.New(reflect.SliceOf(elemType)).Interface()
		if err := json.NewDecoder(resp.Body).Decode(pageSlice); err != nil {
			_ = resp.Body.Close()
			return err
		}
		_ = resp.Body.Close()
		pageSliceVal := reflect.ValueOf(pageSlice).Elem()
		destSlice := rv.Elem()
		for i := 0; i < pageSliceVal.Len(); i++ {
			destSlice = reflect.Append(destSlice, pageSliceVal.Index(i))
		}
		rv.Elem().Set(destSlice)
		link := resp.Header.Get("Link")
		if !strings.Contains(link, `rel="next"`) {
			break
		}
		page++
	}
	return nil
}

// DoWithHeaders performs an HTTP request with additional headers.
func (c *Client) DoWithHeaders(ctx context.Context, method, path string, body any, extra http.Header) (*http.Response, error) {
	c.debugf("DoWithHeaders method=%s path=%s", method, path)
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			delay := c.backoff[len(c.backoff)-1]
			if attempt-1 < len(c.backoff) {
				delay = c.backoff[attempt-1]
			}
			c.debugf("retry attempt %d after %v", attempt, delay)
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, err := c.doOnce(ctx, method, path, body, extra)
		if err != nil {
			lastErr = err
			c.debugf("request error: %v", err)
			c.verbosef("request failed: %v", err)
			if !isRetryableError(err) {
				return nil, err
			}
			continue
		}

		c.debugf("response status: %d", resp.StatusCode)
		c.verbosef("response: %s %s → %d", method, path, resp.StatusCode)
		if !isRetryableStatus(resp.StatusCode) {
			return resp, nil
		}
		c.verbosef("retrying %s %s (status %d)", method, path, resp.StatusCode)

		// Drain and close body before retry to allow connection reuse.
		_, _ = io.Copy(io.Discard, resp.Body)
		reqID := requestID(resp.Header)
		retryAfter := resp.Header.Get("Retry-After")
		_ = resp.Body.Close()
		lastErr = &Error{StatusCode: resp.StatusCode, Method: method, Path: pathOnlyForError(path), RequestID: reqID, RetryAfter: retryAfter}
	}
	return nil, fmt.Errorf("request failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

func (c *Client) doOnce(ctx context.Context, method, path string, body any, extra http.Header) (*http.Response, error) {
	// Separate query string from path before joining.
	pathOnly := path
	query := ""
	if idx := strings.Index(path, "?"); idx >= 0 {
		pathOnly = path[:idx]
		query = path[idx:]
	}
	u, err := url.JoinPath(c.baseURL, "/api", pathOnly)
	if err != nil {
		return nil, fmt.Errorf("build url: %w", err)
	}
	u += query
	c.verbosef("request: %s %s", method, u)

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		if c.tokenSource == "cookie" {
			c.debugf("auth: using cookie token source")
			req.Header.Set("Cookie", "semaphore="+c.token)
		} else {
			c.debugf("auth: using bearer token source")
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
	} else {
		c.debugf("auth: no token configured")
	}
	for k, vv := range extra {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}
	c.debugf("request %s %s headers=%v", method, u, redactHeaders(req.Header))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	return resp, nil
}

func redactHeaders(h http.Header) http.Header {
	safe := make(http.Header, len(h))
	for k, vv := range h {
		lower := strings.ToLower(k)
		if lower == "authorization" || lower == "cookie" {
			safe[k] = []string{"***REDACTED***"}
		} else {
			safe[k] = vv
		}
	}
	return safe
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	// Never retry context cancellation or deadline exceeded.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	// Network-level errors are generally retryable.
	if urlErr, ok := err.(*url.Error); ok {
		return urlErr.Temporary() || urlErr.Timeout()
	}
	return false
}

func isRetryableStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests, // 429
		http.StatusRequestTimeout,     // 408
		http.StatusBadGateway,         // 502
		http.StatusServiceUnavailable, // 503
		http.StatusGatewayTimeout:     // 504
		return true
	default:
		return code >= 500
	}
}

// CheckResponse returns an *Error when the response status is >= 400 and nil
// otherwise. It always closes the response body, so it is intended for mutating
// requests (POST/PUT/DELETE) whose body is not otherwise consumed. Without this
// check, callers that only inspect the returned error miss HTTP 4xx responses,
// because Client.Do only returns a Go error for transport failures and
// retry-exhausted 5xx — not for 4xx.
func CheckResponse(resp *http.Response) error {
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return newError(resp, body)
	}
	return nil
}

// DecodeJSON reads and decodes a JSON response body.
func DecodeJSON(resp *http.Response, dest any) error {
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return newError(resp, body)
	}
	if dest == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

// Error represents an API error response. Method, Path, and RequestID are
// populated for diagnostics and structured error classification; Body is for
// debug/cause only and must never be surfaced in user-facing output.
type Error struct {
	StatusCode int
	Body       []byte
	Method     string
	Path       string
	RequestID  string
	RetryAfter string
}

// newError builds an *Error from a response, recovering the logical request
// path (without the "/api" prefix the client adds) and a request id header.
func newError(resp *http.Response, body []byte) *Error {
	e := &Error{StatusCode: resp.StatusCode, Body: body}
	if resp.Request != nil {
		e.Method = resp.Request.Method
		if resp.Request.URL != nil {
			e.Path = strings.TrimPrefix(resp.Request.URL.Path, "/api")
		}
	}
	e.RequestID = requestID(resp.Header)
	e.RetryAfter = resp.Header.Get("Retry-After")
	return e
}

// pathOnlyForError strips any query string from a logical request path so the
// error metadata never carries query parameters (which may include secrets).
func pathOnlyForError(path string) string {
	if i := strings.Index(path, "?"); i >= 0 {
		return path[:i]
	}
	return path
}

// requestID returns the first present request-id style header.
func requestID(h http.Header) string {
	for _, k := range []string{"X-Request-Id", "X-Request-ID", "Request-Id", "X-Correlation-Id"} {
		if v := h.Get(k); v != "" {
			return v
		}
	}
	return ""
}

func (e *Error) Error() string {
	if len(e.Body) == 0 {
		return fmt.Sprintf("api error %d", e.StatusCode)
	}
	preview := string(e.Body)
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	return fmt.Sprintf("api error %d: %s", e.StatusCode, preview)
}

// BodyString returns the raw response body for debugging.
func (e *Error) BodyString() string {
	return string(e.Body)
}
