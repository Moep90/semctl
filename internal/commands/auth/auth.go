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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/auth"
	"github.com/moep90/semaphore-cli/internal/cli"
	"github.com/moep90/semaphore-cli/internal/config"
)

// NewAuthCommand builds the auth command group.
func NewAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with Semaphore UI",
		Long:  "Manage authentication state and credentials.",
		Example: `  semctl auth login https://semaphore.example.com
  pass show semaphore/token | semctl auth login https://semaphore.example.com --with-token
  semctl auth status
  semctl auth logout`,
	}
	cmd.AddCommand(newLoginCommand())
	cmd.AddCommand(newLogoutCommand())
	cmd.AddCommand(newStatusCommand())
	return cmd
}

func validateHost(host string) error {
	if host == "" {
		return fmt.Errorf("host is required")
	}
	u, err := url.Parse(host)
	if err != nil {
		return fmt.Errorf("invalid host URL: %w", err)
	}
	if !u.IsAbs() {
		return fmt.Errorf("host must be an absolute URL (e.g., https://semaphore.example.com)")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("host scheme must be https or http")
	}
	return nil
}

func newLoginCommand() *cobra.Command {
	var withToken bool
	var plaintext bool
	var cookie bool
	cmd := &cobra.Command{
		Use:   "login [HOST]",
		Short: "Authenticate to a Semaphore UI instance",
		Long: `Log in to a Semaphore UI server with an API token or username/password.

Interactive mode prompts for a token with hidden input. Use --with-token to pipe
a token from stdin (e.g., from a password manager). Tokens are stored in the OS
keyring when possible; use --plaintext only if the keyring is unavailable.

Use --cookie to authenticate with username/password instead of an API token.
This stores a session cookie for subsequent requests.

The host is required and must be an absolute URL (https:// or http://).`,
		Example: `  semctl auth login https://semaphore.example.com
  echo "$TOKEN" | semctl auth login https://semaphore.example.com --with-token
  SEMAPHORE_HOST=https://semaphore.example.com semctl auth login --with-token
  semctl auth login https://semaphore.example.com --cookie`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// login is special: it doesn't need an existing host.
			_, _ = cli.BuildContext(
				config.DefaultConfig(),
				"", "", "", "",
				false, false, false,
			)

			host := ""
			if len(args) > 0 {
				host = args[0]
			}
			if host == "" {
				host = os.Getenv("SEMAPHORE_HOST")
			}
			if host == "" {
				return fmt.Errorf("host required; provide as argument or set SEMAPHORE_HOST")
			}
			if err := validateHost(host); err != nil {
				return err
			}

			var token string
			var tokenSource string
			if cookie {
				tokenSource = "cookie"
				// Prompt for username and password.
				fmt.Fprint(os.Stderr, "? Username: ")
				reader := bufio.NewReader(os.Stdin)
				username, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("read username: %w", err)
				}
				username = strings.TrimSpace(username)
				if username == "" {
					return fmt.Errorf("username is required")
				}

				fmt.Fprint(os.Stderr, "? Password: ")
				var password string
				if term.IsTerminal(int(os.Stdin.Fd())) {
					b, err := term.ReadPassword(int(os.Stdin.Fd()))
					if err != nil {
						return fmt.Errorf("read password: %w", err)
					}
					password = strings.TrimSpace(string(b))
					_, _ = fmt.Fprintln(os.Stderr)
				} else {
					line, err := reader.ReadString('\n')
					if err != nil {
						return fmt.Errorf("read password: %w", err)
					}
					password = strings.TrimSpace(line)
				}
				if password == "" {
					return fmt.Errorf("password is required")
				}

				// POST /api/auth/login
				loginBody, _ := json.Marshal(api.AuthLoginRequest{Auth: username, Password: password})
				resp, err := http.Post(host+"/api/auth/login", "application/json", bytes.NewReader(loginBody))
				if err != nil {
					return fmt.Errorf("cookie login request failed: %w", err)
				}
				defer func() { _ = resp.Body.Close() }()
				if resp.StatusCode >= 400 {
					body, _ := io.ReadAll(resp.Body)
					return fmt.Errorf("cookie login failed (%d): %s", resp.StatusCode, string(body))
				}
				// Extract session cookie.
				for _, c := range resp.Cookies() {
					if c.Name == "semaphore" {
						token = c.Value
						break
					}
				}
				if token == "" {
					return fmt.Errorf("no session cookie received from login endpoint")
				}
				// Verify with a lightweight request.
				client := api.NewClientWithSource(host, token, tokenSource)
				user, err := auth.Login(cmd.Context(), client)
				if err != nil {
					return fmt.Errorf("login failed: %w", err)
				}
				_, _ = fmt.Fprintf(os.Stdout, "✓ Authenticated as %s\n", user.Username)
			} else {
				tokenSource = "bearer"
				if withToken {
					data, err := io.ReadAll(os.Stdin)
					if err != nil {
						return fmt.Errorf("read token from stdin: %w", err)
					}
					token = strings.TrimSpace(string(data))
				} else {
					fmt.Fprint(os.Stderr, "? Token: ")
					if term.IsTerminal(int(os.Stdin.Fd())) {
						b, err := term.ReadPassword(int(os.Stdin.Fd()))
						if err != nil {
							return fmt.Errorf("read token: %w", err)
						}
						token = strings.TrimSpace(string(b))
						_, _ = fmt.Fprintln(os.Stderr)
					} else {
						reader := bufio.NewReader(os.Stdin)
						line, err := reader.ReadString('\n')
						if err != nil {
							return fmt.Errorf("read token: %w", err)
						}
						token = strings.TrimSpace(line)
					}
				}

				if token == "" {
					return fmt.Errorf("token is required")
				}

				client := api.NewClient(host, token)
				user, err := auth.Login(cmd.Context(), client)
				if err != nil {
					return fmt.Errorf("login failed: %w", err)
				}
				_, _ = fmt.Fprintf(os.Stdout, "✓ Authenticated as %s\n", user.Username)
			}

			if err := auth.Store(host, token); err != nil {
				if !plaintext {
					return fmt.Errorf("could not store token in keyring (%s); re-run with --plaintext to store in config file", err)
				}
				_, _ = fmt.Fprintf(os.Stderr, "warning: could not store token in keyring (%s); storing in config file\n", err)
				cfg, _ := config.Load()
				if cfg == nil {
					cfg = config.DefaultConfig()
				}
				profileName := "default"
				if cfg.CurrentProfile != "" {
					profileName = cfg.CurrentProfile
				} else {
					cfg.CurrentProfile = profileName
				}
				if cfg.Profiles[profileName] == nil {
					cfg.Profiles[profileName] = &config.Profile{}
				}
				cfg.Profiles[profileName].Host = host
				cfg.Profiles[profileName].Token = token
				cfg.Profiles[profileName].TokenSource = tokenSource
				if err := config.Save(cfg); err != nil {
					return fmt.Errorf("save config: %w", err)
				}
			} else {
				// Persist host in the active profile so subsequent commands don't need --host.
				cfg, _ := config.Load()
				if cfg == nil {
					cfg = config.DefaultConfig()
				}
				profileName := cfg.CurrentProfile
				if profileName == "" {
					profileName = "default"
					cfg.CurrentProfile = profileName
				}
				if cfg.Profiles[profileName] == nil {
					cfg.Profiles[profileName] = &config.Profile{}
				}
				cfg.Profiles[profileName].Host = host
				cfg.Profiles[profileName].TokenSource = tokenSource
				_ = config.Save(cfg)
			}

			_, _ = fmt.Fprintf(os.Stdout, "✓ Stored credentials for %s\n", host)
			return nil
		},
	}
	cmd.Flags().BoolVar(&withToken, "with-token", false, "Read token from stdin")
	cmd.Flags().BoolVar(&plaintext, "plaintext", false, "Allow storing token in config file if keyring is unavailable")
	cmd.Flags().BoolVar(&cookie, "cookie", false, "Authenticate with username/password cookie session")
	return cmd
}

func newLogoutCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout [HOST]",
		Short: "Remove authentication for a host",
		Long:  `Remove the stored token for a host from the OS keyring and clear it from the active profile.`,
		Example: `  semctl auth logout https://semaphore.example.com
  semctl auth logout`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			host := ""
			if len(args) > 0 {
				host = args[0]
			}
			if host == "" {
				host, _ = cmd.Flags().GetString("host")
			}
			if host == "" {
				host = os.Getenv("SEMAPHORE_HOST")
			}
			if host == "" {
				cfg, err := config.Load()
				if err == nil && cfg.ActiveProfile() != nil {
					host = cfg.ActiveProfile().Host
				}
			}
			if host == "" {
				return fmt.Errorf("host required; provide as argument or set SEMAPHORE_HOST")
			}

			_ = auth.Delete(host)
			cfg, err := config.Load()
			if err == nil && cfg.ActiveProfile() != nil && cfg.ActiveProfile().Host == host {
				cfg.ActiveProfile().Token = ""
				_ = config.Save(cfg)
			}
			_, _ = fmt.Fprintf(os.Stdout, "✓ Logged out of %s\n", host)
			return nil
		},
	}
}

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Short:   "View authentication status",
		Long:    `Show the current host, profile, and whether the stored token is valid.`,
		Example: `  semctl auth status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			profile := cfg.ActiveProfile()
			if profile == nil {
				_, _ = fmt.Fprintln(os.Stdout, "not logged in to any host")
				return nil
			}

			token := auth.GetToken(profile.Host, cfg)
			if token == "" {
				_, _ = fmt.Fprintf(os.Stdout, "logged in to %s (no token stored)\n", profile.Host)
				return nil
			}

			client := api.NewClient(profile.Host, token)
			user, err := auth.Login(cmd.Context(), client)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stdout, "logged in to %s (token invalid)\n", profile.Host)
				return nil
			}

			_, _ = fmt.Fprintf(os.Stdout, "✓ Logged in to %s as %s\n", profile.Host, user.Username)
			return nil
		},
	}
}
