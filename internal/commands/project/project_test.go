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

package project

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/config"
	"github.com/moep90/semaphore-cli/internal/testutil"
)

func TestListCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{
			{ID: 1, Name: "infra", MaxParallelTasks: 5},
			{ID: 2, Name: "app", MaxParallelTasks: 3},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewProjectCommand(), "project", "list", "--host", srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "infra") {
		t.Fatalf("expected infra in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "app") {
		t.Fatalf("expected app in output, got: %s", stdout)
	}
}

func TestListCommandJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{
			{ID: 1, Name: "infra"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewProjectCommand(), "project", "list", "--host", srv.URL, "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stdout) == 0 {
		t.Fatal("empty output")
	}
	var out []map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid json output (%q): %v", stdout, err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 project, got %d", len(out))
	}
}

func TestListCommandJSONFlag(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{
			{ID: 1, Name: "infra"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewProjectCommand(), "project", "list", "--host", srv.URL, "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stdout) == 0 {
		t.Fatal("empty output")
	}
	var out []map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid json output (%q): %v", stdout, err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 project, got %d", len(out))
	}
}

func TestListCommandOutputOverridesJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{
			{ID: 1, Name: "infra"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// --output yaml should win over --json when explicitly set.
	stdout, _, err := testutil.RunCommand(t, NewProjectCommand(), "project", "list", "--host", srv.URL, "--json", "--output", "yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "infra") {
		t.Fatalf("expected infra in output, got: %s", stdout)
	}
	if strings.HasPrefix(strings.TrimSpace(stdout), "[") {
		t.Fatalf("expected YAML output, got JSON: %s", stdout)
	}
}

func TestGetCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{
			{ID: 1, Name: "infra"},
			{ID: 2, Name: "app"},
		})
	})
	mux.HandleFunc("/api/project/2", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(api.Project{ID: 2, Name: "app", MaxParallelTasks: 3})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewProjectCommand(), "project", "get", "app", "--host", srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "app") {
		t.Fatalf("expected app in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "max_parallel_tasks") {
		t.Fatalf("expected max_parallel_tasks in output, got: %s", stdout)
	}
}

func TestDeleteCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{
			{ID: 2, Name: "app"},
		})
	})
	mux.HandleFunc("/api/project/2", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			http.Error(w, "expected DELETE", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewProjectCommand(), "project", "delete", "app", "--host", srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Deleted project") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
}

func TestCreateCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body["name"] != "infra" {
			http.Error(w, "unexpected name", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 5, "name": "infra"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewProjectCommand(), "project", "create", "--name", "infra", "--host", srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Created project") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
}

func TestSetCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{
			{ID: 1, Name: "infra"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Seed a config with an active profile pointing at the test server.
	h := testutil.New(t)
	cfg := config.DefaultConfig()
	cfg.CurrentProfile = "test"
	cfg.Profiles["test"] = &config.Profile{Host: srv.URL}
	h.WriteConfig(t, cfg)

	stdout, _, err := h.Run(t, NewProjectCommand(), "project", "set", "infra", "--host", srv.URL, "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Set project") {
		t.Fatalf("expected success message, got: %s", stdout)
	}

	// Verify the profile was persisted.
	read, err := os.ReadFile(config.Path())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(read), "output: json") && !strings.Contains(string(read), "default_output: json") {
		t.Fatalf("expected profile output to be updated, got: %s", string(read))
	}
}

func TestCreateCommandJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 5, "name": "infra", "max_parallel_tasks": 0})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewProjectCommand(), "project", "create", "--name", "infra", "--host", srv.URL, "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("expected JSON object for created project (issue #87), got: %s", stdout)
	}
	if out["id"] != float64(5) {
		t.Fatalf("expected created project id 5, got: %v", out["id"])
	}
}
