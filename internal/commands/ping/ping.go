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

package ping

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/cli"
	"github.com/moep90/semaphore-cli/internal/output"
)

// NewPingCommand builds the ping command.
func NewPingCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ping",
		Short: "Check connectivity to Semaphore UI",
		Long:  `Send a lightweight request to verify that the configured host is reachable.`,
		Example: `  semctl ping
  semctl ping --host https://semaphore.example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				// ping may work without auth; try with host only.
				return fmt.Errorf("no host configured; use --host or set SEMAPHORE_HOST")
			}
			resp, err := ctx.Client.Do(cmd.Context(), "GET", "/ping", nil)
			if err != nil {
				return fmt.Errorf("ping failed: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode == http.StatusOK {
				if ctx.Printer.Mode == output.ModeJSON || ctx.Printer.Mode == output.ModeYAML {
					return ctx.Printer.Print(map[string]string{"message": "Semaphore UI is reachable"})
				}
				_, _ = fmt.Fprintln(ctx.Printer.Stdout, "✓ Semaphore UI is reachable")
				return nil
			}
			return fmt.Errorf("ping returned status %d", resp.StatusCode)
		},
	}
}
