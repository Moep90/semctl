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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/moep90/semaphore-cli/internal/testutil"
)

func TestAPIGet(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 1, "name": "infra"}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewAPICommand(), "api", "GET", "/projects", "--host", srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "infra") {
		t.Fatalf("expected infra in output, got: %s", stdout)
	}
}

func TestAPIPostWithFields(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["message"] != "hello" {
			t.Fatalf("unexpected message: %v", body["message"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 42})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewAPICommand(), "api", "POST", "/tasks", "--host", srv.URL, "-f", "message=hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "42") {
		t.Fatalf("expected id in output, got: %s", stdout)
	}
}

func TestAPIPostWithTypedFields(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["count"] != float64(5) {
			t.Fatalf("expected count=5 (float64), got %v (type %T)", body["count"], body["count"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 99})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, _, err := testutil.RunCommand(t, NewAPICommand(), "api", "POST", "/tasks", "--host", srv.URL, "-F", "count=5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAPIPostWithBooleanField(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/project/1/tasks", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		// -F dry_run=true must arrive as a JSON boolean, not the string "true"
		// (issue #74).
		if dr, ok := body["dry_run"].(bool); !ok || !dr {
			t.Fatalf("expected dry_run=true (bool), got %v (%T)", body["dry_run"], body["dry_run"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, _, err := testutil.RunCommand(t, NewAPICommand(), "api", "POST", "/project/1/tasks", "--host", srv.URL, "-F", "template_id=1", "-F", "dry_run=true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAPIErrorResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// --output json must NOT suppress the exit code: an HTTP >= 400 response has
	// to surface as a non-nil error so the process exits non-zero (issue #73).
	// The JSON formatting of the error is handled by the top-level error
	// renderer, which writes to stderr — never to stdout.
	stdout, _, err := testutil.RunCommand(t, NewAPICommand(), "api", "GET", "/missing", "--host", srv.URL, "--output", "json")
	if err == nil {
		t.Fatalf("expected error for HTTP 404 with --output json, got nil (stdout=%q)", stdout)
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("expected 404 in error, got: %v", err)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("error must not be written to stdout, got: %q", stdout)
	}
}

func TestAPIInvalidHeader(t *testing.T) {
	_, _, err := testutil.RunCommand(t, NewAPICommand(), "api", "GET", "/projects", "--host", "http://example.com", "--header", ":")
	if err == nil {
		t.Fatal("expected error for invalid header")
	}
	if !strings.Contains(err.Error(), "invalid header") {
		t.Fatalf("expected invalid header error, got: %v", err)
	}
}
