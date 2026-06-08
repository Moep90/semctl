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
)

func TestErrorOutputJSON(t *testing.T) {
	var buf bytes.Buffer
	jsonFlag = false
	outputFlag = "json"
	err := formatError(errors.New("test error"), &buf)
	if !strings.Contains(buf.String(), "test error") {
		t.Fatalf("expected error in output, got: %s", buf.String())
	}
	var out map[string]string
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("expected JSON output, got: %s", buf.String())
	}
	if out["error"] != "test error" {
		t.Fatalf("unexpected error content: %v", out)
	}
	_ = err
}

func TestErrorOutputYAML(t *testing.T) {
	var buf bytes.Buffer
	jsonFlag = false
	outputFlag = "yaml"
	err := formatError(errors.New("test error"), &buf)
	if !strings.Contains(buf.String(), "error") {
		t.Fatalf("expected error in output, got: %s", buf.String())
	}
	_ = err
}

func TestErrorOutputPlain(t *testing.T) {
	var buf bytes.Buffer
	jsonFlag = false
	outputFlag = ""
	err := formatError(errors.New("test error"), &buf)
	if !strings.Contains(buf.String(), "error: test error") {
		t.Fatalf("expected plain text error, got: %s", buf.String())
	}
	_ = err
}
