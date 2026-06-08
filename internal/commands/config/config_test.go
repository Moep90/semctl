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
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	cfgpkg "github.com/moep90/semaphore-cli/internal/config"
)

func TestConfigGet(t *testing.T) {
	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	cfg := cfgpkg.DefaultConfig()
	cfg.CurrentProfile = "prod"
	cfg.Profiles["prod"] = &cfgpkg.Profile{Host: "https://semaphore.example.com"}
	_ = cfgpkg.Save(cfg)

	oldStdout := os.Stdout
	rPipe, wPipe, _ := os.Pipe()
	os.Stdout = wPipe

	root := newTestRoot(nil)
	root.SetArgs([]string{"config", "get", "host"})
	if err := root.Execute(); err != nil {
		_ = wPipe.Close()
		os.Stdout = oldStdout
		t.Fatalf("unexpected error: %v", err)
	}
	_ = wPipe.Close()
	os.Stdout = oldStdout
	data, _ := io.ReadAll(rPipe)
	out := string(data)
	if !strings.Contains(out, "https://semaphore.example.com") {
		t.Fatalf("expected host in output, got: %s", out)
	}
}

func TestConfigSet(t *testing.T) {
	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	cfg := cfgpkg.DefaultConfig()
	cfg.CurrentProfile = "prod"
	cfg.Profiles["prod"] = &cfgpkg.Profile{}
	_ = cfgpkg.Save(cfg)

	var buf bytes.Buffer
	root := newTestRoot(&buf)
	root.SetArgs([]string{"config", "set", "project", "infra"})
	if err := root.Execute(); err != nil {
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
	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	cfg := cfgpkg.DefaultConfig()
	cfg.CurrentProfile = "prod"
	cfg.Profiles["prod"] = &cfgpkg.Profile{
		Host:    "https://semaphore.example.com",
		Project: "infra",
	}
	_ = cfgpkg.Save(cfg)

	var buf bytes.Buffer
	root := newTestRoot(&buf)
	root.SetArgs([]string{"config", "list", "--output", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if out["current_profile"] != "prod" {
		t.Fatalf("unexpected current_profile: %v", out["current_profile"])
	}
}

func TestProfileCreate(t *testing.T) {
	tmp := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	defer func() { _ = os.Unsetenv("XDG_CONFIG_HOME") }()

	oldStdout := os.Stdout
	rPipe, wPipe, _ := os.Pipe()
	os.Stdout = wPipe

	root := newTestRoot(nil)
	root.SetArgs([]string{"config", "profile", "create", "lab"})
	if err := root.Execute(); err != nil {
		_ = wPipe.Close()
		os.Stdout = oldStdout
		t.Fatalf("unexpected error: %v", err)
	}
	_ = wPipe.Close()
	os.Stdout = oldStdout
	data, _ := io.ReadAll(rPipe)
	if !strings.Contains(string(data), "Created profile lab") {
		t.Fatalf("expected success message, got: %s", string(data))
	}

	cfg, err := cfgpkg.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Profiles["lab"] == nil {
		t.Fatal("expected lab profile to exist")
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
	root.AddCommand(NewConfigCommand())
	if out != nil {
		root.SetOut(out)
		root.SetErr(out)
	}
	return root
}
