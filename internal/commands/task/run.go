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

package task

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/cli"
	"github.com/moep90/semaphore-cli/internal/output"
)

func newRunCommand() *cobra.Command {
	var message string
	var branch string
	var environment string
	var inventory string
	var limit string
	var diff bool
	var dryRun bool
	var tags string
	var skipTags string
	var extraVars string
	var check bool
	var watch bool
	var exitCode bool
	cmd := &cobra.Command{
		Use:   "run <TEMPLATE>",
		Short: "Run a template",
		Long: `Queue a new task from a template in the active project.

Use --watch to block until the task finishes. With --exit-code, the CLI returns
a status-specific exit code suitable for CI pipelines:

  0  task succeeded
  1  task failed
  2  task stopped or canceled
  3  timeout
  4  CLI or API error

Ansible execution can be tuned with --tags, --skip-tags, --extra-vars, and
--check. --check enables Ansible check mode (a dry run on the target hosts),
which is distinct from --dry-run.`,
		Example: `  semctl task run deploy-prod --message "Deploy release 1.8.3"
  semctl task run deploy-prod --watch --exit-code
  semctl task run deploy-prod --message "hotfix" --branch release/1.8
  semctl task run deploy-prod --environment 1 --inventory 2 --limit "web*" --diff --dry-run
  semctl task run deploy-prod --tags deploy,restart --skip-tags slow
  semctl task run deploy-prod --extra-vars '{"version":"1.2.3"}' --check`,
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

			body := api.TaskRunRequest{
				TemplateID: templateID,
				Message:    message,
				GitBranch:  branch,
			}
			if environment != "" {
				envID, err := ctx.ResolveEnvironmentID(cmd.Context(), environment)
				if err != nil {
					return fmt.Errorf("resolve environment: %w", err)
				}
				body.EnvironmentID = envID
			}
			if inventory != "" {
				invID, err := ctx.ResolveInventoryID(cmd.Context(), inventory)
				if err != nil {
					return fmt.Errorf("resolve inventory: %w", err)
				}
				body.InventoryID = invID
			}
			if limit != "" {
				body.Limit = limit
			}
			if diff {
				body.Diff = true
			}
			if dryRun {
				body.DryRun = true
			}
			if tags != "" {
				body.Tags = tags
			}
			if skipTags != "" {
				body.SkipTags = skipTags
			}
			if extraVars != "" {
				// Semaphore silently drops malformed extra vars, so validate
				// that the value is a JSON object before submitting (issue #81).
				var probe map[string]any
				if err := json.Unmarshal([]byte(extraVars), &probe); err != nil {
					return fmt.Errorf("--extra-vars must be a valid JSON object: %w", err)
				}
				body.ExtraVars = extraVars
			}
			if check {
				body.Check = true
			}

			resp, err := ctx.Client.Do(cmd.Context(), "POST", fmt.Sprintf("/project/%d/tasks", projectID), body)
			if err != nil {
				return fmt.Errorf("run template: %w", err)
			}
			var task api.Task
			if err := api.DecodeJSON(resp, &task); err != nil {
				return fmt.Errorf("decode task: %w", err)
			}

			ctx.Printer.PrintSuccess(fmt.Sprintf("Queued task %d from template %s", task.ID, args[0]))
			if ctx.Printer.Mode == output.ModeTable || ctx.Printer.Mode == output.ModeText {
				_, _ = fmt.Fprintf(ctx.Printer.Stdout, "\nView logs:\n  semctl task logs %d --follow\n", task.ID)
			}

			if watch {
				return watchTask(cmd.Context(), ctx, task.ID, exitCode)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&message, "message", "", "Task message")
	cmd.Flags().StringVar(&branch, "branch", "", "Git branch override")
	cmd.Flags().StringVar(&environment, "environment", "", "Environment ID or name")
	cmd.Flags().StringVar(&inventory, "inventory", "", "Inventory ID or name")
	cmd.Flags().StringVar(&limit, "limit", "", "Ansible limit pattern")
	cmd.Flags().BoolVar(&diff, "diff", false, "Show diff mode")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Dry run mode")
	cmd.Flags().StringVar(&tags, "tags", "", "Ansible tags to run (comma-separated)")
	cmd.Flags().StringVar(&skipTags, "skip-tags", "", "Ansible tags to skip (comma-separated)")
	cmd.Flags().StringVar(&extraVars, "extra-vars", "", "Ansible extra variables as a raw JSON object (e.g. '{\"version\":\"1.2.3\"}')")
	cmd.Flags().BoolVar(&check, "check", false, "Ansible check mode (dry run on target hosts; distinct from --dry-run)")
	cmd.Flags().BoolVar(&watch, "watch", false, "Wait for the task to complete")
	cmd.Flags().BoolVar(&exitCode, "exit-code", false, "Return task status as process exit code")
	return cmd
}
