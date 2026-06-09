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

package inventory

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/testutil"
)

func TestListCommandPagination(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/projects":
			_ = json.NewEncoder(w).Encode([]api.Project{{ID: 2, Name: "infra"}})
		case "/api/project/2/inventory":
			gotQuery = r.URL.Query()
			_ = json.NewEncoder(w).Encode([]api.Inventory{{ID: 1, Name: "prod-hosts", Type: "static"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	_, _, err := testutil.RunCommand(t, NewInventoryCommand(), "inventory", "list", "--limit", "20", "--page", "2", "--host", srv.URL, "--project", "2")
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
		case "/api/project/2/inventory":
			gotRawQuery = r.URL.RawQuery
			_ = json.NewEncoder(w).Encode([]api.Inventory{{ID: 1, Name: "prod-hosts", Type: "static"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	_, _, err := testutil.RunCommand(t, NewInventoryCommand(), "inventory", "list", "--host", srv.URL, "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotRawQuery != "" {
		t.Fatalf("expected no query params, got %q", gotRawQuery)
	}
}

func TestListCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/project/1/inventory", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Inventory{
			{ID: 1, Name: "prod-hosts", Type: "static"},
			{ID: 2, Name: "dev-hosts", Type: "file"},
		})
	})
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewInventoryCommand(), "inventory", "list", "--host", srv.URL, "--project", "infra")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "prod-hosts") {
		t.Fatalf("expected prod-hosts in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "dev-hosts") {
		t.Fatalf("expected dev-hosts in output, got: %s", stdout)
	}
}

func TestGetCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	mux.HandleFunc("/api/project/1/inventory", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Inventory{
			{ID: 1, Name: "prod-hosts"},
			{ID: 2, Name: "dev-hosts"},
		})
	})
	mux.HandleFunc("/api/project/1/inventory/2", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(api.Inventory{ID: 2, Name: "dev-hosts", Type: "file"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewInventoryCommand(), "inventory", "get", "dev-hosts", "--host", srv.URL, "--project", "infra")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "dev-hosts") {
		t.Fatalf("expected dev-hosts in output, got: %s", stdout)
	}
}

func TestCreateCommand(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()
	srv.Expect("POST", "/api/project/2/inventory", 201, "{}")

	stdout, _, err := testutil.RunCommand(t, NewInventoryCommand(), "inventory", "create", "--name", "prod", "--host", srv.URL(), "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Created inventory prod") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
	srv.AssertCalled(t, "POST", "/api/project/2/inventory")
}

func TestCreateCommandBody(t *testing.T) {
	var got map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/api/project/2/inventory", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{}"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, _, err := testutil.RunCommand(t, NewInventoryCommand(), "inventory", "create",
		"--name", "prod", "--type", "static", "--inventory", "[web]\nhost1\n",
		"--host", srv.URL, "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["name"] != "prod" {
		t.Fatalf("expected name=prod, got: %v", got["name"])
	}
	if got["type"] != "static" {
		t.Fatalf("expected type=static, got: %v", got["type"])
	}
	if got["project_id"] != float64(2) {
		t.Fatalf("expected project_id=2, got: %v", got["project_id"])
	}
	if got["inventory"] != "[web]\nhost1\n" {
		t.Fatalf("expected inventory content, got: %v", got["inventory"])
	}
}

func TestCreateInterpretsEscapeSequences(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	// The raw string literal carries a literal backslash + n (as a shell would
	// pass it), which must be stored as a real newline (issue #79).
	_, _, err := testutil.RunCommand(t, NewInventoryCommand(), "inventory", "create",
		"--name", "prod", "--type", "static", "--inventory", `[localhost]\nlocalhost ansible_connection=local`,
		"--host", srv.URL, "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["inventory"] != "[localhost]\nlocalhost ansible_connection=local" {
		t.Fatalf("expected literal \\n interpreted as newline, got: %q", got["inventory"])
	}
}

