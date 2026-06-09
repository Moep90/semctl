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
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	c := NewClient("https://semaphore.example.com", "token123")
	if c.baseURL != "https://semaphore.example.com" {
		t.Fatalf("unexpected baseURL: %s", c.baseURL)
	}
	if c.token != "token123" {
		t.Fatal("token mismatch")
	}
}

func TestNewClientTrailingSlash(t *testing.T) {
	c := NewClient("https://semaphore.example.com/", "tok")
	if c.baseURL != "https://semaphore.example.com" {
		t.Fatalf("unexpected baseURL after trim: %s", c.baseURL)
	}
}

func TestDoGET(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer tok" {
			t.Fatalf("unexpected auth header: %s", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "name": "infra"}})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	resp, err := c.Do(context.Background(), http.MethodGet, "/projects", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

func TestDoPOST(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("invalid json body: %v", err)
		}
		if payload["template_id"] != float64(7) {
			t.Fatalf("unexpected payload: %v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 42})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	resp, err := c.Do(context.Background(), http.MethodPost, "/tasks", map[string]any{"template_id": 7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

func TestDoWithHeaders(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		custom := r.Header.Get("X-Custom-Header")
		if custom != "hello" {
			t.Fatalf("expected X-Custom-Header=hello, got %s", custom)
		}
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	extra := http.Header{}
	extra.Set("X-Custom-Header", "hello")
	resp, err := c.DoWithHeaders(context.Background(), http.MethodGet, "/tasks", nil, extra)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

func TestErrorTruncation(t *testing.T) {
	longBody := make([]byte, 1000)
	for i := range longBody {
		longBody[i] = 'x'
	}
	e := &Error{StatusCode: 500, Body: longBody}
	msg := e.Error()
	if len(msg) > 250 {
		t.Fatalf("error message too long (%d chars), expected truncation", len(msg))
	}
	if !contains(msg, "xxx") {
		t.Fatalf("expected truncated preview in error, got: %s", msg)
	}
	if contains(msg, string(longBody)) {
		t.Fatal("error message should not contain full body")
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestRetryOnServerError(t *testing.T) {
	var count atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		if count.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"overloaded"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 42})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(srv.URL, "tok").WithRetryPolicy(3, []time.Duration{10 * time.Millisecond, 10 * time.Millisecond, 10 * time.Millisecond})
	resp, err := c.Do(context.Background(), http.MethodGet, "/tasks", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if count.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", count.Load())
	}
}

func TestNoRetryOnClientError(t *testing.T) {
	var count atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(srv.URL, "tok").WithRetryPolicy(3, []time.Duration{10 * time.Millisecond, 10 * time.Millisecond, 10 * time.Millisecond})
	resp, err := c.Do(context.Background(), http.MethodGet, "/tasks", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if count.Load() != 1 {
		t.Fatalf("expected 1 attempt, got %d", count.Load())
	}
}

func TestMaxRetriesExceeded(t *testing.T) {
	var count atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(srv.URL, "tok").WithRetryPolicy(2, []time.Duration{10 * time.Millisecond, 10 * time.Millisecond})
	_, err := c.Do(context.Background(), http.MethodGet, "/tasks", nil)
	if err == nil {
		t.Fatal("expected error after max retries")
	}
	if count.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", count.Load())
	}
}

func TestBackoffBoundsCheck(t *testing.T) {
	var count atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// maxRetries=5 but only 1 backoff duration — should not panic.
	c := NewClient(srv.URL, "tok").WithRetryPolicy(5, []time.Duration{1 * time.Millisecond})
	_, err := c.Do(context.Background(), http.MethodGet, "/tasks", nil)
	if err == nil {
		t.Fatal("expected error after max retries")
	}
	if count.Load() != 6 {
		t.Fatalf("expected 6 attempts, got %d", count.Load())
	}
}

func TestIsRetryableError(t *testing.T) {
	if isRetryableError(nil) {
		t.Error("nil should not be retryable")
	}
	if isRetryableError(context.Canceled) {
		t.Error("context.Canceled should not be retryable")
	}
	if isRetryableError(context.DeadlineExceeded) {
		t.Error("context.DeadlineExceeded should not be retryable")
	}
	if !isRetryableError(&url.Error{Err: &net.DNSError{IsTimeout: true}}) {
		t.Error("timeout URL error should be retryable")
	}
	if isRetryableError(errors.New("some random error")) {
		t.Error("random error should not be retryable")
	}
}

func TestDebugOutput(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	var buf bytes.Buffer
	c := NewClient(srv.URL, "tok").WithDebug(&buf)
	resp, err := c.Do(context.Background(), "GET", "/ping", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	out := buf.String()
	if !strings.Contains(out, "[debug]") {
		t.Fatalf("expected debug output, got: %s", out)
	}
	if !strings.Contains(out, "auth: using bearer token source") {
		t.Fatalf("expected auth debug, got: %s", out)
	}
}

func TestDebugOutputRedactsAuthorization(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	var buf bytes.Buffer
	c := NewClient(srv.URL, "super-secret-token").WithDebug(&buf)
	resp, err := c.Do(context.Background(), "GET", "/ping", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	out := buf.String()
	if strings.Contains(out, "super-secret-token") {
		t.Fatalf("Authorization token should be redacted in debug output, got: %s", out)
	}
	if !strings.Contains(out, "***REDACTED***") {
		t.Fatalf("expected ***REDACTED*** in debug output, got: %s", out)
	}
}

func TestVerboseOutput(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	var buf bytes.Buffer
	c := NewClient(srv.URL, "tok").WithVerbose(&buf)
	resp, err := c.Do(context.Background(), "GET", "/ping", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	out := buf.String()
	if !strings.Contains(out, "[verbose]") {
		t.Fatalf("expected verbose output, got: %s", out)
	}
	if !strings.Contains(out, "request: GET") {
		t.Fatalf("expected request verbose, got: %s", out)
	}
	if !strings.Contains(out, "→ 200") {
		t.Fatalf("expected response verbose, got: %s", out)
	}
}

func TestDecodeJSONError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	resp, err := c.Do(context.Background(), http.MethodGet, "/missing", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var dest map[string]any
	if err := DecodeJSON(resp, &dest); err == nil {
		t.Fatal("expected error for 404")
	} else {
		e, ok := err.(*Error)
		if !ok {
			t.Fatalf("expected *Error, got %T", err)
		}
		if e.StatusCode != http.StatusNotFound {
			t.Fatalf("unexpected status code: %d", e.StatusCode)
		}
	}
}

func TestClient_Get_Post_Put_Delete(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/items", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"action": "get"})
		case http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			var payload map[string]any
			_ = json.Unmarshal(body, &payload)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"action": "post", "id": payload["id"]})
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			var payload map[string]any
			_ = json.Unmarshal(body, &payload)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"action": "put", "id": payload["id"]})
		case http.MethodDelete:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"action": "delete"})
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(srv.URL, "tok")

	// GET
	resp, err := c.Get(context.Background(), "/items")
	if err != nil {
		t.Fatalf("unexpected GET error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected GET status: %d", resp.StatusCode)
	}

	// POST
	resp, err = c.Post(context.Background(), "/items", map[string]any{"id": 1})
	if err != nil {
		t.Fatalf("unexpected POST error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected POST status: %d", resp.StatusCode)
	}

	// PUT
	resp, err = c.Put(context.Background(), "/items", map[string]any{"id": 2})
	if err != nil {
		t.Fatalf("unexpected PUT error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected PUT status: %d", resp.StatusCode)
	}

	// DELETE
	resp, err = c.Delete(context.Background(), "/items")
	if err != nil {
		t.Fatalf("unexpected DELETE error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected DELETE status: %d", resp.StatusCode)
	}
}

func TestClient_FetchAllPages(t *testing.T) {
	var pages atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/items", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("page")
		page := 1
		if q != "" {
			var err error
			page, err = strconv.Atoi(q)
			if err != nil {
				t.Fatalf("invalid page param: %s", q)
			}
		}
		pages.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if page < 3 {
			w.Header().Set("Link", `<http://example.com/api/items?page=2>; rel="next"`)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{"page": page}})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	var dest []map[string]any
	if err := c.FetchAllPages(context.Background(), "/items", &dest); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pages.Load() != 3 {
		t.Fatalf("expected 3 pages, got %d", pages.Load())
	}
	if len(dest) != 3 {
		t.Fatalf("expected 3 dest items, got %d", len(dest))
	}
}

