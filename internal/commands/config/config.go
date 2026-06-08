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

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/config"
)

// NewConfigCommand builds the config command group.
func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `Inspect and edit local CLI configuration, including profiles and defaults.`,
		Example: `  semctl config get host
  semctl config set output json
  semctl config list`,
	}
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newSetCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newProfileCommand())
	return cmd
}

func newGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <KEY>",
		Short: "Get a configuration value",
		Long:  `Read a single value from the active profile in the config file.`,
		Example: `  semctl config get host
  semctl config get current_profile`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			v, err := cfg.Get(args[0])
			if err != nil {
				return err
			}
			fmt.Println(v)
			return nil
		},
	}
}

func newSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <KEY> <VALUE>",
		Short: "Set a configuration value",
		Long:  `Write a value to the active profile in the config file.`,
		Example: `  semctl config set output json
  semctl config set project infra`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			if err := cfg.Set(args[0], args[1]); err != nil {
				return err
			}
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			return nil
		},
	}
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List configuration values",
		Long:    `Show the current profile and key settings from the config file.`,
		Example: `  semctl config list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			fmt.Printf("current_profile: %s\n", cfg.CurrentProfile)
			return nil
		},
	}
}

func newProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage profiles",
		Long:  `Create, switch between, and remove named profiles that store host and project defaults.`,
		Example: `  semctl config profile list
  semctl config profile use prod
  semctl config profile create lab --host https://lab.example.com`,
	}
	cmd.AddCommand(newProfileListCommand())
	cmd.AddCommand(newProfileUseCommand())
	cmd.AddCommand(newProfileCreateCommand())
	cmd.AddCommand(newProfileDeleteCommand())
	return cmd
}

func newProfileListCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List profiles",
		Long:    `Show all configured profiles and mark the active one with an asterisk.`,
		Example: `  semctl config profile list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			for name := range cfg.Profiles {
				marker := " "
				if name == cfg.CurrentProfile {
					marker = "*"
				}
				fmt.Printf("%s %s\n", marker, name)
			}
			return nil
		},
	}
}

func newProfileUseCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "use <NAME>",
		Short:   "Switch to a profile",
		Long:    `Change the active profile used for all subsequent commands.`,
		Example: `  semctl config profile use prod`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			name := args[0]
			if cfg.Profiles[name] == nil {
				return fmt.Errorf("profile not found: %s", name)
			}
			cfg.CurrentProfile = name
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("✓ Active profile is now %s\n", name)
			return nil
		},
	}
}

func newProfileCreateCommand() *cobra.Command {
	var host string
	cmd := &cobra.Command{
		Use:   "create <NAME>",
		Short: "Create a new profile",
		Long:  `Create a profile with an optional host URL. Use 'semctl config set host <url>' afterwards if --host is omitted.`,
		Example: `  semctl config profile create prod --host https://semaphore.example.com
  semctl config profile create lab`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			name := args[0]
			if cfg.Profiles[name] != nil {
				return fmt.Errorf("profile already exists: %s", name)
			}
			cfg.Profiles[name] = &config.Profile{Host: host}
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("✓ Created profile %s\n", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&host, "host", "", "Semaphore UI host URL")
	return cmd
}

func newProfileDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <NAME>",
		Short:   "Delete a profile",
		Long:    `Remove a profile from the config file. If it is the active profile, the active profile is cleared.`,
		Example: `  semctl config profile delete lab`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			name := args[0]
			if cfg.Profiles[name] == nil {
				return fmt.Errorf("profile not found: %s", name)
			}
			delete(cfg.Profiles, name)
			if cfg.CurrentProfile == name {
				cfg.CurrentProfile = ""
			}
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Printf("✓ Deleted profile %s\n", name)
			return nil
		},
	}
}
