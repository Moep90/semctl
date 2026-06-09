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
	"strconv"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/cli"
)

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		Long:  `Show tasks in the active project.`,
		Example: `  semctl task list
  semctl task list --json
  semctl task list --limit 20 --page 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/project/%d/tasks", projectID) + cli.PaginationQuery(cmd)
			resp, err := ctx.Client.Do(cmd.Context(), "GET", path, nil)
			if err != nil {
				return fmt.Errorf("list tasks: %w", err)
			}
			var tasks []api.Task
			if err := api.DecodeJSON(resp, &tasks); err != nil {
				return fmt.Errorf("decode tasks: %w", err)
			}
			tasks = cli.Paginate(tasks, cmd)
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
			return ctx.Printer.PrintList([]string{"ID", "TEMPLATE", "STATUS", "MESSAGE", "CREATED"}, rows, tasks)
		},
	}
	cli.AddPaginationFlags(cmd)
	return cmd
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
