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

package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseMode(t *testing.T) {
	for _, tt := range []struct {
		in      string
		want    Mode
		wantErr bool
	}{
		{"table", ModeTable, false},
		{"json", ModeJSON, false},
		{"yaml", ModeYAML, false},
		{"text", ModeText, false},
		{"unknown", "", true},
	} {
		m, err := ParseMode(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("expected error for %s", tt.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", tt.in, err)
		}
		if m != tt.want {
			t.Fatalf("mode mismatch for %s: got %s, want %s", tt.in, m, tt.want)
		}
	}
}

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Mode: ModeJSON, Stdout: &buf, Stderr: &buf}
	data := map[string]any{"id": 1, "name": "infra"}
	if err := p.Print(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}
	if out["name"] != "infra" {
		t.Fatalf("unexpected output: %v", out)
	}
}

func TestPrintYAML(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Mode: ModeYAML, Stdout: &buf, Stderr: &buf}
	data := map[string]any{"id": 1, "name": "infra"}
	if err := p.Print(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "name: infra") {
		t.Fatalf("unexpected yaml output: %s", buf.String())
	}
}

func TestPrintTable(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Mode: ModeTable, Stdout: &buf, Stderr: &buf}
	if err := p.PrintTable([]string{"ID", "NAME"}, [][]string{{"1", "infra"}, {"2", "app"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "infra") {
		t.Fatalf("table missing expected content: %s", out)
	}
}

func TestPrintTableJSON(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Mode: ModeJSON, Stdout: &buf, Stderr: &buf}
	if err := p.PrintTable([]string{"ID", "NAME"}, [][]string{{"1", "infra"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out []map[string]string
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(out) != 1 || out[0]["NAME"] != "infra" {
		t.Fatalf("unexpected output: %v", out)
	}
}

func TestSanitizeANSI(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want string
	}{
		{"hello world", "hello world"},
		{"\x1b[31mred\x1b[0m", "red"},
		{"\x1b[1;31;40m bold red bg\x1b[0m", " bold red bg"},
		{"\x1b]8;;https://evil.com\x07link\x1b]8;;\x07", "link"},
		{"no\x1b[K escape", "no escape"},
		{"\x1b]0;title\x07", ""},
	} {
		got := SanitizeANSI(tt.in)
		if got != tt.want {
			t.Fatalf("SanitizeANSI(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestPrintErrorJSON(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Mode: ModeJSON, Stdout: &buf, Stderr: &buf}
	p.PrintError("template not found", []string{"semctl template list"})
	out := buf.String()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("expected valid JSON error, got: %s", out)
	}
	if parsed["error"] != "template not found" {
		t.Fatalf("unexpected error message: %v", parsed["error"])
	}
}

func TestPrintErrorYAML(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Mode: ModeYAML, Stdout: &buf, Stderr: &buf}
	p.PrintError("template not found", []string{"semctl template list"})
	out := buf.String()
	if !strings.Contains(out, "error:") {
		t.Fatalf("expected YAML error output, got: %s", out)
	}
}

func TestPrintSuccessJSON(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Mode: ModeJSON, Stdout: &buf, Stderr: &buf}
	p.PrintSuccess("Task completed")
	out := buf.String()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("expected valid JSON success, got: %s", out)
	}
	if parsed["message"] != "Task completed" {
		t.Fatalf("unexpected message: %v", parsed["message"])
	}
}

func TestPrintError(t *testing.T) {
	var buf bytes.Buffer
	p := &Printer{Mode: ModeText, Stdout: &buf, Stderr: &buf}
	p.PrintError("template not found", []string{"semctl template list"})
	out := buf.String()
	if !strings.Contains(out, "template not found") {
		t.Fatalf("missing error message: %s", out)
	}
	if !strings.Contains(out, "semctl template list") {
		t.Fatalf("missing suggestion: %s", out)
	}
}
