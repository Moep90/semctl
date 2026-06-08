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

package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/config"
)

func TestGetTokenEnv(t *testing.T) {
	_ = os.Setenv("SEMAPHORE_TOKEN", "env-token")
	defer func() { _ = os.Unsetenv("SEMAPHORE_TOKEN") }()

	cfg := config.DefaultConfig()
	tok := GetToken("https://semaphore.example.com", cfg)
	if tok != "env-token" {
		t.Fatalf("expected env-token, got %s", tok)
	}
}

func TestGetTokenProfile(t *testing.T) {
	_ = os.Unsetenv("SEMAPHORE_TOKEN")

	cfg := config.DefaultConfig()
	cfg.CurrentProfile = "prod"
	cfg.Profiles["prod"] = &config.Profile{
		Host:  "https://semaphore.example.com",
		Token: "profile-token",
	}

	tok := GetToken("https://semaphore.example.com", cfg)
	if tok != "profile-token" {
		t.Fatalf("expected profile-token, got %s", tok)
	}
}

func TestLogin(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/user", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": "Alice", "username": "alice"})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := api.NewClient(srv.URL, "valid-token")
	user, err := Login(context.Background(), client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Username != "alice" {
		t.Fatalf("unexpected username: %s", user.Username)
	}
}

func TestLoginFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/user", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := api.NewClient(srv.URL, "bad-token")
	_, err := Login(context.Background(), client)
	if err == nil {
		t.Fatal("expected error for bad token")
	}
}
