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

package resolver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/moep90/semaphore-cli/internal/api"
)

func TestResolveProjectNumeric(t *testing.T) {
	c := api.NewClient("http://example.com", "")
	id, err := ResolveProject(context.Background(), c, "42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Fatalf("expected 42, got %d", id)
	}
}

func TestResolveProjectByName(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{
			{ID: 1, Name: "infra"},
			{ID: 2, Name: "app"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL, "")
	id, err := ResolveProject(context.Background(), c, "infra")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 1 {
		t.Fatalf("expected 1, got %d", id)
	}
}

func TestResolveProjectAmbiguous(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{
			{ID: 1, Name: "deploy-dev"},
			{ID: 2, Name: "deploy-prod"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL, "")
	_, err := ResolveProject(context.Background(), c, "deploy")
	if err == nil {
		t.Fatal("expected ambiguous error")
	}
	if !contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous error, got: %v", err)
	}
}

func TestResolveProjectNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL, "")
	_, err := ResolveProject(context.Background(), c, "missing")
	if err == nil {
		t.Fatal("expected not found error")
	}
}

func TestResolveTemplateCaseInsensitive(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/project/1/templates", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Template{
			{ID: 7, Name: "Deploy-Prod"},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := api.NewClient(srv.URL, "")
	id, err := ResolveTemplate(context.Background(), c, 1, "deploy-prod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 7 {
		t.Fatalf("expected 7, got %d", id)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
