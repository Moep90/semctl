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

package config

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/moep90/semaphore-cli/internal/testutil"

	cfgpkg "github.com/moep90/semaphore-cli/internal/config"
)

func TestConfigGet(t *testing.T) {
	h := testutil.New(t)
	cfg := cfgpkg.DefaultConfig()
	cfg.CurrentProfile = "prod"
	cfg.Profiles["prod"] = &cfgpkg.Profile{Host: "https://semaphore.example.com"}
	h.WriteConfig(t, cfg)

	stdout, _, err := h.Run(t, NewConfigCommand(), "config", "get", "host")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "https://semaphore.example.com") {
		t.Fatalf("expected host in output, got: %s", stdout)
	}
}

func TestConfigSet(t *testing.T) {
	h := testutil.New(t)
	cfg := cfgpkg.DefaultConfig()
	cfg.CurrentProfile = "prod"
	cfg.Profiles["prod"] = &cfgpkg.Profile{}
	h.WriteConfig(t, cfg)

	if _, _, err := h.Run(t, NewConfigCommand(), "config", "set", "project", "infra"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := cfgpkg.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if loaded.Profiles["prod"].Project != "infra" {
		t.Fatalf("expected project infra, got %s", loaded.Profiles["prod"].Project)
	}
}

func TestConfigList(t *testing.T) {
	h := testutil.New(t)
	cfg := cfgpkg.DefaultConfig()
	cfg.CurrentProfile = "prod"
	cfg.Profiles["prod"] = &cfgpkg.Profile{
		Host:    "https://semaphore.example.com",
		Project: "infra",
	}
	h.WriteConfig(t, cfg)

	stdout, _, err := h.Run(t, NewConfigCommand(), "config", "list", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out["current_profile"] != "prod" {
		t.Fatalf("unexpected current_profile: %v", out["current_profile"])
	}
}

func TestProfileListJSON(t *testing.T) {
	h := testutil.New(t)
	cfg := cfgpkg.DefaultConfig()
	cfg.CurrentProfile = "prod"
	cfg.Profiles["prod"] = &cfgpkg.Profile{Host: "https://semaphore.example.com"}
	cfg.Profiles["dev"] = &cfgpkg.Profile{Host: "https://dev.example.com"}
	h.WriteConfig(t, cfg)

	stdout, _, err := h.Run(t, NewConfigCommand(), "config", "profile", "list", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out []map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(out))
	}
	foundActive := false
	for _, p := range out {
		if p["active"] == true {
			foundActive = true
		}
	}
	if !foundActive {
		t.Fatalf("expected one active profile, got: %v", out)
	}
}

func TestProfileCreate(t *testing.T) {
	h := testutil.New(t)

	stdout, _, err := h.Run(t, NewConfigCommand(), "config", "profile", "create", "lab")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Created profile lab") {
		t.Fatalf("expected success message, got: %s", stdout)
	}

	cfg, err := cfgpkg.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Profiles["lab"] == nil {
		t.Fatal("expected lab profile to exist")
	}
}

func TestProfileCreateEmptyName(t *testing.T) {
	_, _, err := testutil.RunCommand(t, NewConfigCommand(), "config", "profile", "create", "")
	if err == nil {
		t.Fatal("expected error for empty profile name")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected error about empty name, got: %v", err)
	}
}

func TestProfileCreateWhitespaceName(t *testing.T) {
	_, _, err := testutil.RunCommand(t, NewConfigCommand(), "config", "profile", "create", "   ")
	if err == nil {
		t.Fatal("expected error for whitespace-only profile name")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected error about empty name, got: %v", err)
	}
}

func TestConfigSetInvalidOutput(t *testing.T) {
	h := testutil.New(t)
	cfg := cfgpkg.DefaultConfig()
	cfg.CurrentProfile = "prod"
	cfg.Profiles["prod"] = &cfgpkg.Profile{}
	h.WriteConfig(t, cfg)

	_, _, err := h.Run(t, NewConfigCommand(), "config", "set", "output", "invalid_mode")
	if err == nil {
		t.Fatal("expected error for invalid output mode")
	}
	if !strings.Contains(err.Error(), "invalid") || !strings.Contains(err.Error(), "output") {
		t.Fatalf("expected error about invalid output, got: %v", err)
	}
}
