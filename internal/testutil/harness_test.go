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

package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/cli"
	"github.com/moep90/semaphore-cli/internal/config"
)

// pingCommand is a minimal command that exercises the full request path:
// it builds a command context from the inherited global flags, calls the API
// client, and prints the decoded response. It mirrors what a real command does
// without depending on any command package (which would create an import cycle).
func pingCommand() *cobra.Command {
	return &cobra.Command{
		Use: "ping",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			resp, err := ctx.Client.Do(cmd.Context(), "GET", "/ping", nil)
			if err != nil {
				return err
			}
			var data map[string]string
			if err := api.DecodeJSON(resp, &data); err != nil {
				return err
			}
			_, err = fmt.Fprintln(ctx.Printer.Stdout, data["msg"])
			return err
		},
	}
}

func TestRunCommand_CapturesStdoutAndRoutesToMockServer(t *testing.T) {
	srv := NewMockServer()
	defer srv.Close()
	srv.ExpectJSON("GET", "/api/ping", 200, map[string]string{"msg": "pong"})

	stdout, stderr, err := RunCommand(t, pingCommand(), "ping", "--host", srv.URL())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "pong") {
		t.Fatalf("expected %q in stdout, got %q (stderr %q)", "pong", stdout, stderr)
	}
	srv.AssertCalled(t, "GET", "/api/ping")
}

func TestNew_IsolatesConfigDir(t *testing.T) {
	h := New(t)
	if h.ConfigDir == "" {
		t.Fatal("expected ConfigDir to be set")
	}
	if got := os.Getenv("XDG_CONFIG_HOME"); got != h.ConfigDir {
		t.Fatalf("expected XDG_CONFIG_HOME=%q, got %q", h.ConfigDir, got)
	}
	if !strings.HasPrefix(config.Path(), h.ConfigDir) {
		t.Fatalf("expected config path under %q, got %q", h.ConfigDir, config.Path())
	}
}

func TestHarness_WriteConfigPersists(t *testing.T) {
	h := New(t)
	cfg := config.DefaultConfig()
	cfg.Profiles["test"] = &config.Profile{Host: "https://example.test"}
	h.WriteConfig(t, cfg)

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if loaded.Profiles["test"] == nil || loaded.Profiles["test"].Host != "https://example.test" {
		t.Fatalf("expected persisted profile, got %+v", loaded.Profiles)
	}
	// Sanity: the file lives inside the isolated dir.
	if _, err := os.Stat(filepath.Join(h.ConfigDir, "semctl", "config.yml")); err != nil {
		t.Fatalf("expected config file in isolated dir: %v", err)
	}
}
