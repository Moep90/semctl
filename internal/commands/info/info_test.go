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

package info

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/testutil"
)

func TestInfoCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(api.Info{Version: "2.10.0"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewInfoCommand(), "info", "--host", srv.URL, "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out["version"] != "2.10.0" {
		t.Fatalf("unexpected version: %v", out["version"])
	}
}

func TestInfoCommandTable(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(api.Info{Version: "2.10.0"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewInfoCommand(), "info", "--host", srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "2.10.0") {
		t.Fatalf("expected version in output, got: %s", stdout)
	}
}
