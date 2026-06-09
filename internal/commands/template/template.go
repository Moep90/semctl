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

package template

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/cli"
)

// paginationQuery builds the "?count=<limit>&page=<page>" query string from the
// --limit and --page flags, including only the flags that were explicitly set.
// It returns an empty string when neither flag is set, preserving the
// unpaginated request behavior.
func paginationQuery(cmd *cobra.Command) string {
	q := url.Values{}
	if cmd.Flags().Changed("limit") {
		limit, _ := cmd.Flags().GetInt("limit")
		q.Set("count", strconv.Itoa(limit))
	}
	if cmd.Flags().Changed("page") {
		page, _ := cmd.Flags().GetInt("page")
		q.Set("page", strconv.Itoa(page))
	}
	if len(q) == 0 {
		return ""
	}
	return "?" + q.Encode()
}

// addPaginationFlags registers the --limit and --page pagination flags.
func addPaginationFlags(cmd *cobra.Command) {
	cmd.Flags().Int("limit", 0, "Maximum number of items to return per page")
	cmd.Flags().Int("page", 0, "Page number to retrieve (1-based)")
}

// NewTemplateCommand builds the template command group.
func NewTemplateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage templates",
		Long:  `List and inspect task templates within the active project.`,
		Example: `  semctl template list
  semctl template get deploy-prod`,
	}
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newDeleteCommand())
	cmd.AddCommand(newCloneCommand())
	cmd.AddCommand(newTasksCommand())
	return cmd
}

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List templates",
		Long:  `Show all task templates in the active project.`,
		Example: `  semctl template list
  semctl template list --json
  semctl template list --limit 20 --page 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/project/%d/templates", projectID) + paginationQuery(cmd)
			resp, err := ctx.Client.Do(cmd.Context(), "GET", path, nil)
			if err != nil {
				return fmt.Errorf("list templates: %w", err)
			}
			var templates []api.Template
			if err := api.DecodeJSON(resp, &templates); err != nil {
				return fmt.Errorf("decode templates: %w", err)
			}
			rows := make([][]string, len(templates))
			for i, t := range templates {
				rows[i] = []string{
					strconv.Itoa(t.ID),
					t.Name,
					t.App,
					t.Playbook,
					t.Repository,
					t.Inventory,
					t.Environment,
				}
			}
			return ctx.Printer.PrintTable([]string{"ID", "NAME", "APP", "PLAYBOOK", "REPOSITORY", "INVENTORY", "ENVIRONMENT"}, rows)
		},
	}
	addPaginationFlags(cmd)
	return cmd
}

func newGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <TEMPLATE>",
		Short: "Get template details",
		Long:  `Show full details for a template. Accepts a template ID or name.`,
		Example: `  semctl template get deploy-prod
  semctl template get 7 --output yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			templateID, err := ctx.ResolveTemplateID(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			projectID, _ := ctx.ResolveProjectID(cmd.Context())
			resp, err := ctx.Client.Do(cmd.Context(), "GET", fmt.Sprintf("/project/%d/templates/%d", projectID, templateID), nil)
			if err != nil {
				return fmt.Errorf("get template: %w", err)
			}
			var template api.Template
			if err := api.DecodeJSON(resp, &template); err != nil {
				return fmt.Errorf("decode template: %w", err)
			}
			return ctx.Printer.Print(template)
		},
	}
}

func newDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <TEMPLATE>",
		Short: "Delete a template",
		Long:  `Delete a task template. Accepts a template ID or name.`,
		Example: `  semctl template delete deploy-prod
  semctl template delete 7`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			templateID, err := ctx.ResolveTemplateID(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			projectID, _ := ctx.ResolveProjectID(cmd.Context())
			_, err = ctx.Client.Do(cmd.Context(), "DELETE", fmt.Sprintf("/project/%d/templates/%d", projectID, templateID), nil)
			if err != nil {
				return fmt.Errorf("delete template: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Deleted template %s\n", args[0])
			return nil
		},
	}
}

func newCloneCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clone <TEMPLATE> <NEW_NAME>",
		Short: "Clone a template",
		Long:  `Clone an existing task template with a new name. Accepts a template ID or name.`,
		Example: `  semctl template clone deploy-prod deploy-staging
  semctl template clone 7 deploy-staging`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			templateID, err := ctx.ResolveTemplateID(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			projectID, _ := ctx.ResolveProjectID(cmd.Context())
			body := map[string]string{"name": args[1]}
			_, err = ctx.Client.Do(cmd.Context(), "POST", fmt.Sprintf("/project/%d/templates/%d/clone", projectID, templateID), body)
			if err != nil {
				return fmt.Errorf("clone template: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Cloned template %s to %s\n", args[0], args[1])
			return nil
		},
	}
}

func newTasksCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tasks <TEMPLATE>",
		Short: "List tasks for a template",
		Long:  `Show all tasks associated with a template. Accepts a template ID or name.`,
		Example: `  semctl template tasks deploy-prod
  semctl template tasks 7`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			templateID, err := ctx.ResolveTemplateID(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			projectID, _ := ctx.ResolveProjectID(cmd.Context())
			resp, err := ctx.Client.Do(cmd.Context(), "GET", fmt.Sprintf("/project/%d/templates/%d/tasks", projectID, templateID), nil)
			if err != nil {
				return fmt.Errorf("list template tasks: %w", err)
			}
			var tasks []api.Task
			if err := api.DecodeJSON(resp, &tasks); err != nil {
				return fmt.Errorf("decode tasks: %w", err)
			}
			rows := make([][]string, len(tasks))
			for i, t := range tasks {
				rows[i] = []string{
					strconv.Itoa(t.ID),
					t.Status,
					t.Message,
					t.Created.Format("2006-01-02"),
				}
			}
			return ctx.Printer.PrintTable([]string{"ID", "STATUS", "MESSAGE", "CREATED"}, rows)
		},
	}
}
