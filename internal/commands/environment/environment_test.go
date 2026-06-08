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

package environment

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

	"github.com/moep90/semaphore-cli/internal/api"
)

func TestListCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/project/1/environment", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Environment{
			{ID: 1, Name: "staging"},
			{ID: 2, Name: "production"},
		})
	})
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	root := newTestRoot(nil)
	root.SetArgs([]string{"environment", "list", "--host", srv.URL, "--project", "infra"})
	err := root.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := io.ReadAll(r)
	out := string(data)
	if !strings.Contains(out, "staging") {
		t.Fatalf("expected staging in output, got: %s", out)
	}
	if !strings.Contains(out, "production") {
		t.Fatalf("expected production in output, got: %s", out)
	}
}

func TestGetCommand(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Project{{ID: 1, Name: "infra"}})
	})
	mux.HandleFunc("/api/project/1/environment", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]api.Environment{
			{ID: 1, Name: "staging"},
			{ID: 2, Name: "production"},
		})
	})
	mux.HandleFunc("/api/project/1/environment/2", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(api.Environment{ID: 2, Name: "production", JSON: `{}`})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	root := newTestRoot(nil)
	root.SetArgs([]string{"environment", "get", "production", "--host", srv.URL, "--project", "infra"})
	err := root.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := io.ReadAll(r)
	out := string(data)
	if !strings.Contains(out, "production") {
		t.Fatalf("expected production in output, got: %s", out)
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
	root.AddCommand(NewEnvironmentCommand())
	if out != nil {
		root.SetOut(out)
		root.SetErr(out)
	}
	return root
}
