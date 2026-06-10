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

package ping

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/moep90/semaphore-cli/internal/testutil"
)

func TestPingSuccess(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewPingCommand(), "ping", "--host", srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "reachable") {
		t.Fatalf("expected reachable in output, got: %s", stdout)
	}
}

func TestPingJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewPingCommand(), "ping", "--host", srv.URL, "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out["message"] != "Semaphore UI is reachable" {
		t.Fatalf("unexpected message: %v", out["message"])
	}
}

func TestPingFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	if _, _, err := testutil.RunCommand(t, NewPingCommand(), "ping", "--host", srv.URL); err == nil {
		t.Fatal("expected error for non-200 ping")
	}
}

func TestPingNoHost(t *testing.T) {
	_, _, err := testutil.RunCommand(t, NewPingCommand(), "ping")
	if err == nil {
		t.Fatal("expected error when host is missing")
	}
	// The no-host gate now surfaces a typed config error class.
	if !strings.Contains(err.Error(), "SEM200001") {
		t.Fatalf("expected SEM200001 config-not-found class, got: %v", err)
	}
}
