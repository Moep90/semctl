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
	"os"
	"path/filepath"
	"testing"
)

func TestPath(t *testing.T) {
	p := Path()
	if p == "" {
		t.Fatal("expected non-empty path")
	}
	if filepath.Ext(p) != ".yml" {
		t.Fatalf("expected .yml extension, got %s", filepath.Ext(p))
	}
}

func TestLoadMissing(t *testing.T) {
	tmp := t.TempDir()
	old := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", old) }()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.CurrentProfile != "" {
		t.Fatalf("expected empty current_profile, got %s", cfg.CurrentProfile)
	}
	if len(cfg.Profiles) != 0 {
		t.Fatalf("expected empty profiles, got %d", len(cfg.Profiles))
	}
}

func TestRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	old := os.Getenv("XDG_CONFIG_HOME")
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Setenv("XDG_CONFIG_HOME", old) }()

	cfg := DefaultConfig()
	cfg.CurrentProfile = "prod"
	cfg.Profiles["prod"] = &Profile{
		Host:          "https://semaphore.example.com",
		Project:       "infra",
		TokenSource:   "keyring",
		DefaultOutput: "table",
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.CurrentProfile != "prod" {
		t.Fatalf("expected current_profile prod, got %s", loaded.CurrentProfile)
	}
	p, ok := loaded.Profiles["prod"]
	if !ok {
		t.Fatal("expected prod profile")
	}
	if p.Host != "https://semaphore.example.com" {
		t.Fatalf("unexpected host: %s", p.Host)
	}
}

func TestSetGet(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Set("current_profile", "lab"); err != nil {
		t.Fatalf("set: %v", err)
	}
	v, err := cfg.Get("current_profile")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if v != "lab" {
		t.Fatalf("expected lab, got %s", v)
	}
	_, err = cfg.Get("unknown_key")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}