func TestUpdateCommand(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()
	srv.ExpectJSON("GET", "/api/project/2/inventory", 200, []api.Inventory{{ID: 7, Name: "prod"}})
	srv.Expect("PUT", "/api/project/2/inventory/7", 204, "")

	stdout, _, err := testutil.RunCommand(t, NewInventoryCommand(), "inventory", "update", "prod", "--type", "file", "--host", srv.URL(), "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Updated inventory prod") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
	srv.AssertCalled(t, "PUT", "/api/project/2/inventory/7")
}

func TestUpdateCommandReportsServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"Inventory ID in body and URL must be the same"}`))
	}))
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewInventoryCommand(),
		"inventory", "update", "3", "--inventory", "new content",
		"--host", srv.URL, "--project", "2")
	if err == nil {
		t.Fatalf("expected error on HTTP 400, got nil (stdout=%q)", stdout)
	}
	if strings.Contains(stdout, "Updated inventory") {
		t.Fatalf("must not report false success, got: %q", stdout)
	}
}

func TestDeleteCommand(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()
	srv.ExpectJSON("GET", "/api/project/2/inventory", 200, []api.Inventory{{ID: 7, Name: "prod"}})
	srv.Expect("DELETE", "/api/project/2/inventory/7", 204, "")

	stdout, _, err := testutil.RunCommand(t, NewInventoryCommand(), "inventory", "delete", "prod", "--host", srv.URL(), "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Deleted inventory prod") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
	srv.AssertCalled(t, "DELETE", "/api/project/2/inventory/7")
}

func TestGetCommandFullFields(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()
	// Numeric arg short-circuits resolution, so only the get call is needed.
	srv.Expect("GET", "/api/project/2/inventory/22", http.StatusOK,
		`{"id":22,"name":"prod","project_id":2,"type":"static","inventory":"[web]\nhost1","ssh_key_id":5,"become_key_id":null}`)

	stdout, _, err := testutil.RunCommand(t, NewInventoryCommand(),
		"inventory", "get", "22", "--host", srv.URL(), "--project", "2", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out["inventory"] != "[web]\nhost1" {
		t.Fatalf("expected inventory content (issue #78), got: %v", out["inventory"])
	}
	if out["ssh_key_id"] != float64(5) {
		t.Fatalf("expected ssh_key_id=5, got: %v", out["ssh_key_id"])
	}
	// become_key_id must be present as null so it can be audited (issue #78).
	if v, ok := out["become_key_id"]; !ok || v != nil {
		t.Fatalf("expected become_key_id present and null, got ok=%v val=%v", ok, v)
	}
}

func TestCreateCommandKeyFlags(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	_, _, err := testutil.RunCommand(t, NewInventoryCommand(), "inventory", "create",
		"--name", "prod", "--type", "static", "--ssh-key-id", "5", "--become-key-id", "7",
		"--host", srv.URL, "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["ssh_key_id"] != float64(5) {
		t.Fatalf("expected ssh_key_id=5, got: %v", got["ssh_key_id"])
	}
	if got["become_key_id"] != float64(7) {
		t.Fatalf("expected become_key_id=7, got: %v", got["become_key_id"])
	}
}

func TestCreateCommandBecomeKeyNull(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	// `null` must send a JSON null so NOPASSWD hosts can clear become_key_id.
	_, _, err := testutil.RunCommand(t, NewInventoryCommand(), "inventory", "create",
		"--name", "prod", "--type", "static", "--become-key-id", "null",
		"--host", srv.URL, "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v, ok := got["become_key_id"]; !ok || v != nil {
		t.Fatalf("expected become_key_id present and null, got ok=%v val=%v", ok, v)
	}
}

func TestUpdateCommandKeyFlags(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			_ = json.NewDecoder(r.Body).Decode(&got)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "unexpected", http.StatusNotFound)
	}))
	defer srv.Close()

	_, _, err := testutil.RunCommand(t, NewInventoryCommand(), "inventory", "update", "3",
		"--ssh-key-id", "9", "--host", srv.URL, "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["ssh_key_id"] != float64(9) {
		t.Fatalf("expected ssh_key_id=9 in PUT body, got: %v", got["ssh_key_id"])
	}
}
