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
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/cli"
	"github.com/moep90/semaphore-cli/internal/output"
)

// NewTaskCommand builds the task command group.
func NewTaskCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage tasks",
		Long:  `List, inspect, run, stop, and watch tasks. Tasks are the core operational workflow in Semaphore UI.`,
		Example: `  semctl task list
  semctl task run deploy-prod --message "Deploy 1.8.3" --watch --exit-code
  semctl task logs --follow
  semctl task stop 812`,
	}
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newLastCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newRunCommand())
	cmd.AddCommand(newStopCommand())
	cmd.AddCommand(newLogsCommand())
	cmd.AddCommand(newWatchCommand())
	return cmd
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		Long:  `Show tasks in the active project.`,
		Example: `  semctl task list
  semctl task list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			resp, err := ctx.Client.Do(cmd.Context(), "GET", fmt.Sprintf("/project/%d/tasks", projectID), nil)
			if err != nil {
				return fmt.Errorf("list tasks: %w", err)
			}
			var tasks []api.Task
			if err := api.DecodeJSON(resp, &tasks); err != nil {
				return fmt.Errorf("decode tasks: %w", err)
			}
			rows := make([][]string, len(tasks))
			for i, t := range tasks {
				rows[i] = []string{
					strconv.Itoa(t.ID),
					strconv.Itoa(t.TemplateID),
					t.Status,
					t.Message,
					t.Created.Format("2006-01-02 15:04"),
				}
			}
			return ctx.Printer.PrintTable([]string{"ID", "TEMPLATE", "STATUS", "MESSAGE", "CREATED"}, rows)
		},
	}
}

func newLastCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "last",
		Short: "Show the latest task",
		Long:  `Show details for the most recently created task in the active project.`,
		Example: `  semctl task last
  semctl task last --output yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			taskID, err := ctx.LatestTaskID(cmd.Context())
			if err != nil {
				return err
			}
			return printTask(cmd.Context(), ctx, taskID)
		},
	}
}

func newGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <TASK>",
		Short: "Get task details",
		Long:  `Show full details for a task. Accepts a task ID or name.`,
		Example: `  semctl task get 812
  semctl task get 812 --output yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			taskID, err := ctx.ResolveTaskID(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return printTask(cmd.Context(), ctx, taskID)
		},
	}
}

func printTask(ctx context.Context, c *cli.Context, taskID int) error {
	projectID, err := c.ResolveProjectID(ctx)
	if err != nil {
		return err
	}
	resp, err := c.Client.Do(ctx, "GET", fmt.Sprintf("/project/%d/tasks/%d", projectID, taskID), nil)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}
	var task api.Task
	if err := api.DecodeJSON(resp, &task); err != nil {
		return fmt.Errorf("decode task: %w", err)
	}
	return c.Printer.Print(task)
}

func newRunCommand() *cobra.Command {
	var message string
	var branch string
	var environment string
	var inventory string
	var limit string
	var diff bool
	var dryRun bool
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
  4  CLI or API error`,
		Example: `  semctl task run deploy-prod --message "Deploy release 1.8.3"
  semctl task run deploy-prod --watch --exit-code
  semctl task run deploy-prod --message "hotfix" --branch release/1.8
  semctl task run deploy-prod --environment 1 --inventory 2 --limit "web*" --diff --dry-run`,
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

			body := map[string]any{
				"template_id": templateID,
			}
			if message != "" {
				body["message"] = message
			}
			if branch != "" {
				body["git_branch"] = branch
			}
			if environment != "" {
				body["environment_id"] = environment
			}
			if inventory != "" {
				body["inventory_id"] = inventory
			}
			if limit != "" {
				body["limit"] = limit
			}
			if diff {
				body["diff"] = true
			}
			if dryRun {
				body["dry_run"] = true
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
			_, _ = fmt.Fprintf(ctx.Printer.Stdout, "\nView logs:\n  semctl task logs %d --follow\n", task.ID)

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
	cmd.Flags().BoolVar(&watch, "watch", false, "Wait for the task to complete")
	cmd.Flags().BoolVar(&exitCode, "exit-code", false, "Return task status as process exit code")
	return cmd
}

func newStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "stop <TASK>",
		Short:   "Stop a running task",
		Long:    `Request that a running or pending task be stopped. Accepts a task ID.`,
		Example: `  semctl task stop 812`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			taskID, err := ctx.ResolveTaskID(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			projectID, _ := ctx.ResolveProjectID(cmd.Context())
			_, err = ctx.Client.Do(cmd.Context(), "POST", fmt.Sprintf("/project/%d/tasks/%d/stop", projectID, taskID), nil)
			if err != nil {
				return fmt.Errorf("stop task: %w", err)
			}
			ctx.Printer.PrintSuccess(fmt.Sprintf("Stopped task %d", taskID))
			return nil
		},
	}
}

func newLogsCommand() *cobra.Command {
	var follow bool
	var tail int
	var raw bool
	var interval time.Duration
	var escapeSanitize bool
	cmd := &cobra.Command{
		Use:   "logs [TASK]",
		Short: "View task logs",
		Long: `Print task output. If no task is given, uses the latest task in the active project.

In follow mode, new output is polled and printed until the task finishes.
ANSI escape sequences are stripped by default in TTY mode to prevent malicious
playbooks from hiding text or altering the terminal; use --escape-sanitize=false
to preserve them.`,
		Example: `  semctl task logs 812
  semctl task logs --follow
  semctl task logs 812 --tail 50 --raw
  semctl task logs 812 --follow --interval 5s`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}

			var taskID int
			if len(args) > 0 {
				taskID, err = ctx.ResolveTaskID(cmd.Context(), args[0])
				if err != nil {
					return err
				}
			} else {
				taskID, err = ctx.LatestTaskID(cmd.Context())
				if err != nil {
					return err
				}
			}

			projectID, _ := ctx.ResolveProjectID(cmd.Context())
			path := fmt.Sprintf("/project/%d/tasks/%d/output", projectID, taskID)

			if !follow {
				resp, err := ctx.Client.Do(cmd.Context(), "GET", path, nil)
				if err != nil {
					return fmt.Errorf("fetch logs: %w", err)
				}
				var outputs []api.TaskOutput
				if err := api.DecodeJSON(resp, &outputs); err != nil {
					return fmt.Errorf("decode logs: %w", err)
				}
				start := 0
				if tail > 0 && tail < len(outputs) {
					start = len(outputs) - tail
				}
				for _, o := range outputs[start:] {
					out := o.Output
					if escapeSanitize {
						out = output.SanitizeANSI(out)
					}
					if raw {
						_, _ = fmt.Fprintln(ctx.Printer.Stdout, out)
					} else {
						_, _ = fmt.Fprintf(ctx.Printer.Stdout, "[%s] %s\n", o.Time, out)
					}
				}
				return nil
			}

			// Follow mode with polling.
			seen := make(map[string]bool)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				resp, err := ctx.Client.Do(cmd.Context(), "GET", path, nil)
				if err != nil {
					return fmt.Errorf("poll logs: %w", err)
				}
				var outputs []api.TaskOutput
				if err := api.DecodeJSON(resp, &outputs); err != nil {
					return fmt.Errorf("decode logs: %w", err)
				}
				for i, o := range outputs {
					key := fmt.Sprintf("%d|%s|%s", i, o.Time, o.Output)
					if !seen[key] {
						seen[key] = true
						out := o.Output
						if escapeSanitize {
							out = output.SanitizeANSI(out)
						}
						if raw {
							_, _ = fmt.Fprintln(ctx.Printer.Stdout, out)
						} else {
							_, _ = fmt.Fprintf(ctx.Printer.Stdout, "[%s] %s\n", o.Time, out)
						}
					}
				}

				// Check if task is done.
				resp2, err := ctx.Client.Do(cmd.Context(), "GET", fmt.Sprintf("/project/%d/tasks/%d", projectID, taskID), nil)
				if err == nil {
					var task api.Task
					if err := api.DecodeJSON(resp2, &task); err == nil {
						if strings.EqualFold(task.Status, "success") || strings.EqualFold(task.Status, "error") || strings.EqualFold(task.Status, "stopped") {
							return nil
						}
					}
				}

				select {
				case <-ticker.C:
					continue
				case <-cmd.Context().Done():
					return cmd.Context().Err()
				}
			}
		},
	}
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Stream new output")
	cmd.Flags().IntVar(&tail, "tail", 0, "Limit initial output lines")
	cmd.Flags().BoolVar(&raw, "raw", false, "Skip formatting")
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "Polling interval")
	cmd.Flags().BoolVar(&escapeSanitize, "escape-sanitize", true, "Strip ANSI escape sequences from task output")
	return cmd
}

