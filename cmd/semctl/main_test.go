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
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/api"
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

// decodeErrorObject decodes {"error": {...}} from JSON output.
func decodeErrorObject(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var out struct {
		Error map[string]any `json:"error"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("expected JSON output, got: %s", b)
	}
	if out.Error == nil {
		t.Fatalf("expected structured error object, got: %s", b)
	}
	return out.Error
}

func TestErrorOutputJSONStructured(t *testing.T) {
	var buf bytes.Buffer
	// A plain error classifies to SEM000001 (UNKNOWN_ERROR), message preserved.
	code, err := formatError(cmdWithOutput(t, "json"), errors.New("test error"), &buf)
	if err != nil {
		t.Fatalf("formatError: %v", err)
	}
	if code != 1 {
		t.Fatalf("unknown error exit code = %d, want 1", code)
	}
	obj := decodeErrorObject(t, buf.Bytes())
	if obj["code"] != "SEM000001" {
		t.Fatalf("code: %v", obj["code"])
	}
	if obj["message"] != "test error" {
		t.Fatalf("message: %v", obj["message"])
	}
}

func TestErrorOutputJSONAPINotFound(t *testing.T) {
	var buf bytes.Buffer
	apiErr := fmt.Errorf("api request: %w", &api.Error{StatusCode: 404, Method: "GET", Path: "/project/1/tasks/last"})
	code, err := formatError(cmdWithOutput(t, "json"), apiErr, &buf)
	if err != nil {
		t.Fatalf("formatError: %v", err)
	}
	if code != 44 {
		t.Fatalf("404 exit code = %d, want 44", code)
	}
	obj := decodeErrorObject(t, buf.Bytes())
	if obj["code"] != "SEM500004" {
		t.Fatalf("code: %v", obj["code"])
	}
	if obj["http_status"] != float64(404) {
		t.Fatalf("http_status: %v", obj["http_status"])
	}
}

func TestErrorOutputYAML(t *testing.T) {
	var buf bytes.Buffer
	if _, err := formatError(cmdWithOutput(t, "yaml"), errors.New("test error"), &buf); err != nil {
		t.Fatalf("formatError: %v", err)
	}
	if !strings.Contains(buf.String(), "SEM000001") {
		t.Fatalf("expected error code in YAML output, got: %s", buf.String())
	}
}

func TestErrorOutputPlain(t *testing.T) {
	var buf bytes.Buffer
	if _, err := formatError(cmdWithOutput(t, ""), errors.New("test error"), &buf); err != nil {
		t.Fatalf("formatError: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "error SEM000001") || !strings.Contains(out, "test error") {
		t.Fatalf("expected structured plain text error, got: %s", out)
	}
}

func TestErrorDebugShowsCause(t *testing.T) {
	cmd := &cobra.Command{Use: "semctl"}
	cli.RegisterGlobalFlags(cmd)
	if err := cmd.PersistentFlags().Set("debug", "true"); err != nil {
		t.Fatalf("set debug flag: %v", err)
	}
	// The cause (api.Error) carries detail kept out of the default message.
	wrapped := fmt.Errorf("api request: %w", &api.Error{StatusCode: 404, Method: "GET", Path: "/x", Body: []byte("boom-detail")})
	var buf bytes.Buffer
	if _, err := formatError(cmd, wrapped, &buf); err != nil {
		t.Fatalf("formatError: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "SEM500004") {
		t.Fatalf("expected class in output, got: %s", out)
	}
	// --debug must surface the underlying cause for diagnosis.
	if !strings.Contains(out, "boom-detail") {
		t.Fatalf("expected --debug to show the cause, got: %s", out)
	}
}

func TestErrorNoDebugHidesCause(t *testing.T) {
	wrapped := fmt.Errorf("api request: %w", &api.Error{StatusCode: 404, Method: "GET", Path: "/x", Body: []byte("boom-detail")})
	var buf bytes.Buffer
	if _, err := formatError(cmdWithOutput(t, ""), wrapped, &buf); err != nil {
		t.Fatalf("formatError: %v", err)
	}
	if strings.Contains(buf.String(), "boom-detail") {
		t.Fatalf("without --debug the cause/body must not appear, got: %s", buf.String())
	}
}

func TestErrorOutputJSONShorthand(t *testing.T) {
	cmd := &cobra.Command{Use: "semctl"}
	cli.RegisterGlobalFlags(cmd)
	if err := cmd.PersistentFlags().Set("json", "true"); err != nil {
		t.Fatalf("set json flag: %v", err)
	}
	var buf bytes.Buffer
	if _, err := formatError(cmd, errors.New("boom"), &buf); err != nil {
		t.Fatalf("formatError: %v", err)
	}
	obj := decodeErrorObject(t, buf.Bytes())
	if obj["message"] != "boom" {
		t.Fatalf("unexpected error content: %v", obj)
	}
}
