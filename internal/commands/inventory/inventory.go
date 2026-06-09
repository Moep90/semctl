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
	"net/url"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/cli"
)

// paginationQuery builds the "?count=<limit>&page=<page>" query string from the
// --limit and --page flags, including only the flags that were explicitly set.
// It returns an empty string when neither flag is set, preserving the
// unpaginated request behavior.
func paginationQuery(cmd *cobra.Command) string {
	q := url.Values{}
	if cmd.Flags().Changed("limit") {
		limit, _ := cmd.Flags().GetInt("limit")
		q.Set("count", strconv.Itoa(limit))
	}
	if cmd.Flags().Changed("page") {
		page, _ := cmd.Flags().GetInt("page")
		q.Set("page", strconv.Itoa(page))
	}
	if len(q) == 0 {
		return ""
	}
	return "?" + q.Encode()
}

// addPaginationFlags registers the --limit and --page pagination flags.
func addPaginationFlags(cmd *cobra.Command) {
	cmd.Flags().Int("limit", 0, "Maximum number of items to return per page")
	cmd.Flags().Int("page", 0, "Page number to retrieve (1-based)")
}

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
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List inventories",
		Long:  `Show all inventories in the active project.`,
		Example: `  semctl inventory list
  semctl inventory list --json
  semctl inventory list --limit 20 --page 2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			path := fmt.Sprintf("/project/%d/inventory", projectID) + paginationQuery(cmd)
			resp, err := ctx.Client.Do(cmd.Context(), "GET", path, nil)
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
	addPaginationFlags(cmd)
	return cmd
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
