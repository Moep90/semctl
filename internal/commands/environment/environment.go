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

package environment

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/cli"
)

// NewEnvironmentCommand builds the environment command group.
func NewEnvironmentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "environment",
		Short: "Manage environments",
		Long:  `List and inspect environments within the active project.`,
		Example: `  semctl environment list
  semctl environment get staging-env`,
	}
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newGetCommand())
	return cmd
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List environments",
		Long:  `Show all environments in the active project.`,
		Example: `  semctl environment list
  semctl environment list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			resp, err := ctx.Client.Do(cmd.Context(), "GET", fmt.Sprintf("/project/%d/environment", projectID), nil)
			if err != nil {
				return fmt.Errorf("list environment: %w", err)
			}
			var environments []api.Environment
			if err := api.DecodeJSON(resp, &environments); err != nil {
				return fmt.Errorf("decode environment: %w", err)
			}
			rows := make([][]string, len(environments))
			for i, env := range environments {
				rows[i] = []string{
					strconv.Itoa(env.ID),
					env.Name,
				}
			}
			return ctx.Printer.PrintTable([]string{"ID", "NAME"}, rows)
		},
	}
}

func newGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <ENVIRONMENT>",
		Short: "Get environment details",
		Long:  `Show full details for an environment. Accepts an environment ID or name.`,
		Example: `  semctl environment get staging-env
  semctl environment get 5 --output yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			environmentID, err := ctx.ResolveEnvironmentID(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			projectID, _ := ctx.ResolveProjectID(cmd.Context())
			resp, err := ctx.Client.Do(cmd.Context(), "GET", fmt.Sprintf("/project/%d/environment/%d", projectID, environmentID), nil)
			if err != nil {
				return fmt.Errorf("get environment: %w", err)
			}
			var environment api.Environment
			if err := api.DecodeJSON(resp, &environment); err != nil {
				return fmt.Errorf("decode environment: %w", err)
			}
			return ctx.Printer.Print(environment)
		},
	}
}
