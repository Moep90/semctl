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

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/cli"
)

// cmdWithOutput builds a command carrying the global flags with --output set to
// mode, as if the user had passed it.
func cmdWithOutput(t *testing.T, mode string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{Use: "semctl"}
	cli.RegisterGlobalFlags(cmd)
	if err := cmd.PersistentFlags().Set("output", mode); err != nil {
		t.Fatalf("set output flag: %v", err)
	}
	return cmd
}

func TestErrorOutputJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := formatError(cmdWithOutput(t, "json"), errors.New("test error"), &buf); err != nil {
		t.Fatalf("formatError: %v", err)
	}
	var out map[string]string
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("expected JSON output, got: %s", buf.String())
	}
	if out["error"] != "test error" {
		t.Fatalf("unexpected error content: %v", out)
	}
}

func TestErrorOutputYAML(t *testing.T) {
	var buf bytes.Buffer
	if err := formatError(cmdWithOutput(t, "yaml"), errors.New("test error"), &buf); err != nil {
		t.Fatalf("formatError: %v", err)
	}
	if !strings.Contains(buf.String(), "error") {
		t.Fatalf("expected error in output, got: %s", buf.String())
	}
}

func TestErrorOutputPlain(t *testing.T) {
	var buf bytes.Buffer
	if err := formatError(cmdWithOutput(t, ""), errors.New("test error"), &buf); err != nil {
		t.Fatalf("formatError: %v", err)
	}
	if !strings.Contains(buf.String(), "error: test error") {
		t.Fatalf("expected plain text error, got: %s", buf.String())
	}
}

// TestErrorOutputJSONShorthand verifies the --json shorthand also triggers JSON.
func TestErrorOutputJSONShorthand(t *testing.T) {
	cmd := &cobra.Command{Use: "semctl"}
	cli.RegisterGlobalFlags(cmd)
	if err := cmd.PersistentFlags().Set("json", "true"); err != nil {
		t.Fatalf("set json flag: %v", err)
	}
	var buf bytes.Buffer
	if err := formatError(cmd, errors.New("boom"), &buf); err != nil {
		t.Fatalf("formatError: %v", err)
	}
	var out map[string]string
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("expected JSON output, got: %s", buf.String())
	}
	if out["error"] != "boom" {
		t.Fatalf("unexpected error content: %v", out)
	}
}
