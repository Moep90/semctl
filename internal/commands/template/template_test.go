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

// cloneServer fakes the source GET and the create POST for the clone command.
// Semaphore has no clone endpoint, so clone must read the source template and
// re-create it under the new name. posted captures the create body.
func cloneServer(t *testing.T, createStatus int) (*httptest.Server, *map[string]any) {
	t.Helper()
	posted := map[string]any{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/project/1/templates/7", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 7, "name": "deploy-prod", "project_id": 1,
			"playbook": "site.yml", "inventory_id": 9,
		})
	})
	mux.HandleFunc("/api/project/1/templates", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&posted)
		w.WriteHeader(createStatus)
		if createStatus < 300 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 42, "name": "deploy-staging", "project_id": 1,
				"playbook": "site.yml", "inventory_id": 9,
			})
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &posted
}

func TestCloneCommand(t *testing.T) {
	srv, posted := cloneServer(t, http.StatusCreated)

	stdout, _, err := testutil.RunCommand(t, NewTemplateCommand(), "template", "clone", "7", "deploy-staging", "--host", srv.URL, "--project", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The create body must carry the new name, drop the source id, and preserve
	// the rest of the source template.
	if (*posted)["name"] != "deploy-staging" {
		t.Fatalf("expected name=deploy-staging in create body, got: %v", *posted)
	}
	if _, ok := (*posted)["id"]; ok {
		t.Fatalf("source id must be stripped from create body, got: %v", *posted)
	}
	if (*posted)["playbook"] != "site.yml" {
		t.Fatalf("expected source fields preserved, got: %v", *posted)
	}
	if !strings.Contains(stdout, "Cloned template") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
}

func TestCloneCommandJSON(t *testing.T) {
	srv, _ := cloneServer(t, http.StatusCreated)

	stdout, _, err := testutil.RunCommand(t, NewTemplateCommand(), "template", "clone", "7", "deploy-staging", "--host", srv.URL, "--project", "1", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("expected JSON object for created template (issue #66), got: %s", stdout)
	}
	if out["id"] != float64(42) {
		t.Fatalf("expected new template id 42, got: %v", out["id"])
	}
	if out["name"] != "deploy-staging" {
		t.Fatalf("expected name deploy-staging, got: %v", out["name"])
	}
}

func TestCloneCommandServerError(t *testing.T) {
	srv, _ := cloneServer(t, http.StatusBadRequest)

	stdout, _, err := testutil.RunCommand(t, NewTemplateCommand(), "template", "clone", "7", "deploy-staging", "--host", srv.URL, "--project", "1")
	if err == nil {
		t.Fatalf("expected error when create returns 400 (issue #65), got nil (stdout=%q)", stdout)
	}
	if strings.Contains(stdout, "Cloned template") {
		t.Fatalf("must not report false success, got: %q", stdout)
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

func TestTasksCommandPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/project/1/templates/7/tasks" {
			_ = json.NewEncoder(w).Encode([]api.Task{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}})
			return
		}
		http.Error(w, "unexpected", http.StatusNotFound)
	}))
	defer srv.Close()

	// --limit caps a template's task history (576+ runs is common).
	stdout, _, err := testutil.RunCommand(t, NewTemplateCommand(), "template", "tasks", "7",
		"--limit", "2", "--host", srv.URL, "--project", "1", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var page1 []map[string]any
	if err := json.Unmarshal([]byte(stdout), &page1); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(page1) != 2 || page1[0]["id"] != float64(1) {
		t.Fatalf("expected first 2 tasks, got: %s", stdout)
	}

	stdout2, _, err := testutil.RunCommand(t, NewTemplateCommand(), "template", "tasks", "7",
		"--limit", "2", "--page", "2", "--host", srv.URL, "--project", "1", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var page2 []map[string]any
	if err := json.Unmarshal([]byte(stdout2), &page2); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(page2) != 2 || page2[0]["id"] != float64(3) {
		t.Fatalf("expected tasks 3,4 on page 2, got: %s", stdout2)
	}
}
