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

package template

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/testutil"
)

func TestListCommandPagination(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/projects":
			_ = json.NewEncoder(w).Encode([]api.Project{{ID: 2, Name: "infra"}})
		case "/api/project/2/templates":
			gotQuery = r.URL.Query()
			_ = json.NewEncoder(w).Encode([]api.Template{{ID: 7, Name: "deploy-prod"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	_, _, err := testutil.RunCommand(t, NewTemplateCommand(), "template", "list", "--limit", "20", "--page", "2", "--host", srv.URL, "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := gotQuery.Get("count"); got != "20" {
		t.Fatalf("expected count=20, got count=%q", got)
	}
	if got := gotQuery.Get("page"); got != "2" {
		t.Fatalf("expected page=2, got page=%q", got)
	}
}

func TestListCommandNoPagination(t *testing.T) {
	gotRawQuery := "unset"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/projects":
			_ = json.NewEncoder(w).Encode([]api.Project{{ID: 2, Name: "infra"}})
		case "/api/project/2/templates":
			gotRawQuery = r.URL.RawQuery
			_ = json.NewEncoder(w).Encode([]api.Template{{ID: 7, Name: "deploy-prod"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	_, _, err := testutil.RunCommand(t, NewTemplateCommand(), "template", "list", "--host", srv.URL, "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotRawQuery != "" {
		t.Fatalf("expected no query params, got %q", gotRawQuery)
	}
}

func TestListCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	mux.HandleFunc("/api/project/1/templates", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Template{
			{ID: 7, Name: "deploy-prod", App: "ansible", Playbook: "site.yml"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewTemplateCommand(), "template", "list", "--host", srv.URL, "--project", "infra", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out []map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 template, got %d", len(out))
	}
}

func TestGetCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	mux.HandleFunc("/api/project/1/templates", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Template{
			{ID: 7, Name: "deploy-prod"},
			{ID: 8, Name: "deploy-dev"},
		})
	})
	mux.HandleFunc("/api/project/1/templates/7", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(api.Template{ID: 7, Name: "deploy-prod", Playbook: "site.yml"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewTemplateCommand(), "template", "get", "deploy-prod", "--host", srv.URL, "--project", "infra", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out["name"] != "deploy-prod" {
		t.Fatalf("unexpected name: %v", out["name"])
	}
}

func TestDeleteCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	mux.HandleFunc("/api/project/1/templates", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Template{
			{ID: 7, Name: "deploy-prod"},
		})
	})
	mux.HandleFunc("/api/project/1/templates/7", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			http.Error(w, "expected DELETE", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewTemplateCommand(), "template", "delete", "deploy-prod", "--host", srv.URL, "--project", "infra", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Deleted template") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
}

func TestCloneCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	mux.HandleFunc("/api/project/1/templates", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Template{
			{ID: 7, Name: "deploy-prod"},
		})
	})
	mux.HandleFunc("/api/project/1/templates/7/clone", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body["name"] != "deploy-staging" {
			http.Error(w, "unexpected name", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewTemplateCommand(), "template", "clone", "deploy-prod", "deploy-staging", "--host", srv.URL, "--project", "infra", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Cloned template") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
}

func TestTasksCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	mux.HandleFunc("/api/project/1/templates", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Template{
			{ID: 7, Name: "deploy-prod"},
		})
	})
	mux.HandleFunc("/api/project/1/templates/7/tasks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Task{
			{ID: 101, TemplateID: 7, ProjectID: 1, Status: "success", Message: "deployed", Created: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewTemplateCommand(), "template", "tasks", "deploy-prod", "--host", srv.URL, "--project", "infra", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out []map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 task, got %d", len(out))
	}
}

// TestGetCommandIncludesFullConfig is a regression test for #43: template get
// must surface the *_id and behavior fields the API returns, not just the
// display names.
func TestGetCommandIncludesFullConfig(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()
	srv.ExpectJSON("GET", "/api/project/2/templates/5", 200, api.Template{
		ID: 5, Name: "deploy", ProjectID: 2, App: "ansible", Playbook: "deploy.yml",
		InventoryID: 8, EnvironmentID: 6, RepositoryID: 1, ViewID: 3,
		GitBranch: "main", AllowOverrideBranchInTask: true, SuppressSuccessAlert: true,
	})

	stdout, _, err := testutil.RunCommand(t, NewTemplateCommand(),
		"template", "get", "5", "--host", srv.URL(), "--project", "2", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"inventory_id", "environment_id", "repository_id", "git_branch", "allow_override_branch_in_task"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected %q in template get output, got: %s", want, stdout)
		}
	}
}

func TestListCommandJSONSchema(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Template{
			{ID: 7, Name: "deploy-prod", App: "ansible", Playbook: "site.yml",
				InventoryID: 9, EnvironmentID: 5, RepositoryID: 3},
		})
	}))
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewTemplateCommand(), "template", "list", "--host", srv.URL, "--project", "2", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out []map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	// id must be an integer (issue #71), not the string "7".
	if out[0]["id"] != float64(7) {
		t.Fatalf("expected numeric id=7, got %v (%T)", out[0]["id"], out[0]["id"])
	}
	// association ids must be present (issue #67), not blank.
	if out[0]["inventory_id"] != float64(9) {
		t.Fatalf("expected inventory_id=9, got: %s", stdout)
	}
	if _, ok := out[0]["ID"]; ok {
		t.Fatalf("must not emit uppercase keys, got: %s", stdout)
	}
}
