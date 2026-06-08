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

package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/moep90/semaphore-cli/internal/config"
)

func TestBuildContextVerbose(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.CurrentProfile = "prod"
	cfg.Profiles["prod"] = &config.Profile{
		Host:    "https://semaphore.example.com",
		Project: "infra",
	}

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ctx, err := BuildContext(cfg, "", "", "", "", false, true, false)

	_ = w.Close()
	os.Stderr = oldStderr

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "[verbose]") {
		t.Fatalf("expected verbose output on stderr, got: %s", out)
	}
	if !strings.Contains(out, "host:") {
		t.Fatalf("expected host in verbose output, got: %s", out)
	}
}

func TestValidateHost(t *testing.T) {
	for _, tt := range []struct {
		host    string
		wantErr bool
	}{
		{"https://semaphore.example.com", false},
		{"http://localhost:3000", false},
		{"", true},
		{"semaphore.example.com", true},
		{"ftp://semaphore.example.com", true},
		{"/path/to/socket", true},
	} {
		err := validateHost(tt.host)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("expected error for host %q", tt.host)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error for host %q: %v", tt.host, err)
		}
	}
}