func newWatchCommand() *cobra.Command {
	var exitCode bool
	cmd := &cobra.Command{
		Use:   "watch [TASK]",
		Short: "Wait for a task to complete",
		Long: `Poll a task until it reaches a terminal state (success, error, or stopped).

If no task is given, watches the latest task in the active project.
With --exit-code, the CLI exits with a status-specific code:

  0  task succeeded
  1  task failed
  2  task stopped or canceled
  3  timeout / context cancelled
  4  CLI or API error`,
		Example: `  semctl task watch 812
  semctl task watch --exit-code
  semctl task run deploy-prod --watch --exit-code`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}

			var taskID int
			if len(args) > 0 {
				taskID, err = ctx.ResolveTaskID(cmd.Context(), args[0])
				if err != nil {
					return err
				}
			} else {
				taskID, err = ctx.LatestTaskID(cmd.Context())
				if err != nil {
					return err
				}
			}

			err = watchTask(cmd.Context(), ctx, taskID, exitCode)
			if err != nil {
				if exitCode {
					// Return special exit codes via cobra error handling is tricky;
					// we print and exit manually for CI compatibility.
					_, _ = fmt.Fprintln(ctx.Printer.Stderr, err)
					os.Exit(taskExitCode(err))
				}
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&exitCode, "exit-code", false, "Return task status as process exit code")
	return cmd
}

func watchTask(ctx context.Context, c *cli.Context, taskID int, returnExitCode bool) error {
	projectID, err := c.ResolveProjectID(ctx)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		resp, err := c.Client.Do(ctx, "GET", fmt.Sprintf("/project/%d/tasks/%d", projectID, taskID), nil)
		if err != nil {
			return fmt.Errorf("watch task: %w", err)
		}
		var task api.Task
		if err := api.DecodeJSON(resp, &task); err != nil {
			return fmt.Errorf("decode task: %w", err)
		}

		if strings.EqualFold(task.Status, "success") {
			if returnExitCode {
				return &exitCodeError{code: 0}
			}
			c.Printer.PrintSuccess(fmt.Sprintf("Task %d succeeded", taskID))
			return nil
		}
		if strings.EqualFold(task.Status, "error") || strings.EqualFold(task.Status, "failed") {
			if returnExitCode {
				return &exitCodeError{code: 1}
			}
			return fmt.Errorf("task %d failed", taskID)
		}
		if strings.EqualFold(task.Status, "stopped") || strings.EqualFold(task.Status, "canceled") {
			if returnExitCode {
				return &exitCodeError{code: 2}
			}
			return fmt.Errorf("task %d stopped", taskID)
		}

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			if returnExitCode {
				return &exitCodeError{code: 3}
			}
			return ctx.Err()
		}
	}
}

type exitCodeError struct {
	code int
	msg  string
}

func (e *exitCodeError) Error() string {
	if e.msg != "" {
		return e.msg
	}
	return fmt.Sprintf("exit code %d", e.code)
}

func taskExitCode(err error) int {
	if e, ok := err.(*exitCodeError); ok {
		return e.code
	}
	return 4
}
