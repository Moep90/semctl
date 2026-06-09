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

package project

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/cli"
	"github.com/moep90/semaphore-cli/internal/config"
	"github.com/moep90/semaphore-cli/internal/resolver"
)

// NewProjectCommand builds the project command group.
func NewProjectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
		Long:  `List projects, inspect details, and set the default project for the active profile.`,
		Example: `  semctl project list
  semctl project get infra
  semctl project use infra`,
	}
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newDeleteCommand())
	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newSetCommand())
	cmd.AddCommand(newUseCommand())
	return cmd
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List projects",
		Long:  `Show all projects accessible on the current host.`,
		Example: `  semctl project list
  semctl project list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := buildCmdContext(cmd)
			if err != nil {
				return err
			}
			resp, err := ctx.Client.Do(cmd.Context(), "GET", "/projects", nil)
			if err != nil {
				return fmt.Errorf("list projects: %w", err)
			}
			var projects []api.Project
			if err := api.DecodeJSON(resp, &projects); err != nil {
				return fmt.Errorf("decode projects: %w", err)
			}
			rows := make([][]string, len(projects))
			for i, p := range projects {
				rows[i] = []string{
					strconv.Itoa(p.ID),
					p.Name,
					fmt.Sprintf("%d", p.MaxParallelTasks),
					p.Created.Format("2006-01-02"),
				}
			}
			return ctx.Printer.PrintTable([]string{"ID", "NAME", "MAX_PARALLEL_TASKS", "CREATED"}, rows)
		},
	}
}

func newGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <PROJECT>",
		Short: "Get project details",
		Long:  `Show full details for a project. Accepts a project ID or name.`,
		Example: `  semctl project get infra
  semctl project get 1 --output yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := buildCmdContext(cmd)
			if err != nil {
				return err
			}
			projectID, err := resolver.ResolveProject(cmd.Context(), ctx.Client, args[0])
			if err != nil {
				return err
			}
			resp, err := ctx.Client.Do(cmd.Context(), "GET", fmt.Sprintf("/project/%d", projectID), nil)
			if err != nil {
				return fmt.Errorf("get project: %w", err)
			}
			var project api.Project
			if err := api.DecodeJSON(resp, &project); err != nil {
				return fmt.Errorf("decode project: %w", err)
			}
			return ctx.Printer.Print(project)
		},
	}
}

func newDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <PROJECT>",
		Short: "Delete a project",
		Long:  `Delete a project. Accepts a project ID or name.`,
		Example: `  semctl project delete infra
  semctl project delete 1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := buildCmdContext(cmd)
			if err != nil {
				return err
			}
			projectID, err := resolver.ResolveProject(cmd.Context(), ctx.Client, args[0])
			if err != nil {
				return err
			}
			_, err = ctx.Client.Do(cmd.Context(), "DELETE", fmt.Sprintf("/project/%d", projectID), nil)
			if err != nil {
				return fmt.Errorf("delete project: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Deleted project %s\n", args[0])
			return nil
		},
	}
}

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a project",
		Long:  `Create a new project on the current host.`,
		Example: `  semctl project create --name infra
  semctl project create --name infra --max-parallel 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := buildCmdContext(cmd)
			if err != nil {
				return err
			}
			name, _ := cmd.Flags().GetString("name")
			maxParallel, _ := cmd.Flags().GetInt("max-parallel")
			body := map[string]any{"name": name}
			if cmd.Flags().Changed("max-parallel") {
				body["max_parallel_tasks"] = maxParallel
			}
			_, err = ctx.Client.Do(cmd.Context(), "POST", "/projects", body)
			if err != nil {
				return fmt.Errorf("create project: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Created project %s\n", name)
			return nil
		},
	}
	cmd.Flags().String("name", "", "Project name")
	cmd.Flags().Int("max-parallel", 0, "Maximum parallel tasks")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <PROJECT>",
		Short: "Set the default project for a profile",
		Long:  `Store the default project name in the active profile so commands that require a project use it automatically. Also supports updating host and output.`,
		Example: `  semctl project set infra
  semctl project set infra --host https://semaphore.example.com --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			name := cfg.CurrentProfile
			if name == "" {
				return fmt.Errorf("no active profile; create one with 'semctl config profile create'")
			}
			if cfg.Profiles[name] == nil {
				return fmt.Errorf("active profile not found: %s", name)
			}
			cfg.Profiles[name].Project = args[0]
			if cmd.Flags().Changed("host") {
				host, _ := cmd.Flags().GetString("host")
				cfg.Profiles[name].Host = host
			}
			if cmd.Flags().Changed("output") {
				output, _ := cmd.Flags().GetString("output")
				cfg.Profiles[name].DefaultOutput = output
			}
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Set project to %s for profile %s\n", args[0], name)
			return nil
		},
	}
	cmd.Flags().String("host", "", "Update profile host")
	cmd.Flags().StringP("output", "o", "", "Update profile default output")
	return cmd
}

func newUseCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "use <PROJECT>",
		Short:   "Set the default project for a profile",
		Long:    `Store the default project name in the active profile so commands that require a project use it automatically.`,
		Example: `  semctl project use infra`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			name := cfg.CurrentProfile
			if name == "" {
				return fmt.Errorf("no active profile; create one with 'semctl config profile create'")
			}
			if cfg.Profiles[name] == nil {
				return fmt.Errorf("active profile not found: %s", name)
			}
			cfg.Profiles[name].Project = args[0]
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Set project to %s for profile %s\n", args[0], name)
			return nil
		},
	}
}

func buildCmdContext(cmd *cobra.Command) (*cli.Context, error) {
	hostFlag, _ := cmd.Flags().GetString("host")
	projectFlag, _ := cmd.Flags().GetString("project")
	outputFlag, _ := cmd.Flags().GetString("output")
	profileFlag, _ := cmd.Flags().GetString("profile")
	jsonFlag, _ := cmd.Flags().GetBool("json")
	noColor, _ := cmd.Flags().GetBool("no-color")
	verbose, _ := cmd.Flags().GetBool("verbose")
	debug, _ := cmd.Flags().GetBool("debug")

	// Only apply --json shorthand when --output was not explicitly set.
	if jsonFlag && !cmd.Flags().Changed("output") {
		outputFlag = "json"
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return cli.BuildContext(cfg, hostFlag, projectFlag, outputFlag, profileFlag, noColor, verbose, debug)
}
