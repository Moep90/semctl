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
