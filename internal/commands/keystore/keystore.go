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

package keystore

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/cli"
)

// NewKeystoreCommand builds the keystore command group.
func NewKeystoreCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keystore",
		Short: "Manage keystore / access keys",
		Long:  `List and inspect access keys within the active project.`,
		Example: `  semctl keystore list
  semctl keystore get deploy-key`,
	}
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newGetCommand())
	return cmd
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List keystores",
		Long:  `Show all access keys in the active project.`,
		Example: `  semctl keystore list
  semctl keystore list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			resp, err := ctx.Client.Do(cmd.Context(), "GET", fmt.Sprintf("/project/%d/keys", projectID), nil)
			if err != nil {
				return fmt.Errorf("list keystore: %w", err)
			}
			var keystores []api.Keystore
			if err := api.DecodeJSON(resp, &keystores); err != nil {
				return fmt.Errorf("decode keystore: %w", err)
			}
			rows := make([][]string, len(keystores))
			for i, k := range keystores {
				rows[i] = []string{
					strconv.Itoa(k.ID),
					k.Name,
					k.Type,
				}
			}
			return ctx.Printer.PrintTable([]string{"ID", "NAME", "TYPE"}, rows)
		},
	}
}

func newGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <KEYSTORE>",
		Short: "Get keystore details",
		Long:  `Show full details for an access key. Accepts a keystore ID or name.`,
		Example: `  semctl keystore get deploy-key
  semctl keystore get 9 --output yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			keystoreID, err := ctx.ResolveKeystoreID(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			projectID, _ := ctx.ResolveProjectID(cmd.Context())
			resp, err := ctx.Client.Do(cmd.Context(), "GET", fmt.Sprintf("/project/%d/keys/%d", projectID, keystoreID), nil)
			if err != nil {
				return fmt.Errorf("get keystore: %w", err)
			}
			var keystore api.Keystore
			if err := api.DecodeJSON(resp, &keystore); err != nil {
				return fmt.Errorf("decode keystore: %w", err)
			}
			return ctx.Printer.Print(keystore)
		},
	}
}
