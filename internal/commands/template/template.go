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
	"strconv"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/cli"
	"github.com/spf13/cobra"
)

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
	return cmd
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List templates",
		Long:  `Show all task templates in the active project.`,
		Example: `  semctl template list
  semctl template list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			resp, err := ctx.Client.Do(cmd.Context(), "GET", fmt.Sprintf("/project/%d/templates", projectID), nil)
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
