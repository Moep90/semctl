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
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/commands/api"
	"github.com/moep90/semaphore-cli/internal/commands/auth"
	"github.com/moep90/semaphore-cli/internal/commands/config"
	"github.com/moep90/semaphore-cli/internal/commands/info"
	"github.com/moep90/semaphore-cli/internal/commands/ping"
	"github.com/moep90/semaphore-cli/internal/commands/project"
	"github.com/moep90/semaphore-cli/internal/commands/task"
	"github.com/moep90/semaphore-cli/internal/commands/template"
)

var (
	version       = "dev"
	hostFlag      string
	projectFlag   string
	outputFlag    string
	profileFlag   string
	jsonFlag      bool
	noColor       bool
	verboseFlag   bool
	debugFlag     bool
	noInteractive bool
)

func main() {
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

	root.PersistentFlags().StringVar(&hostFlag, "host", "", "Semaphore UI host URL")
	root.PersistentFlags().StringVarP(&projectFlag, "project", "p", "", "Default project")
	root.PersistentFlags().StringVarP(&outputFlag, "output", "o", "", "Output format (table, json, yaml, text)")
	root.PersistentFlags().StringVar(&profileFlag, "profile", "", "Configuration profile")
	root.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output in JSON format")
	root.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable colored output")
	root.PersistentFlags().BoolVar(&verboseFlag, "verbose", false, "Verbose output")
	root.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Debug output")
	root.PersistentFlags().BoolVar(&noInteractive, "no-interactive", false, "Disable interactive prompts")

	root.AddCommand(auth.NewAuthCommand())
	root.AddCommand(config.NewConfigCommand())
	root.AddCommand(api.NewAPICommand())
	root.AddCommand(project.NewProjectCommand())
	root.AddCommand(template.NewTemplateCommand())
	root.AddCommand(task.NewTaskCommand())
	root.AddCommand(info.NewInfoCommand())
	root.AddCommand(ping.NewPingCommand())

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
