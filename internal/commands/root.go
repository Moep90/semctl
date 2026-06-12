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

// Package commands assembles the fully wired semctl command tree. It lives in
// an importable package (rather than package main) so that both the binary and
// the documentation generator build the exact same command set, guaranteeing
// docs/commands.md never drifts from the shipped CLI.
package commands

//go:generate go run ../../cmd/gen-commands-doc

import (
	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/cli"
	"github.com/moep90/semaphore-cli/internal/commands/api"
	"github.com/moep90/semaphore-cli/internal/commands/auth"
	"github.com/moep90/semaphore-cli/internal/commands/config"
	"github.com/moep90/semaphore-cli/internal/commands/environment"
	"github.com/moep90/semaphore-cli/internal/commands/info"
	"github.com/moep90/semaphore-cli/internal/commands/inventory"
	"github.com/moep90/semaphore-cli/internal/commands/keystore"
	"github.com/moep90/semaphore-cli/internal/commands/ping"
	"github.com/moep90/semaphore-cli/internal/commands/project"
	"github.com/moep90/semaphore-cli/internal/commands/schedule"
	"github.com/moep90/semaphore-cli/internal/commands/task"
	"github.com/moep90/semaphore-cli/internal/commands/template"
	"github.com/moep90/semaphore-cli/internal/semerr"
)

// NewRootCommand builds the fully wired root command. The version string is
// injected by the binary (via -ldflags) and reported by `semctl --version`.
func NewRootCommand(version string) *cobra.Command {
	root := &cobra.Command{
		Use:   "semctl",
		Short: "Semaphore UI CLI",
		Long: `A command line interface for Semaphore UI.

Disclaimer: semctl is an independent, open-source command line interface for
Semaphore UI. It is NOT affiliated with, endorsed by, sponsored by, or
officially connected to the Semaphore UI project or its creators. This tool
is intended for personal use, educational purposes, and operational
convenience at your own risk. All product names, logos, and brands are
property of their respective owners.`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: false,
		},
	}

	// Classify flag-parse errors (unknown flag, bad value) as a CLI-usage class.
	// Inherited by all subcommands unless they set their own.
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return semerr.New("SEM100003").WithMessage(err.Error()).Wrap(err)
	})

	cli.RegisterGlobalFlags(root)

	root.AddCommand(auth.NewAuthCommand())
	root.AddCommand(config.NewConfigCommand())
	root.AddCommand(api.NewAPICommand())
	root.AddCommand(project.NewProjectCommand())
	root.AddCommand(schedule.NewScheduleCommand())
	root.AddCommand(template.NewTemplateCommand())
	root.AddCommand(task.NewTaskCommand())
	root.AddCommand(inventory.NewInventoryCommand())
	root.AddCommand(environment.NewEnvironmentCommand())
	root.AddCommand(keystore.NewKeystoreCommand())
	root.AddCommand(info.NewInfoCommand())
	root.AddCommand(ping.NewPingCommand())

	return root
}
