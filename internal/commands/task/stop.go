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

func newStopCommand() *cobra.Command {
	var (
		force    bool
		wait     bool
		exitCode bool
		timeout  time.Duration
		interval time.Duration
	)
	cmd := &cobra.Command{
		Use:   "stop <TASK>",
		Short: "Stop a running task",
		Long: `Request that a running or pending task be stopped. Accepts a task ID.

By default this performs a graceful stop (the task moves to "stopping" and is
finalized once its runner reports back). A waiting/queued task that no runner
has picked up will not transition this way — use --force to mark it stopped
immediately, matching the "Force Stop" action in the Semaphore UI.

The stop request returns as soon as Semaphore accepts it. Use --wait to poll
until the task actually reaches a terminal state, bounded by --timeout
(default 5m; 0 waits indefinitely). With --exit-code the process exit status
reflects the task's final state:

  0  task finished successfully (before the stop took effect)
  1  task failed
  2  task stopped or canceled (the expected result of a stop)
  3  timed out waiting
  4  CLI or API error`,
		Example: `  semctl task stop 812
  semctl task stop 812 --force
  semctl task stop 812 --force --wait --timeout 60s`,
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
			projectID, _ := ctx.ResolveProjectID(cmd.Context())

			// A graceful stop only moves a queued task to "stopping"; it is
			// finalized when a runner reports back, which never happens for a
			// task no runner picked up. Warn so --wait does not look like a hang.
			if wait && !force {
				ctx.Printer.PrintWarning("graceful stop may never complete for a waiting/queued task; use --force to stop it immediately")
			}

			// The Semaphore stop endpoint reads a single {"force": bool} body and
			// always replies 204, so the force flag is the only thing that decides
			// whether a queued task actually stops.
			body := map[string]bool{"force": force}
			resp, err := ctx.Client.Do(cmd.Context(), "POST", fmt.Sprintf("/project/%d/tasks/%d/stop", projectID, taskID), body)
			if err != nil {
				return fmt.Errorf("stop task: %w", err)
			}
			if err := api.CheckResponse(resp); err != nil {
				return fmt.Errorf("stop task: %w", err)
			}

			// Without --wait we only know the request was accepted, not that the
			// task stopped — say exactly that. With --wait the confirmed result
			// is printed by waitForStop, so emit only a provisional line here.
			verb := "stop"
			if force {
				verb = "forced stop"
			}
			if !wait {
				ctx.Printer.PrintSuccess(fmt.Sprintf("Requested %s of task %d", verb, taskID))
				return nil
			}
			ctx.Printer.PrintInfo(fmt.Sprintf("Requested %s of task %d, waiting for it to stop…", verb, taskID))

			err = waitForStop(cmd.Context(), ctx, projectID, taskID, timeout, interval, exitCode)
			if err != nil && exitCode {
				_, _ = fmt.Fprintln(ctx.Printer.Stderr, err)
				os.Exit(taskExitCode(err))
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "force the task to stop immediately (required for queued/waiting tasks)")
	cmd.Flags().BoolVar(&wait, "wait", false, "poll until the task reaches a terminal state after the stop is requested")
	cmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "with --wait, give up after this duration (0 = wait indefinitely)")
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "with --wait, polling interval")
	cmd.Flags().BoolVar(&exitCode, "exit-code", false, "with --wait, return the task's final state as the process exit code")
	return cmd
}

// waitForStop polls a task until it leaves the active states (waiting, running,
// starting, stopping) and reports the terminal state it landed on. Unlike
// watchTask, reaching "stopped"/"canceled" here is the success case. Exit codes
// mirror watchTask so users learn a single scheme across the task subcommands.
func waitForStop(ctx context.Context, c *cli.Context, projectID, taskID int, timeout, interval time.Duration, returnExitCode bool) error {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		resp, err := c.Client.Do(ctx, "GET", fmt.Sprintf("/project/%d/tasks/%d", projectID, taskID), nil)
		if err != nil {
			// A timeout/cancel can land mid-request; surface it as a timeout
			// rather than a raw transport error.
			if ctx.Err() != nil {
				return stopTimeoutErr(taskID, returnExitCode, ctx.Err())
			}
			return fmt.Errorf("wait for stop: %w", err)
		}
		var task api.Task
		if err := api.DecodeJSON(resp, &task); err != nil {
			return fmt.Errorf("decode task: %w", err)
		}

		switch strings.ToLower(task.Status) {
		case "stopped", "canceled", "cancelled":
			if returnExitCode {
				return &exitCodeError{code: 2}
			}
			c.Printer.PrintSuccess(fmt.Sprintf("Task %d stopped", taskID))
			return nil
		case "success":
			if returnExitCode {
				return &exitCodeError{code: 0}
			}
			c.Printer.PrintWarning(fmt.Sprintf("Task %d finished successfully before the stop took effect", taskID))
			return nil
		case "error", "failed":
			if returnExitCode {
				return &exitCodeError{code: 1}
			}
			c.Printer.PrintWarning(fmt.Sprintf("Task %d failed before the stop took effect", taskID))
			return nil
		}

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return stopTimeoutErr(taskID, returnExitCode, ctx.Err())
		}
	}
}

func stopTimeoutErr(taskID int, returnExitCode bool, cause error) error {
	if returnExitCode {
		return &exitCodeError{code: 3, msg: fmt.Sprintf("timed out waiting for task %d to stop", taskID)}
	}
	return fmt.Errorf("wait for stop: timed out waiting for task %d to stop: %w", taskID, cause)
}
