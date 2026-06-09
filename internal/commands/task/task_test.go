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

package task

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
		case "/api/project/2/tasks":
			gotQuery = r.URL.Query()
			_ = json.NewEncoder(w).Encode([]api.Task{{ID: 10, TemplateID: 7, Status: "success"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	_, _, err := testutil.RunCommand(t, NewTaskCommand(), "task", "list", "--limit", "20", "--page", "2", "--host", srv.URL, "--project", "2")
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
		case "/api/project/2/tasks":
			gotRawQuery = r.URL.RawQuery
			_ = json.NewEncoder(w).Encode([]api.Task{{ID: 10, TemplateID: 7, Status: "success"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	_, _, err := testutil.RunCommand(t, NewTaskCommand(), "task", "list", "--host", srv.URL, "--project", "2")
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
	mux.HandleFunc("/api/project/1/tasks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Task{
			{ID: 10, TemplateID: 7, Status: "success", Message: "Deploy"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewTaskCommand(), "task", "list", "--host", srv.URL, "--project", "infra", "--output", "json")
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

func TestGetCommandFullFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/project/1/tasks/812" {
			http.Error(w, "unexpected", http.StatusNotFound)
			return
		}
		// A still-running task: the Semaphore API returns Go's zero time for
		// `end`, which must surface as null (issue #70), not 0001-01-01.
		_, _ = w.Write([]byte(`{
			"id": 812, "template_id": 7, "project_id": 1, "status": "running",
			"commit_hash": "abc123", "commit_message": "deploy fix",
			"playbook": "site.yml", "limit": "web*", "git_branch": "main",
			"environment": "staging",
			"start": "2026-06-09T10:00:00Z", "end": "0001-01-01T00:00:00Z"
		}`))
	}))
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewTaskCommand(), "task", "get", "812", "--host", srv.URL, "--project", "1", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	for _, k := range []string{"commit_hash", "commit_message", "playbook", "limit"} {
		if _, ok := out[k]; !ok {
			t.Fatalf("expected %q in output (issue #82), got: %s", k, stdout)
		}
	}
	if out["commit_hash"] != "abc123" {
		t.Fatalf("commit_hash: %v", out["commit_hash"])
	}
	if out["playbook"] != "site.yml" {
		t.Fatalf("playbook: %v", out["playbook"])
	}
	if v, ok := out["end"]; ok && v != nil {
		t.Fatalf("expected end null for running task (issue #70), got: %v", v)
	}
}

func TestRunCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	mux.HandleFunc("/api/project/1/templates", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Template{{ID: 7, Name: "deploy-prod"}})
	})
	mux.HandleFunc("/api/project/1/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["template_id"] != float64(7) {
			t.Fatalf("unexpected template_id: %v", body["template_id"])
		}
		_ = json.NewEncoder(w).Encode(api.Task{ID: 812, TemplateID: 7, Status: "running"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewTaskCommand(), "task", "run", "deploy-prod", "--host", srv.URL, "--project", "infra", "--message", "Deploy release 1.8.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Queued task 812") {
		t.Fatalf("expected task queued message, got: %s", stdout)
	}
}

func TestRunCommandWithEnvInvResolution(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	mux.HandleFunc("/api/project/1/templates", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Template{{ID: 7, Name: "deploy-prod"}})
	})
	mux.HandleFunc("/api/project/1/environment", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Environment{
			{ID: 5, Name: "staging"},
			{ID: 6, Name: "production"},
		})
	})
	mux.HandleFunc("/api/project/1/inventory", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Inventory{
			{ID: 10, Name: "prod-hosts"},
			{ID: 11, Name: "dev-hosts"},
		})
	})
	mux.HandleFunc("/api/project/1/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		// environment_id and inventory_id MUST be numeric (float64 in JSON), not strings.
		envID, envOk := body["environment_id"].(float64)
		invID, invOk := body["inventory_id"].(float64)
		if !envOk || envID != 6 {
			t.Fatalf("expected environment_id=6 (float64), got %v (type %T)", body["environment_id"], body["environment_id"])
		}
		if !invOk || invID != 10 {
			t.Fatalf("expected inventory_id=10 (float64), got %v (type %T)", body["inventory_id"], body["inventory_id"])
		}
		_ = json.NewEncoder(w).Encode(api.Task{ID: 812, TemplateID: 7, Status: "running"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewTaskCommand(), "task", "run", "deploy-prod", "--host", srv.URL, "--project", "infra", "--environment", "production", "--inventory", "prod-hosts")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Queued task 812") {
		t.Fatalf("expected task queued message, got: %s", stdout)
	}
}

