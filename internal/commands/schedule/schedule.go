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

package schedule

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/cli"
)

// NewScheduleCommand builds the schedule command group.
func NewScheduleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Manage schedules",
		Long:  `List, inspect, create, update, and delete cron schedules within the active project.`,
		Example: `  semctl schedule list
  semctl schedule get nightly-deploy
  semctl schedule create --template deploy-prod --cron "0 2 * * *" --name nightly-deploy`,
	}
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newUpdateCommand())
	cmd.AddCommand(newDeleteCommand())
	return cmd
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List schedules",
		Long:  `Show all schedules in the active project.`,
		Example: `  semctl schedule list
  semctl schedule list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			resp, err := ctx.Client.Do(cmd.Context(), "GET", fmt.Sprintf("/project/%d/schedules", projectID), nil)
			if err != nil {
				return fmt.Errorf("list schedules: %w", err)
			}
			var schedules []api.Schedule
			if err := api.DecodeJSON(resp, &schedules); err != nil {
				return fmt.Errorf("decode schedules: %w", err)
			}
			rows := make([][]string, len(schedules))
			for i, s := range schedules {
				rows[i] = []string{
					strconv.Itoa(s.ID),
					s.Name,
					strconv.Itoa(s.TemplateID),
					s.CronFormat,
					strconv.FormatBool(s.Active),
				}
			}
			return ctx.Printer.PrintTable([]string{"ID", "NAME", "TEMPLATE", "CRON", "ENABLED"}, rows)
		},
	}
}

func newGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <SCHEDULE>",
		Short: "Get schedule details",
		Long:  `Show full details for a schedule. Accepts a schedule ID or name.`,
		Example: `  semctl schedule get nightly-deploy
  semctl schedule get 3 --output yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			scheduleID, err := ctx.ResolveScheduleID(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			projectID, _ := ctx.ResolveProjectID(cmd.Context())
			resp, err := ctx.Client.Do(cmd.Context(), "GET", fmt.Sprintf("/project/%d/schedules/%d", projectID, scheduleID), nil)
			if err != nil {
				return fmt.Errorf("get schedule: %w", err)
			}
			var schedule api.Schedule
			if err := api.DecodeJSON(resp, &schedule); err != nil {
				return fmt.Errorf("decode schedule: %w", err)
			}
			return ctx.Printer.Print(schedule)
		},
	}
}

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a schedule",
		Long:  `Create a new cron schedule for a template in the active project.`,
		Example: `  semctl schedule create --template deploy-prod --cron "0 2 * * *"
  semctl schedule create --template 7 --cron "0 2 * * *" --name nightly-deploy`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			templateArg, _ := cmd.Flags().GetString("template")
			templateID, err := ctx.ResolveTemplateID(cmd.Context(), templateArg)
			if err != nil {
				return err
			}
			cron, _ := cmd.Flags().GetString("cron")
			name, _ := cmd.Flags().GetString("name")
			// NOTE: cron_format and active follow the Semaphore API body schema.
			body := map[string]any{
				"project_id":  projectID,
				"template_id": templateID,
				"name":        name,
				"cron_format": cron,
				"active":      true,
			}
			resp, err := ctx.Client.Do(cmd.Context(), "POST", fmt.Sprintf("/project/%d/schedules", projectID), body)
			if err != nil {
				return fmt.Errorf("create schedule: %w", err)
			}
			if err := api.CheckResponse(resp); err != nil {
				return fmt.Errorf("create schedule: %w", err)
			}
			label := name
			if label == "" {
				label = cron
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Created schedule %s\n", label)
			return nil
		},
	}
	cmd.Flags().String("template", "", "Template ID or name (required)")
	cmd.Flags().String("cron", "", "Cron expression (required)")
	cmd.Flags().String("name", "", "Schedule name")
	_ = cmd.MarkFlagRequired("template")
	_ = cmd.MarkFlagRequired("cron")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <SCHEDULE>",
		Short: "Update a schedule",
		Long:  `Update an existing schedule. Accepts a schedule ID or name.`,
		Example: `  semctl schedule update nightly-deploy --cron "0 3 * * *"
  semctl schedule update 3 --name weekly-deploy --template deploy-prod`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			scheduleID, err := ctx.ResolveScheduleID(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			body := map[string]any{
				"id":         scheduleID,
				"project_id": projectID,
			}
			if cmd.Flags().Changed("template") {
				templateArg, _ := cmd.Flags().GetString("template")
				templateID, err := ctx.ResolveTemplateID(cmd.Context(), templateArg)
				if err != nil {
					return err
				}
				body["template_id"] = templateID
			}
			if cmd.Flags().Changed("cron") {
				cron, _ := cmd.Flags().GetString("cron")
				// NOTE: cron_format follows the Semaphore API body schema.
				body["cron_format"] = cron
			}
			if cmd.Flags().Changed("name") {
				name, _ := cmd.Flags().GetString("name")
				body["name"] = name
			}
			resp, err := ctx.Client.Do(cmd.Context(), "PUT", fmt.Sprintf("/project/%d/schedules/%d", projectID, scheduleID), body)
			if err != nil {
				return fmt.Errorf("update schedule: %w", err)
			}
			if err := api.CheckResponse(resp); err != nil {
				return fmt.Errorf("update schedule: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Updated schedule %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().String("template", "", "Template ID or name")
	cmd.Flags().String("cron", "", "Cron expression")
	cmd.Flags().String("name", "", "Schedule name")
	return cmd
}

func newDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <SCHEDULE>",
		Short: "Delete a schedule",
		Long:  `Delete a schedule. Accepts a schedule ID or name.`,
		Example: `  semctl schedule delete nightly-deploy
  semctl schedule delete 3`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			scheduleID, err := ctx.ResolveScheduleID(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			projectID, _ := ctx.ResolveProjectID(cmd.Context())
			resp, err := ctx.Client.Do(cmd.Context(), "DELETE", fmt.Sprintf("/project/%d/schedules/%d", projectID, scheduleID), nil)
			if err != nil {
				return fmt.Errorf("delete schedule: %w", err)
			}
			if err := api.CheckResponse(resp); err != nil {
				return fmt.Errorf("delete schedule: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Deleted schedule %s\n", args[0])
			return nil
		},
	}
}
