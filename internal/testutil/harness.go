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
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/cli"
	"github.com/moep90/semaphore-cli/internal/config"
)

// Harness isolates a single test from the user's real environment and runs
// commands against an in-memory root, capturing their output.
//
// It removes the boilerplate that every command test used to repeat: building a
// cobra root with the global flags, pointing XDG_CONFIG_HOME at a temp dir, and
// swapping the process-global os.Stdout with a pipe. Output is captured via
// cobra's SetOut/SetErr instead of the global os.Stdout, so there is no shared
// mutable global to clean up.
//
// Config isolation uses the XDG_CONFIG_HOME environment variable, which is
// process-global; tests using a Harness therefore must not call t.Parallel().
type Harness struct {
	// ConfigDir is the isolated XDG_CONFIG_HOME for this test. The config file
	// lives at <ConfigDir>/semctl/config.yml.
	ConfigDir string
}

// New creates a Harness with an isolated, empty config directory. The previous
// XDG_CONFIG_HOME value is restored automatically when the test finishes.
func New(t testing.TB) *Harness {
	t.Helper()
	dir := t.TempDir()

	prev, had := os.LookupEnv("XDG_CONFIG_HOME")
	if err := os.Setenv("XDG_CONFIG_HOME", dir); err != nil {
		t.Fatalf("set XDG_CONFIG_HOME: %v", err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("XDG_CONFIG_HOME", prev)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	})

	return &Harness{ConfigDir: dir}
}

// Run executes cmd as a subcommand of a fresh root configured with the global
// flags, returning whatever the command wrote to stdout and stderr. Pass a
// freshly constructed command (e.g. project.NewProjectCommand()) so cobra's
// per-run state does not leak between calls.
func (h *Harness) Run(t testing.TB, cmd *cobra.Command, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := newTestRoot()
	root.AddCommand(cmd)

	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs(args)

	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

// WriteConfig persists cfg into the isolated config directory using the same
// writer the application uses, so tests exercise the real save/load path.
func (h *Harness) WriteConfig(t testing.TB, cfg *config.Config) {
	t.Helper()
	if err := config.Save(cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

// RunCommand is a convenience wrapper for the common case of a single,
// stateless command run with an isolated empty config.
func RunCommand(t testing.TB, cmd *cobra.Command, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	return New(t).Run(t, cmd, args...)
}

// newTestRoot builds a bare root command carrying the same global flags as the
// real CLI, so BuildCmdContext resolves them identically under test.
func newTestRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "semctl",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cli.RegisterGlobalFlags(root)
	return root
}
