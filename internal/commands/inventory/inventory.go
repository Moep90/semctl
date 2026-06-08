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

package inventory

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/cli"
)

// NewInventoryCommand builds the inventory command group.
func NewInventoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inventory",
		Short: "Manage inventories",
		Long:  `List and inspect inventories within the active project.`,
		Example: `  semctl inventory list
  semctl inventory get prod-hosts`,
	}
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newGetCommand())
	return cmd
}

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List inventories",
		Long:  `Show all inventories in the active project.`,
		Example: `  semctl inventory list
  semctl inventory list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			resp, err := ctx.Client.Do(cmd.Context(), "GET", fmt.Sprintf("/project/%d/inventory", projectID), nil)
			if err != nil {
				return fmt.Errorf("list inventory: %w", err)
			}
			var inventories []api.Inventory
			if err := api.DecodeJSON(resp, &inventories); err != nil {
				return fmt.Errorf("decode inventory: %w", err)
			}
			rows := make([][]string, len(inventories))
			for i, inv := range inventories {
				rows[i] = []string{
					strconv.Itoa(inv.ID),
					inv.Name,
					inv.Type,
				}
			}
			return ctx.Printer.PrintTable([]string{"ID", "NAME", "TYPE"}, rows)
		},
	}
}

func newGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <INVENTORY>",
		Short: "Get inventory details",
		Long:  `Show full details for an inventory. Accepts an inventory ID or name.`,
		Example: `  semctl inventory get prod-hosts
  semctl inventory get 3 --output yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			inventoryID, err := ctx.ResolveInventoryID(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			projectID, _ := ctx.ResolveProjectID(cmd.Context())
			resp, err := ctx.Client.Do(cmd.Context(), "GET", fmt.Sprintf("/project/%d/inventory/%d", projectID, inventoryID), nil)
			if err != nil {
				return fmt.Errorf("get inventory: %w", err)
			}
			var inventory api.Inventory
			if err := api.DecodeJSON(resp, &inventory); err != nil {
				return fmt.Errorf("decode inventory: %w", err)
			}
			return ctx.Printer.Print(inventory)
		},
	}
}
