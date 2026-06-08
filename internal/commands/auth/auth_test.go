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
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/config"
)

func TestLogoutCommandRespectsHostFlag(t *testing.T) {
	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	// Create a config with an active profile that has a different host.
	cfg := config.DefaultConfig()
	cfg.CurrentProfile = "default"
	cfg.Profiles["default"] = &config.Profile{
		Host:  "http://profile.example.com",
		Token: "profile-token",
	}
	_ = config.Save(cfg)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	root := newTestRoot(nil)
	root.AddCommand(NewAuthCommand())
	root.SetArgs([]string{"auth", "logout", "--host", "http://flag.example.com"})
	err := root.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := io.ReadAll(r)
	out := string(data)
	if !strings.Contains(out, "http://flag.example.com") {
		t.Fatalf("expected --host flag value in output, got: %s", out)
	}
	if strings.Contains(out, "http://profile.example.com") {
		t.Fatalf("expected profile host to NOT be in output, got: %s", out)
	}
}

func TestCookieLoginFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid credentials"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	// Pipe username and password to stdin.
	oldStdin := os.Stdin
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdin = r
	os.Stderr = w // capture prompts too
	go func() {
		_, _ = io.WriteString(w, "admin\n")
		_, _ = io.WriteString(w, "badpass\n")
		_ = w.Close()
	}()

	root := newTestRoot(nil)
	root.AddCommand(NewAuthCommand())
	root.SetArgs([]string{"auth", "login", srv.URL, "--cookie", "--plaintext"})
	err := root.Execute()

	os.Stdin = oldStdin
	os.Stderr = oldStderr

	if err == nil {
		t.Fatal("expected error for failed cookie login")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected 401 in error, got: %v", err)
	}
}

func TestCookieLoginNoInteractive(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["auth"] != "admin" || req["password"] != "badpass" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid credentials"}`))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "semaphore", Value: "session123"})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/user", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": "Admin", "username": "admin"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	oldStdout := os.Stdout
	rPipe, wPipe, _ := os.Pipe()
	os.Stdout = wPipe

	root := newTestRoot(nil)
	root.AddCommand(NewAuthCommand())
	root.SetArgs([]string{"auth", "login", srv.URL, "--cookie", "--no-interactive", "--username", "admin", "--password", "badpass", "--plaintext"})
	err := root.Execute()

	_ = wPipe.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := io.ReadAll(rPipe)
	if !strings.Contains(string(data), "Authenticated as admin") {
		t.Fatalf("expected success message, got: %s", string(data))
	}
}

func TestCookieLoginWithFlags(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["auth"] != "admin" || req["password"] != "changeme" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid credentials"}`))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "semaphore", Value: "session123"})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/user", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "name": "Admin", "username": "admin"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	oldStdout := os.Stdout
	rPipe, wPipe, _ := os.Pipe()
	os.Stdout = wPipe

	root := newTestRoot(nil)
	root.AddCommand(NewAuthCommand())
	root.SetArgs([]string{"auth", "login", srv.URL, "--cookie", "--username", "admin", "--password", "changeme", "--plaintext"})
	err := root.Execute()

	_ = wPipe.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := io.ReadAll(rPipe)
	if !strings.Contains(string(data), "Authenticated as admin") {
		t.Fatalf("expected success message, got: %s", string(data))
	}
}

func TestCookieLoginEnvVars(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["auth"] != "envuser" || req["password"] != "envpass" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid credentials"}`))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "semaphore", Value: "session456"})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/user", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 2, "name": "EnvUser", "username": "envuser"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	_ = os.Setenv("SEMAPHORE_USERNAME", "envuser")
	defer func() { _ = os.Unsetenv("SEMAPHORE_USERNAME") }()
	_ = os.Setenv("SEMAPHORE_PASSWORD", "envpass")
	defer func() { _ = os.Unsetenv("SEMAPHORE_PASSWORD") }()

	oldStdout := os.Stdout
	rPipe, wPipe, _ := os.Pipe()
	os.Stdout = wPipe

	root := newTestRoot(nil)
	root.AddCommand(NewAuthCommand())
	root.SetArgs([]string{"auth", "login", srv.URL, "--cookie", "--plaintext"})
	err := root.Execute()

	_ = wPipe.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := io.ReadAll(rPipe)
	if !strings.Contains(string(data), "Authenticated as envuser") {
		t.Fatalf("expected success message, got: %s", string(data))
	}
}

func TestCookieLoginFlagsOverrideEnvVars(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		// Should use flags, not env vars
		if req["auth"] != "flaguser" || req["password"] != "flagpass" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid credentials"}`))
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "semaphore", Value: "session789"})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/user", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 3, "name": "FlagUser", "username": "flaguser"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	_ = os.Setenv("SEMAPHORE_USERNAME", "envuser")
	defer func() { _ = os.Unsetenv("SEMAPHORE_USERNAME") }()
	_ = os.Setenv("SEMAPHORE_PASSWORD", "envpass")
	defer func() { _ = os.Unsetenv("SEMAPHORE_PASSWORD") }()

	oldStdout := os.Stdout
	rPipe, wPipe, _ := os.Pipe()
	os.Stdout = wPipe

	root := newTestRoot(nil)
	root.AddCommand(NewAuthCommand())
	root.SetArgs([]string{"auth", "login", srv.URL, "--cookie", "--username", "flaguser", "--password", "flagpass", "--plaintext"})
	err := root.Execute()

	_ = wPipe.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := io.ReadAll(rPipe)
	if !strings.Contains(string(data), "Authenticated as flaguser") {
		t.Fatalf("expected success message, got: %s", string(data))
	}
}

func TestCookieLoginNoInteractiveMissingPassword(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	root := newTestRoot(nil)
	root.AddCommand(NewAuthCommand())
	root.SetArgs([]string{"auth", "login", srv.URL, "--cookie", "--username", "admin", "--no-interactive"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --password is missing with --no-interactive")
	}
	if !strings.Contains(err.Error(), "password") {
		t.Fatalf("expected error about missing password, got: %v", err)
	}
}

func TestCookieLoginNoInteractiveMissingUsername(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	root := newTestRoot(nil)
	root.AddCommand(NewAuthCommand())
	root.SetArgs([]string{"auth", "login", srv.URL, "--cookie", "--password", "changeme", "--no-interactive"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --username is missing with --no-interactive")
	}
	if !strings.Contains(err.Error(), "username") {
		t.Fatalf("expected error about missing username, got: %v", err)
	}
}

func newTestRoot(out *bytes.Buffer) *cobra.Command {
	root := &cobra.Command{
		Use:           "semctl",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().String("host", "", "")
	root.PersistentFlags().StringP("project", "p", "", "")
	root.PersistentFlags().StringP("output", "o", "", "")
	root.PersistentFlags().String("profile", "", "")
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("no-color", false, "")
	root.PersistentFlags().Bool("verbose", false, "")
	root.PersistentFlags().Bool("debug", false, "")
	root.PersistentFlags().Bool("no-interactive", false, "")
	if out != nil {
		root.SetOut(out)
		root.SetErr(out)
	}
	return root
}
