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

package schedule

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
	srv := testutil.NewMockServer()
	defer srv.Close()
	srv.ExpectJSON("GET", "/api/project/2/schedules", http.StatusOK, []api.Schedule{
		{ID: 5, Name: "nightly", TemplateID: 7, CronFormat: "0 2 * * *", Active: true},
	})

	stdout, _, err := testutil.RunCommand(t, NewScheduleCommand(),
		"schedule", "list", "--host", srv.URL(), "--project", "2", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out []map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(out))
	}
	// list JSON now matches `get`: native lowercase snake_case keys.
	if out[0]["name"] != "nightly" {
		t.Fatalf("unexpected name: %v", out[0]["name"])
	}
	if out[0]["cron_format"] != "0 2 * * *" {
		t.Fatalf("expected cron_format in list JSON, got: %s", stdout)
	}
}

func TestGetCommand(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()
	srv.ExpectJSON("GET", "/api/project/2/schedules/5", http.StatusOK,
		api.Schedule{ID: 5, Name: "nightly", TemplateID: 7, CronFormat: "0 2 * * *", Active: true})

	// Numeric arg short-circuits resolution, so no list call is required.
	stdout, _, err := testutil.RunCommand(t, NewScheduleCommand(),
		"schedule", "get", "5", "--host", srv.URL(), "--project", "2", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out["name"] != "nightly" {
		t.Fatalf("unexpected name: %v", out["name"])
	}
}

func TestGetCommandFullFields(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()
	// The Semaphore API uses `cron_format` and `active`; the struct previously
	// decoded `cron_expression`/`enabled`, so these never populated (issue #75).
	srv.Expect("GET", "/api/project/2/schedules/12", http.StatusOK,
		`{"id":12,"project_id":2,"template_id":1,"cron_format":"0 2 * * *","name":"nightly","active":true,"type":"","delete_after_run":false}`)

	stdout, _, err := testutil.RunCommand(t, NewScheduleCommand(),
		"schedule", "get", "12", "--host", srv.URL(), "--project", "2", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out["cron_format"] != "0 2 * * *" {
		t.Fatalf("expected cron_format populated, got: %s", stdout)
	}
	if out["active"] != true {
		t.Fatalf("expected active=true, got: %v", out["active"])
	}
}

func TestCreateCommand(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/project/2/schedules" {
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		http.Error(w, "unexpected request", http.StatusNotFound)
	}))
	defer srv.Close()

	// Numeric --template (7) short-circuits ResolveTemplateID so no list call is needed.
	stdout, _, err := testutil.RunCommand(t, NewScheduleCommand(),
		"schedule", "create",
		"--template", "7", "--cron", "0 2 * * *", "--name", "nightly",
		"--host", srv.URL, "--project", "2", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("server did not receive POST body")
	}
	if _, ok := got["template_id"]; !ok {
		t.Fatalf("expected template_id in body, got %v", got)
	}
	if got["cron_format"] != "0 2 * * *" {
		t.Fatalf("expected cron_format, got %v", got["cron_format"])
	}
	if got["name"] != "nightly" {
		t.Fatalf("expected name, got %v", got["name"])
	}
	if !strings.Contains(stdout, "Created schedule") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
}

func TestUpdateCommand(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && r.URL.Path == "/api/project/2/schedules/5" {
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "unexpected request", http.StatusNotFound)
	}))
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewScheduleCommand(),
		"schedule", "update", "5", "--cron", "0 3 * * *",
		"--host", srv.URL, "--project", "2", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["cron_format"] != "0 3 * * *" {
		t.Fatalf("expected updated cron_format, got %v", got["cron_format"])
	}
	if !strings.Contains(stdout, "Updated schedule") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
}

func TestCreateCommandReportsServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid cron"}`))
	}))
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewScheduleCommand(),
		"schedule", "create", "--template", "7", "--cron", "not-a-cron", "--name", "bad",
		"--host", srv.URL, "--project", "2")
	if err == nil {
		t.Fatalf("expected error on HTTP 400, got nil (stdout=%q)", stdout)
	}
	if strings.Contains(stdout, "Created schedule") {
		t.Fatalf("must not report false success, got: %q", stdout)
	}
}

func TestUpdateCommandReportsServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"schedule id in URL and in body must be the same"}`))
	}))
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewScheduleCommand(),
		"schedule", "update", "5", "--cron", "0 3 * * *",
		"--host", srv.URL, "--project", "2")
	if err == nil {
		t.Fatalf("expected error on HTTP 400, got nil (stdout=%q)", stdout)
	}
	if strings.Contains(stdout, "Updated schedule") {
		t.Fatalf("must not report false success, got: %q", stdout)
	}
}

func TestDeleteCommand(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()
	srv.Expect("DELETE", "/api/project/2/schedules/5", http.StatusNoContent, "")

	stdout, _, err := testutil.RunCommand(t, NewScheduleCommand(),
		"schedule", "delete", "5", "--host", srv.URL(), "--project", "2", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	srv.AssertCalled(t, "DELETE", "/api/project/2/schedules/5")
	if !strings.Contains(stdout, "Deleted schedule") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
}
