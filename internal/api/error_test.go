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
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckResponseErrorCarriesMethodPathAndRequestID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req_abc123")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok")
	resp, err := c.Do(context.Background(), http.MethodGet, "/project/1/tasks/last", nil)
	if err != nil {
		t.Fatalf("Do returned transport error: %v", err)
	}
	err = CheckResponse(resp)

	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *api.Error, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 404 {
		t.Fatalf("status: %d", apiErr.StatusCode)
	}
	if apiErr.Method != http.MethodGet {
		t.Fatalf("method: %q", apiErr.Method)
	}
	// Path must be the logical path the caller used, without the /api prefix.
	if apiErr.Path != "/project/1/tasks/last" {
		t.Fatalf("path: %q", apiErr.Path)
	}
	if apiErr.RequestID != "req_abc123" {
		t.Fatalf("request id: %q", apiErr.RequestID)
	}
}

func TestRetryExhaustedErrorCapturesRetryAfter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests) // retryable; exhausts immediately
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok").WithRetryPolicy(0, nil)
	_, err := c.Do(context.Background(), http.MethodGet, "/projects", nil)
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *api.Error, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 429 {
		t.Fatalf("status: %d", apiErr.StatusCode)
	}
	if apiErr.RetryAfter != "30" {
		t.Fatalf("retry-after: %q", apiErr.RetryAfter)
	}
}
