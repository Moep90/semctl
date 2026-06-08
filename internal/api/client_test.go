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
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
