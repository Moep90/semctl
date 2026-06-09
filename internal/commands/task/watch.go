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
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/cli"
)

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
