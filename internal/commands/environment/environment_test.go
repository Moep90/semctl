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

package environment

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/testutil"
)

func TestListCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/project/1/environment", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Environment{
			{ID: 1, Name: "staging"},
			{ID: 2, Name: "production"},
		})
	})
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewEnvironmentCommand(), "environment", "list", "--host", srv.URL, "--project", "infra")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "staging") {
		t.Fatalf("expected staging in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "production") {
		t.Fatalf("expected production in output, got: %s", stdout)
	}
}

func TestGetCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	mux.HandleFunc("/api/project/1/environment", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Environment{
			{ID: 1, Name: "staging"},
			{ID: 2, Name: "production"},
		})
	})
	mux.HandleFunc("/api/project/1/environment/2", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(api.Environment{ID: 2, Name: "production", JSON: `{}`})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewEnvironmentCommand(), "environment", "get", "production", "--host", srv.URL, "--project", "infra")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "production") {
		t.Fatalf("expected production in output, got: %s", stdout)
	}
}

func TestCreateCommand(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()
	srv.Expect("POST", "/api/project/2/environment", 201, "{}")

	stdout, _, err := testutil.RunCommand(t, NewEnvironmentCommand(), "environment", "create", "--name", "staging", "--host", srv.URL(), "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Created environment staging") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
	srv.AssertCalled(t, "POST", "/api/project/2/environment")
}

func TestCreateCommandBody(t *testing.T) {
	var got map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/api/project/2/environment", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{}"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, _, err := testutil.RunCommand(t, NewEnvironmentCommand(), "environment", "create",
		"--name", "staging", "--json", `{"KEY":"value"}`, "--host", srv.URL, "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["name"] != "staging" {
		t.Fatalf("expected name=staging, got: %v", got["name"])
	}
	if got["project_id"] != float64(2) {
		t.Fatalf("expected project_id=2, got: %v", got["project_id"])
	}
	if got["json"] != `{"KEY":"value"}` {
		t.Fatalf("expected json env string, got: %v", got["json"])
	}
}

func TestUpdateCommand(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()
	srv.ExpectJSON("GET", "/api/project/2/environment", 200, []api.Environment{{ID: 5, Name: "staging"}})
	srv.Expect("PUT", "/api/project/2/environment/5", 204, "")

	stdout, _, err := testutil.RunCommand(t, NewEnvironmentCommand(), "environment", "update", "staging", "--json", `{"K":"v"}`, "--host", srv.URL(), "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Updated environment staging") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
	srv.AssertCalled(t, "PUT", "/api/project/2/environment/5")
}

func TestDeleteCommand(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()
	srv.ExpectJSON("GET", "/api/project/2/environment", 200, []api.Environment{{ID: 5, Name: "staging"}})
	srv.Expect("DELETE", "/api/project/2/environment/5", 204, "")

	stdout, _, err := testutil.RunCommand(t, NewEnvironmentCommand(), "environment", "delete", "staging", "--host", srv.URL(), "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Deleted environment staging") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
	srv.AssertCalled(t, "DELETE", "/api/project/2/environment/5")
}
