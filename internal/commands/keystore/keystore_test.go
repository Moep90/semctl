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

package keystore

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
	mux.HandleFunc("/api/project/1/keys", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Keystore{
			{ID: 1, Name: "deploy-key", Type: "ssh"},
			{ID: 2, Name: "vault-pass", Type: "login_password"},
		})
	})
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewKeystoreCommand(), "keystore", "list", "--host", srv.URL, "--project", "infra")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "deploy-key") {
		t.Fatalf("expected deploy-key in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "vault-pass") {
		t.Fatalf("expected vault-pass in output, got: %s", stdout)
	}
}

func TestGetCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	mux.HandleFunc("/api/project/1/keys", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Keystore{
			{ID: 1, Name: "deploy-key"},
			{ID: 2, Name: "vault-pass"},
		})
	})
	mux.HandleFunc("/api/project/1/keys/2", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(api.Keystore{ID: 2, Name: "vault-pass", Type: "login_password"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewKeystoreCommand(), "keystore", "get", "vault-pass", "--host", srv.URL, "--project", "infra")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "vault-pass") {
		t.Fatalf("expected vault-pass in output, got: %s", stdout)
	}
}

func TestCreateCommand(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()
	srv.Expect("POST", "/api/project/2/keys", 201, "{}")

	stdout, _, err := testutil.RunCommand(t, NewKeystoreCommand(), "keystore", "create", "--name", "deploy-key", "--host", srv.URL(), "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Created keystore deploy-key") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
	srv.AssertCalled(t, "POST", "/api/project/2/keys")
}

func TestCreateCommandSSHBody(t *testing.T) {
	var got map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/api/project/2/keys", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{}"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, _, err := testutil.RunCommand(t, NewKeystoreCommand(), "keystore", "create",
		"--name", "deploy-key", "--type", "ssh", "--login", "git", "--private-key", "PRIVKEY",
		"--host", srv.URL, "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["name"] != "deploy-key" {
		t.Fatalf("expected name=deploy-key, got: %v", got["name"])
	}
	if got["type"] != "ssh" {
		t.Fatalf("expected type=ssh, got: %v", got["type"])
	}
	if got["project_id"] != float64(2) {
		t.Fatalf("expected project_id=2, got: %v", got["project_id"])
	}
	ssh, ok := got["ssh"].(map[string]any)
	if !ok {
		t.Fatalf("expected ssh block, got: %v", got["ssh"])
	}
	if ssh["login"] != "git" {
		t.Fatalf("expected ssh.login=git, got: %v", ssh["login"])
	}
	if ssh["private_key"] != "PRIVKEY" {
		t.Fatalf("expected ssh.private_key=PRIVKEY, got: %v", ssh["private_key"])
	}
}

func TestCreateCommandLoginPasswordBody(t *testing.T) {
	var got map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/api/project/2/keys", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{}"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	_, _, err := testutil.RunCommand(t, NewKeystoreCommand(), "keystore", "create",
		"--name", "vault-pass", "--type", "login_password", "--login", "admin", "--password", "s3cret",
		"--host", srv.URL, "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lp, ok := got["login_password"].(map[string]any)
	if !ok {
		t.Fatalf("expected login_password block, got: %v", got["login_password"])
	}
	if lp["login"] != "admin" {
		t.Fatalf("expected login_password.login=admin, got: %v", lp["login"])
	}
	if lp["password"] != "s3cret" {
		t.Fatalf("expected login_password.password=s3cret, got: %v", lp["password"])
	}
}

func TestUpdateCommand(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()
	srv.ExpectJSON("GET", "/api/project/2/keys", 200, []api.Keystore{{ID: 9, Name: "deploy-key"}})
	srv.Expect("PUT", "/api/project/2/keys/9", 204, "")

	stdout, _, err := testutil.RunCommand(t, NewKeystoreCommand(), "keystore", "update", "deploy-key", "--name", "renamed", "--host", srv.URL(), "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Updated keystore deploy-key") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
	srv.AssertCalled(t, "PUT", "/api/project/2/keys/9")
}

func TestDeleteCommand(t *testing.T) {
	srv := testutil.NewMockServer()
	defer srv.Close()
	srv.ExpectJSON("GET", "/api/project/2/keys", 200, []api.Keystore{{ID: 9, Name: "deploy-key"}})
	srv.Expect("DELETE", "/api/project/2/keys/9", 204, "")

	stdout, _, err := testutil.RunCommand(t, NewKeystoreCommand(), "keystore", "delete", "deploy-key", "--host", srv.URL(), "--project", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Deleted keystore deploy-key") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
	srv.AssertCalled(t, "DELETE", "/api/project/2/keys/9")
}
