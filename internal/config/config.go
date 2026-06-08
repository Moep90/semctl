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
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Config holds the entire CLI configuration.
type Config struct {
	CurrentProfile string              `yaml:"current_profile,omitempty"`
	Profiles       map[string]*Profile `yaml:"profiles,omitempty"`
}

// Profile holds settings for one Semaphore UI instance.
type Profile struct {
	Host          string `yaml:"host,omitempty"`
	Project       string `yaml:"project,omitempty"`
	TokenSource   string `yaml:"token_source,omitempty"`
	DefaultOutput string `yaml:"default_output,omitempty"`
	Token         string `yaml:"token,omitempty"`
}

// DefaultConfig returns an empty config with initialized map.
func DefaultConfig() *Config {
	return &Config{
		Profiles: make(map[string]*Profile),
	}
}

// Path returns the platform-native configuration file path.
func Path() string {
	if p := os.Getenv("XDG_CONFIG_HOME"); p != "" && runtime.GOOS == "linux" {
		return filepath.Join(p, "semctl", "config.yml")
	}
	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "semctl", "config.yml")
	case "windows":
		if p := os.Getenv("AppData"); p != "" {
			return filepath.Join(p, "semctl", "config.yml")
		}
		return filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming", "semctl", "config.yml")
	default:
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", "semctl", "config.yml")
	}
}

// Load reads the configuration file.
func Load() (*Config, error) {
	p := Path()
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]*Profile)
	}
	return &cfg, nil
}

// Save writes the configuration file atomically.
func Save(cfg *Config) error {
	p := Path()
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if stat, err := os.Stat(dir); err == nil {
		if stat.Mode().Perm()&0002 != 0 {
			return fmt.Errorf("config directory %s is world-writable; refusing to write secrets", dir)
		}
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	// Write to temp then rename for atomicity.
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("write config temp: %w", err)
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

// ActiveProfile returns the current profile or nil.
func (c *Config) ActiveProfile() *Profile {
	if c.CurrentProfile == "" {
		return nil
	}
	return c.Profiles[c.CurrentProfile]
}

// Set sets a simple config key on the active profile.
func (c *Config) Set(key, value string) error {
	if key == "current_profile" {
		c.CurrentProfile = value
		return nil
	}
	p := c.ActiveProfile()
	if p == nil {
		return fmt.Errorf("no active profile; create one with 'semctl config profile create'")
	}
	switch key {
	case "host":
		p.Host = value
	case "project":
		p.Project = value
	case "token_source":
		p.TokenSource = value
	case "output", "default_output":
		switch value {
		case "table", "json", "yaml", "text":
			p.DefaultOutput = value
		default:
			return fmt.Errorf("invalid output mode %q; must be table, json, yaml, or text", value)
		}
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

// Get returns a simple config value from the active profile.
func (c *Config) Get(key string) (string, error) {
	if key == "current_profile" {
		return c.CurrentProfile, nil
	}
	p := c.ActiveProfile()
	if p == nil {
		return "", fmt.Errorf("no active profile; create one with 'semctl config profile create'")
	}
	switch key {
	case "host":
		return p.Host, nil
	case "project":
		return p.Project, nil
	case "token_source":
		return p.TokenSource, nil
	case "output", "default_output":
		return p.DefaultOutput, nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}
