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

// Package task implements the `semctl task` command group: listing, inspecting,
// running, stopping, watching, and streaming logs for Semaphore UI tasks. Each
// subcommand lives in its own file (list.go, run.go, stop.go, logs.go,
// watch.go); this file only wires them into the group.
package task

import (
	"github.com/spf13/cobra"
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
