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

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/cli"
)

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
