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

// Do performs an HTTP request against the API.
func (c *Client) Do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	return c.DoWithHeaders(ctx, method, path, body, nil)
}

// DoWithHeaders performs an HTTP request with additional headers.
func (c *Client) DoWithHeaders(ctx context.Context, method, path string, body any, extra http.Header) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			delay := c.backoff[len(c.backoff)-1]
			if attempt-1 < len(c.backoff) {
				delay = c.backoff[attempt-1]
			}
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, err := c.doOnce(ctx, method, path, body, extra)
		if err != nil {
			lastErr = err
			if !isRetryableError(err) {
				return nil, err
			}
			continue
		}

		if !isRetryableStatus(resp.StatusCode) {
			return resp, nil
		}

		// Drain and close body before retry to allow connection reuse.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		lastErr = &Error{StatusCode: resp.StatusCode}
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
			req.Header.Set("Cookie", "semaphore="+c.token)
		} else {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
	}
	for k, vv := range extra {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}

	return resp, nil
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

// DecodeJSON reads and decodes a JSON response body.
func DecodeJSON(resp *http.Response, dest any) error {
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return &Error{StatusCode: resp.StatusCode, Body: body}
	}
	if dest == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

// Error represents an API error response.
type Error struct {
	StatusCode int
	Body       []byte
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
