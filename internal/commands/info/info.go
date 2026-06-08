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

package info

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/cli"
)

// NewInfoCommand builds the info command.
func NewInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show server information",
		Long:  `Display version and health information from the Semaphore UI server.`,
		Example: `  semctl info
  semctl info --host https://semaphore.example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			resp, err := ctx.Client.Do(cmd.Context(), "GET", "/info", nil)
			if err != nil {
				return fmt.Errorf("fetch info: %w", err)
			}
			var info api.Info
			if err := api.DecodeJSON(resp, &info); err != nil {
				return fmt.Errorf("decode info: %w", err)
			}
			return ctx.Printer.Print(info)
		},
	}
}