func TestClient_IsAuthenticated_true_false(t *testing.T) {
	tests := []struct {
		name string
		tok  string
		want bool
	}{
		{"empty token", "", false},
		{"non-empty token", "abc123", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient("https://example.com", tt.tok)
			if got := c.IsAuthenticated(); got != tt.want {
				t.Fatalf("IsAuthenticated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_WithRetryPolicy_custom_backoff(t *testing.T) {
	var count atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/items", func(w http.ResponseWriter, r *http.Request) {
		if count.Add(1) < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	customBackoff := []time.Duration{5 * time.Millisecond, 5 * time.Millisecond}
	c := NewClient(srv.URL, "tok").WithRetryPolicy(2, customBackoff)
	resp, err := c.Do(context.Background(), http.MethodGet, "/items", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
	if count.Load() != 2 {
		t.Fatalf("expected 2 attempts, got %d", count.Load())
	}
}

func TestClient_Token_and_BaseURL_getters(t *testing.T) {
	c := NewClient("https://example.com/api/", "my-token")
	if c.Token() != "my-token" {
		t.Fatalf("Token() = %q, want %q", c.Token(), "my-token")
	}
	if c.BaseURL() != "https://example.com/api" {
		t.Fatalf("BaseURL() = %q, want %q", c.BaseURL(), "https://example.com/api")
	}
}

func TestDecodeJSON_nil_dest(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	resp, err := c.Do(context.Background(), http.MethodGet, "/ping", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// nil dest should not panic and return nil
	if err := DecodeJSON(resp, nil); err != nil {
		t.Fatalf("unexpected error for nil dest: %v", err)
	}
}

func TestRedactHeaders_partial(t *testing.T) {
	h := http.Header{}
	h.Set("Authorization", "Bearer secret")
	h.Set("Cookie", "session=abc")
	h.Set("X-Custom", "hello")
	h.Set("Content-Type", "application/json")

	safe := redactHeaders(h)
	if safe.Get("Authorization") != "***REDACTED***" {
		t.Fatalf("expected Authorization redacted, got %q", safe.Get("Authorization"))
	}
	if safe.Get("Cookie") != "***REDACTED***" {
		t.Fatalf("expected Cookie redacted, got %q", safe.Get("Cookie"))
	}
	if safe.Get("X-Custom") != "hello" {
		t.Fatalf("expected X-Custom preserved, got %q", safe.Get("X-Custom"))
	}
	if safe.Get("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type preserved, got %q", safe.Get("Content-Type"))
	}
}

func BenchmarkClient_Do(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := c.Do(ctx, http.MethodGet, "/ping", nil)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
		_ = resp.Body.Close()
	}
}

func FuzzIsRetryableStatus(f *testing.F) {
	f.Add(200)
	f.Add(404)
	f.Add(429)
	f.Add(500)
	f.Add(502)
	f.Add(503)
	f.Add(504)
	f.Fuzz(func(t *testing.T, code int) {
		result := isRetryableStatus(code)
		// fuzz test ensures the function doesn't panic; no fixed assertion
		_ = result
	})
}
