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

package testutil

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestMockServer_Expect_returns_response(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	ms.Expect(http.MethodGet, "/api/v1/projects", http.StatusOK, "hello")

	resp, err := http.Get(ms.URL() + "/api/v1/projects")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello" {
		t.Fatalf("expected body 'hello', got %s", string(body))
	}
}

func TestMockServer_ExpectJSON(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	payload := map[string]any{"id": 42, "name": "test"}
	ms.ExpectJSON(http.MethodPost, "/api/v1/items", http.StatusCreated, payload)

	resp, err := http.Post(ms.URL()+"/api/v1/items", "application/json", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if got["id"] != float64(42) || got["name"] != "test" {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestMockServer_AssertCalled(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	ms.Expect(http.MethodGet, "/api/ping", http.StatusOK, "pong")

	_, err := http.Get(ms.URL() + "/api/ping")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ms.AssertCalled(t, http.MethodGet, "/api/ping")
}

func TestMockServer_Reset(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	ms.Expect(http.MethodGet, "/api/ping", http.StatusOK, "pong")
	_, _ = http.Get(ms.URL() + "/api/ping")

	ms.Reset()

	// After reset the expectation is gone, so the same request should 404.
	resp, err := http.Get(ms.URL() + "/api/ping")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after reset, got %d", resp.StatusCode)
	}

	// AssertCalled should fail for a cleared call log.
	fake := &fakeT{T: t}
	ms.AssertCalled(fake, http.MethodGet, "/api/ping")
	if !fake.failed {
		t.Fatal("expected AssertCalled to fail after Reset")
	}
}

func TestMockServer_Reset_keeps_calls_after_reset(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	ms.Expect(http.MethodGet, "/api/ping", http.StatusOK, "pong")
	_, _ = http.Get(ms.URL() + "/api/ping")

	ms.Reset()
	// Do NOT make another request — the call log was cleared by Reset.

	fake := &fakeT{T: t}
	ms.AssertCalled(fake, http.MethodGet, "/api/ping")
	if !fake.failed {
		t.Fatal("expected AssertCalled to fail because Reset cleared the call log")
	}
}

func TestMockServer_unmatched_returns_404(t *testing.T) {
	ms := NewMockServer()
	defer ms.Close()

	resp, err := http.Get(ms.URL() + "/unknown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// fakeT is a wrapper around *testing.T that tracks whether Fatalf was called.
// It does NOT implement testing.TB; it is used to observe AssertCalled behavior
// by overriding Fatalf on an embedded testing.T.
type fakeT struct {
	*testing.T
	failed bool
}

func (f *fakeT) Fatalf(format string, args ...any) {
	f.failed = true
}
