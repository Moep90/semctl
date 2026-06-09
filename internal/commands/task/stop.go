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
until the task actually reaches a terminal state, optionally bounded by
--timeout.`,
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
			if force {
				ctx.Printer.PrintSuccess(fmt.Sprintf("Stopped task %d (forced)", taskID))
			} else {
				ctx.Printer.PrintSuccess(fmt.Sprintf("Requested stop of task %d", taskID))
			}
			if wait {
				return waitForStop(cmd.Context(), ctx, projectID, taskID, timeout, interval)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "force the task to stop immediately (required for queued/waiting tasks)")
	cmd.Flags().BoolVar(&wait, "wait", false, "poll until the task reaches a terminal state after the stop is requested")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "with --wait, give up after this duration (0 = wait indefinitely)")
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "with --wait, polling interval")
	return cmd
}

// waitForStop polls a task until it leaves the active states (waiting, running,
// starting, stopping) and reports the terminal state it landed on. Unlike
// watchTask, reaching "stopped"/"canceled" here is the success case.
func waitForStop(ctx context.Context, c *cli.Context, projectID, taskID int, timeout, interval time.Duration) error {
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
				return fmt.Errorf("wait for stop: timed out waiting for task %d to stop: %w", taskID, ctx.Err())
			}
			return fmt.Errorf("wait for stop: %w", err)
		}
		var task api.Task
		if err := api.DecodeJSON(resp, &task); err != nil {
			return fmt.Errorf("decode task: %w", err)
		}

		switch strings.ToLower(task.Status) {
		case "stopped", "canceled", "cancelled":
			c.Printer.PrintSuccess(fmt.Sprintf("Task %d stopped", taskID))
			return nil
		case "success", "error", "failed":
			// The task finished on its own before the stop took effect.
			c.Printer.PrintWarning(fmt.Sprintf("Task %d reached %q before the stop took effect", taskID, task.Status))
			return nil
		}

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return fmt.Errorf("wait for stop: timed out waiting for task %d to stop: %w", taskID, ctx.Err())
		}
	}
}