func TestRunCommandWithAnsibleFlags(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	mux.HandleFunc("/api/project/1/templates", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Template{{ID: 7, Name: "deploy-prod"}})
	})
	mux.HandleFunc("/api/project/1/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["tags"] != "deploy,restart" {
			t.Fatalf("expected tags=deploy,restart, got %v", body["tags"])
		}
		if body["skip_tags"] != "slow" {
			t.Fatalf("expected skip_tags=slow, got %v", body["skip_tags"])
		}
		if body["extra_vars"] != `{"version":"1.2.3"}` {
			t.Fatalf("expected extra_vars={\"version\":\"1.2.3\"}, got %v", body["extra_vars"])
		}
		check, ok := body["check"].(bool)
		if !ok || !check {
			t.Fatalf("expected check=true (bool), got %v (type %T)", body["check"], body["check"])
		}
		_ = json.NewEncoder(w).Encode(api.Task{ID: 812, TemplateID: 7, Status: "running"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewTaskCommand(), "task", "run", "deploy-prod",
		"--host", srv.URL, "--project", "infra",
		"--tags", "deploy,restart",
		"--skip-tags", "slow",
		"--extra-vars", `{"version":"1.2.3"}`,
		"--check")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Queued task 812") {
		t.Fatalf("expected task queued message, got: %s", stdout)
	}
}

func TestStopCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	mux.HandleFunc("/api/project/1/tasks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Task{{ID: 812, TemplateID: 7, Status: "running"}})
	})
	mux.HandleFunc("/api/project/1/tasks/812/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewTaskCommand(), "task", "stop", "812", "--host", srv.URL, "--project", "infra")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Stopped task 812") {
		t.Fatalf("expected stop confirmation, got: %s", stdout)
	}
}

func TestLogsCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	mux.HandleFunc("/api/project/1/tasks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Task{{ID: 812, TemplateID: 7, Status: "success"}})
	})
	mux.HandleFunc("/api/project/1/tasks/812/output", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.TaskOutput{
			{Time: time.Now().Format("15:04:05"), Output: "hello world"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewTaskCommand(), "task", "logs", "812", "--host", srv.URL, "--project", "infra")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "hello world") {
		t.Fatalf("expected log output, got: %s", stdout)
	}
}

func TestLogsFollowDeduplication(t *testing.T) {
	callCount := 0
	statusCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	mux.HandleFunc("/api/project/1/tasks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Task{{ID: 812, TemplateID: 7, Status: "running"}})
	})
	mux.HandleFunc("/api/project/1/tasks/812", func(w http.ResponseWriter, r *http.Request) {
		statusCount++
		status := "running"
		if statusCount >= 3 {
			status = "success"
		}
		_ = json.NewEncoder(w).Encode(api.Task{ID: 812, TemplateID: 7, Status: status})
	})
	mux.HandleFunc("/api/project/1/tasks/812/output", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// Simulate API that returns last N lines with indices resetting (like some paginated APIs).
		// First call: indices 0,1. Second call: indices 0,1 (same content). Third call: indices 0,1,2.
		// The current code would handle first->second fine, but let's test the case where
		// the same content appears with different indices.
		if callCount == 1 {
			_ = json.NewEncoder(w).Encode([]api.TaskOutput{
				{Time: "10:00:00", Output: "line one"},
				{Time: "10:00:01", Output: "line two"},
			})
		} else {
			_ = json.NewEncoder(w).Encode([]api.TaskOutput{
				{Time: "10:00:00", Output: "line one"},
				{Time: "10:00:01", Output: "line two"},
				{Time: "10:00:02", Output: "line three"},
			})
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Run with a timeout to avoid hanging if the test breaks.
	type result struct {
		stdout string
		err    error
	}
	done := make(chan result, 1)
	go func() {
		stdout, _, err := testutil.RunCommand(t, NewTaskCommand(), "task", "logs", "812", "--host", srv.URL, "--project", "infra", "--follow", "--interval", "10ms")
		done <- result{stdout: stdout, err: err}
	}()

	var out string
	select {
	case res := <-done:
		if res.err != nil {
			t.Fatalf("unexpected error: %v", res.err)
		}
		out = res.stdout
	case <-time.After(2 * time.Second):
		t.Fatal("follow logs timed out")
	}

	// Count occurrences of each line.
	countOne := strings.Count(out, "line one")
	countTwo := strings.Count(out, "line two")
	if countOne > 1 {
		t.Fatalf("expected line one once, got %d times in output: %s", countOne, out)
	}
	if countTwo > 1 {
		t.Fatalf("expected line two once, got %d times in output: %s", countTwo, out)
	}
	if callCount < 2 {
		t.Fatalf("expected at least 2 API calls, got %d", callCount)
	}
}
