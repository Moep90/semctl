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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/moep90/semaphore-cli/internal/config"
	"github.com/moep90/semaphore-cli/internal/testutil"
)

func TestLogoutCommandRespectsHostFlag(t *testing.T) {
	// An active profile with a different host than the one passed via --host.
	h := testutil.New(t)
	cfg := config.DefaultConfig()
	cfg.CurrentProfile = "default"
	cfg.Profiles["default"] = &config.Profile{
		Host:  "http://profile.example.com",
		Token: "profile-token",
	}
	h.WriteConfig(t, cfg)

	stdout, _, err := h.Run(t, NewAuthCommand(), "auth", "logout", "--host", "http://flag.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "http://flag.example.com") {
		t.Fatalf("expected --host flag value in output, got: %s", stdout)
	}
	if strings.Contains(stdout, "http://profile.example.com") {
		t.Fatalf("expected profile host to NOT be in output, got: %s", stdout)
	}
}

func TestCookieLoginFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid credentials"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Drive the interactive username/password prompts via stdin. Prompts are
	// written to the command's stderr (captured by the harness); credentials are
	// read from os.Stdin, which the login command consumes via cmd.InOrStdin().
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })
	go func() {
		_, _ = io.WriteString(w, "admin\nbadpass\n")
		_ = w.Close()
	}()

	_, _, err := testutil.RunCommand(t, NewAuthCommand(), "auth", "login", srv.URL, "--cookie", "--plaintext")
	if err == nil {
		t.Fatal("expected error for failed cookie login")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected 401 in error, got: %v", err)
	}
}

// cookieLoginServer returns a server that accepts only the given credentials and
// issues a session cookie, plus a /api/user endpoint for the verification step.
func cookieLoginServer(t *testing.T, wantUser, wantPass string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["auth"] != wantUser || req["password"] != wantPass {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid credentials"}`))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "semaphore", Value: "session123"})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/user", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": "User", "username": wantUser})
	})
	return httptest.NewServer(mux)
}

func TestCookieLoginNoInteractive(t *testing.T) {
	srv := cookieLoginServer(t, "admin", "badpass")
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewAuthCommand(),
		"auth", "login", srv.URL, "--cookie", "--no-interactive",
		"--username", "admin", "--password", "badpass", "--plaintext")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Authenticated as admin") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
}

func TestCookieLoginWithFlags(t *testing.T) {
	srv := cookieLoginServer(t, "admin", "changeme")
	defer srv.Close()

	stdout, _, err := testutil.RunCommand(t, NewAuthCommand(),
		"auth", "login", srv.URL, "--cookie",
		"--username", "admin", "--password", "changeme", "--plaintext")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Authenticated as admin") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
}

func TestCookieLoginEnvVars(t *testing.T) {
	srv := cookieLoginServer(t, "envuser", "envpass")
	defer srv.Close()

	t.Setenv("SEMAPHORE_USERNAME", "envuser")
	t.Setenv("SEMAPHORE_PASSWORD", "envpass")

	stdout, _, err := testutil.RunCommand(t, NewAuthCommand(),
		"auth", "login", srv.URL, "--cookie", "--plaintext")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Authenticated as envuser") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
}

func TestCookieLoginFlagsOverrideEnvVars(t *testing.T) {
	srv := cookieLoginServer(t, "flaguser", "flagpass")
	defer srv.Close()

	t.Setenv("SEMAPHORE_USERNAME", "envuser")
	t.Setenv("SEMAPHORE_PASSWORD", "envpass")

	stdout, _, err := testutil.RunCommand(t, NewAuthCommand(),
		"auth", "login", srv.URL, "--cookie",
		"--username", "flaguser", "--password", "flagpass", "--plaintext")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Authenticated as flaguser") {
		t.Fatalf("expected success message, got: %s", stdout)
	}
}

func TestCookieLoginNoInteractiveMissingPassword(t *testing.T) {
	srv := cookieLoginServer(t, "admin", "changeme")
	defer srv.Close()

	_, _, err := testutil.RunCommand(t, NewAuthCommand(),
		"auth", "login", srv.URL, "--cookie", "--username", "admin", "--no-interactive")
	if err == nil {
		t.Fatal("expected error when --password is missing with --no-interactive")
	}
	if !strings.Contains(err.Error(), "password") {
		t.Fatalf("expected error about missing password, got: %v", err)
	}
}

func TestCookieLoginNoInteractiveMissingUsername(t *testing.T) {
	srv := cookieLoginServer(t, "admin", "changeme")
	defer srv.Close()

	_, _, err := testutil.RunCommand(t, NewAuthCommand(),
		"auth", "login", srv.URL, "--cookie", "--password", "changeme", "--no-interactive")
	if err == nil {
		t.Fatal("expected error when --username is missing with --no-interactive")
	}
	if !strings.Contains(err.Error(), "username") {
		t.Fatalf("expected error about missing username, got: %v", err)
	}
}
