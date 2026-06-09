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
		Long:  `List, inspect, create, update, and delete environments within the active project.`,
		Example: `  semctl environment list
  semctl environment get staging-env
  semctl environment create --name staging-env --json '{"KEY":"value"}'
  semctl environment update staging-env --json '{"KEY":"new"}'
  semctl environment delete staging-env`,
	}
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newUpdateCommand())
	cmd.AddCommand(newDeleteCommand())
	return cmd
}

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List environments",
		Long:  `Show all environments in the active project.`,
		Example: `  semctl environment list
  semctl environment list --json
  semctl environment list --limit 20 --page 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/project/%d/environment", projectID) + cli.PaginationQuery(cmd)
			resp, err := ctx.Client.Do(cmd.Context(), "GET", path, nil)
			if err != nil {
				return fmt.Errorf("list environment: %w", err)
			}
			var environments []api.Environment
			if err := api.DecodeJSON(resp, &environments); err != nil {
				return fmt.Errorf("decode environment: %w", err)
			}
			environments = cli.Paginate(environments, cmd)
			rows := make([][]string, len(environments))
			for i, env := range environments {
				rows[i] = []string{
					strconv.Itoa(env.ID),
					env.Name,
				}
			}
			return ctx.Printer.PrintList([]string{"ID", "NAME"}, rows, environments)
		},
	}
	cli.AddPaginationFlags(cmd)
	return cmd
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

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an environment",
		Long:  `Create a new environment in the active project.`,
		Example: `  semctl environment create --name staging-env
  semctl environment create --name staging-env --json '{"KEY":"value"}'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			name, _ := cmd.Flags().GetString("name")
			envJSON, _ := cmd.Flags().GetString("json")
			body := map[string]any{
				"name":       name,
				"project_id": projectID,
				"json":       envJSON,
			}
			resp, err := ctx.Client.Do(cmd.Context(), "POST", fmt.Sprintf("/project/%d/environment", projectID), body)
			if err != nil {
				return fmt.Errorf("create environment: %w", err)
			}
			if err := api.CheckResponse(resp); err != nil {
				return fmt.Errorf("create environment: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Created environment %s\n", name)
			return nil
		},
	}
	cmd.Flags().String("name", "", "Environment name")
	cmd.Flags().String("json", "{}", "Environment variables as a JSON string")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <ENVIRONMENT>",
		Short: "Update an environment",
		Long:  `Update an environment. Accepts an environment ID or name. Only changed fields are sent.`,
		Example: `  semctl environment update staging-env --json '{"KEY":"new"}'
  semctl environment update 5 --name prod-env`,
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
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			body := map[string]any{
				"id":         environmentID,
				"project_id": projectID,
			}
			if cmd.Flags().Changed("name") {
				name, _ := cmd.Flags().GetString("name")
				body["name"] = name
			}
			if cmd.Flags().Changed("json") {
				envJSON, _ := cmd.Flags().GetString("json")
				body["json"] = envJSON
			}
			resp, err := ctx.Client.Do(cmd.Context(), "PUT", fmt.Sprintf("/project/%d/environment/%d", projectID, environmentID), body)
			if err != nil {
				return fmt.Errorf("update environment: %w", err)
			}
			if err := api.CheckResponse(resp); err != nil {
				return fmt.Errorf("update environment: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Updated environment %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().String("name", "", "Environment name")
	cmd.Flags().String("json", "{}", "Environment variables as a JSON string")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <ENVIRONMENT>",
		Short: "Delete an environment",
		Long:  `Delete an environment. Accepts an environment ID or name.`,
		Example: `  semctl environment delete staging-env
  semctl environment delete 5`,
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
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			resp, err := ctx.Client.Do(cmd.Context(), "DELETE", fmt.Sprintf("/project/%d/environment/%d", projectID, environmentID), nil)
			if err != nil {
				return fmt.Errorf("delete environment: %w", err)
			}
			if err := api.CheckResponse(resp); err != nil {
				return fmt.Errorf("delete environment: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Deleted environment %s\n", args[0])
			return nil
		},
	}
}
