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
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

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

var version = "dev"

func main() {
	root := newRootCommand()
	if err := root.Execute(); err != nil {
		code, _ := formatError(root, err, os.Stderr)
		os.Exit(resolveExitCode(root, code))
	}
}

// newRootCommand builds the fully wired root command. Extracted from main so it
// can be exercised in tests (e.g. flag-error classification).
func newRootCommand() *cobra.Command {
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

// resolveExitCode decides the process exit code. By default every failure exits
// 1 (backward compatible). When --rich-exit-codes or SEMCTL_RICH_EXIT is set,
// the per-class exit code is used instead, so automation can branch on it.
func resolveExitCode(cmd *cobra.Command, classCode int) int {
	rich, _ := cmd.PersistentFlags().GetBool("rich-exit-codes")
	if !rich && os.Getenv("SEMCTL_RICH_EXIT") == "" {
		return 1
	}
	if classCode <= 0 {
		return 1
	}
	return classCode
}

// formatError renders a top-level error as a structured semerr class, honoring
// the --json / --output / --verbose flags. It reads the flags off the command
// so it stays in sync with how the rest of the CLI resolves output mode.
//
// The returned int is the error class's exit code. The process currently always
// exits 1 (see main) for backward compatibility; surfacing the class exit code
// here lets a future opt-in or major release activate richer exit codes in one
// place without touching call sites.
func formatError(cmd *cobra.Command, err error, w io.Writer) (int, error) {
	se := semerr.Classify(err)
	if se == nil {
		return 0, nil
	}
	jsonFlag, _ := cmd.PersistentFlags().GetBool("json")
	outputFlag, _ := cmd.PersistentFlags().GetString("output")
	verbose, _ := cmd.PersistentFlags().GetBool("verbose")
	debug, _ := cmd.PersistentFlags().GetBool("debug")
	switch {
	case jsonFlag || outputFlag == "json":
		return se.ExitCode, json.NewEncoder(w).Encode(map[string]any{"error": se.Payload()})
	case outputFlag == "yaml":
		return se.ExitCode, yaml.NewEncoder(w).Encode(map[string]any{"error": se.Payload()})
	default:
		se.WriteHuman(w, verbose)
		// --debug surfaces the underlying cause (which may carry the response
		// body) that the default message deliberately omits.
		if debug {
			if cause := se.Cause(); cause != nil {
				fmt.Fprintf(w, "\nCause (debug):\n  %v\n", cause)
			}
		}
		return se.ExitCode, nil
	}
}
