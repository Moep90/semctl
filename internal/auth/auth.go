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

package auth

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/config"
	"github.com/zalando/go-keyring"
)

const keyringService = "semctl"

// Store saves a token for a host.
func Store(host, token string) error {
	user, _ := os.UserHomeDir()
	if user == "" {
		user = "default"
	}
	// Keyring uses service + user as lookup; we encode host into user.
	krUser := fmt.Sprintf("%s@%s", user, host)
	return keyring.Set(keyringService, krUser, token)
}

// Retrieve loads a token for a host.
func Retrieve(host string) (string, error) {
	user, _ := os.UserHomeDir()
	if user == "" {
		user = "default"
	}
	krUser := fmt.Sprintf("%s@%s", user, host)
	return keyring.Get(keyringService, krUser)
}

// Delete removes a token for a host.
func Delete(host string) error {
	user, _ := os.UserHomeDir()
	if user == "" {
		user = "default"
	}
	krUser := fmt.Sprintf("%s@%s", user, host)
	return keyring.Delete(keyringService, krUser)
}

// GetToken resolves the authentication token using the precedence:
// 1. Environment variable SEMAPHORE_TOKEN
// 2. OS keyring for the host
// 3. Profile token field
func GetToken(host string, cfg *config.Config) string {
	if t := os.Getenv("SEMAPHORE_TOKEN"); t != "" {
		return t
	}
	if t, err := Retrieve(host); err == nil && t != "" {
		return t
	}
	if p := cfg.ActiveProfile(); p != nil && p.Host == host {
		return p.Token
	}
	return ""
}

// Login attempts to authenticate with a token and returns the user if successful.
func Login(ctx context.Context, client *api.Client) (*api.User, error) {
	resp, err := client.Do(ctx, "GET", "/user", nil)
	if err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}
	var user api.User
	if err := api.DecodeJSON(resp, &user); err != nil {
		return nil, fmt.Errorf("decode user: %w", err)
	}
	return &user, nil
}

// IsKeyringAvailable checks whether the OS keyring is usable.
func IsKeyringAvailable() bool {
	if runtime.GOOS == "linux" {
		// On Linux, keyring typically needs a running secret service.
		// We do a best-effort probe.
		_, err := keyring.Get(keyringService, "_probe_")
		return err == nil || err == keyring.ErrNotFound
	}
	return true
}
