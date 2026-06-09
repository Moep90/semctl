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
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/cli"
	"github.com/moep90/semaphore-cli/internal/output"
)

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
				for _, o := range outputs {
					key := o.Time + "|" + o.Output
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
