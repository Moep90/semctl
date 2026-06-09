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
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/cli"
)

// NewInventoryCommand builds the inventory command group.
func NewInventoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inventory",
		Short: "Manage inventories",
		Long:  `List, inspect, create, update, and delete inventories within the active project.`,
		Example: `  semctl inventory list
  semctl inventory get prod-hosts
  semctl inventory create --name prod-hosts --inventory-file hosts.ini
  semctl inventory update prod-hosts --type file
  semctl inventory delete prod-hosts`,
	}
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newUpdateCommand())
	cmd.AddCommand(newDeleteCommand())
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
			path := fmt.Sprintf("/project/%d/inventory", projectID) + cli.PaginationQuery(cmd)
			resp, err := ctx.Client.Do(cmd.Context(), "GET", path, nil)
			if err != nil {
				return fmt.Errorf("list inventory: %w", err)
			}
			var inventories []api.Inventory
			if err := api.DecodeJSON(resp, &inventories); err != nil {
				return fmt.Errorf("decode inventory: %w", err)
			}
			inventories = cli.Paginate(inventories, cmd)
			rows := make([][]string, len(inventories))
			for i, inv := range inventories {
				rows[i] = []string{
					strconv.Itoa(inv.ID),
					inv.Name,
					inv.Type,
				}
			}
			return ctx.Printer.PrintList([]string{"ID", "NAME", "TYPE"}, rows, inventories)
		},
	}
	cli.AddPaginationFlags(cmd)
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

// inlineEscapes interprets common backslash escape sequences in an inline
// --inventory value so that, e.g., `[web]\nhost1` becomes a two-line INI body.
// `\\` is handled first so a literal backslash can be preserved. Content read
// from --inventory-file already contains real newlines and is left untouched.
var inlineEscapes = strings.NewReplacer(`\\`, "\\", `\n`, "\n", `\t`, "\t", `\r`, "\r")

// addKeyFlags registers the --ssh-key-id and --become-key-id flags shared by
// create and update. They are strings so that "null" can clear the association
// (e.g. become_key_id must be null for NOPASSWD hosts).
func addKeyFlags(cmd *cobra.Command) {
	cmd.Flags().String("ssh-key-id", "", "SSH key (keystore) ID; 'null' to unset")
	cmd.Flags().String("become-key-id", "", "Become key (login_password keystore) ID; 'null' to unset")
}

// applyKeyFlags maps any set --ssh-key-id/--become-key-id flags into body. A
// value of "" or "null" sends a JSON null; otherwise the value must parse as an
// integer. Flags that were not set are left out so update preserves them.
func applyKeyFlags(cmd *cobra.Command, body map[string]any) error {
	for flag, field := range map[string]string{"ssh-key-id": "ssh_key_id", "become-key-id": "become_key_id"} {
		if !cmd.Flags().Changed(flag) {
			continue
		}
		raw, _ := cmd.Flags().GetString(flag)
		if raw == "" || strings.EqualFold(raw, "null") {
			body[field] = nil
			continue
		}
		id, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("--%s must be an integer or 'null'", flag)
		}
		body[field] = id
	}
	return nil
}

// readInventoryContent returns the inventory content from --inventory-file (if
// set) or --inventory.
func readInventoryContent(cmd *cobra.Command) (string, bool, error) {
	if file, _ := cmd.Flags().GetString("inventory-file"); file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", false, fmt.Errorf("read inventory file: %w", err)
		}
		return string(data), true, nil
	}
	if cmd.Flags().Changed("inventory") {
		content, _ := cmd.Flags().GetString("inventory")
		return inlineEscapes.Replace(content), true, nil
	}
	return "", false, nil
}

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an inventory",
		Long:  `Create a new inventory in the active project.`,
		Example: `  semctl inventory create --name prod-hosts --inventory-file hosts.ini
  semctl inventory create --name dynamic --type file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			name, _ := cmd.Flags().GetString("name")
			invType, _ := cmd.Flags().GetString("type")
			content, hasContent, err := readInventoryContent(cmd)
			if err != nil {
				return err
			}
			body := map[string]any{
				"name":       name,
				"project_id": projectID,
				"type":       invType,
			}
			if hasContent {
				body["inventory"] = content
			}
			if err := applyKeyFlags(cmd, body); err != nil {
				return err
			}
			resp, err := ctx.Client.Do(cmd.Context(), "POST", fmt.Sprintf("/project/%d/inventory", projectID), body)
			if err != nil {
				return fmt.Errorf("create inventory: %w", err)
			}
			if err := api.CheckResponse(resp); err != nil {
				return fmt.Errorf("create inventory: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Created inventory %s\n", name)
			return nil
		},
	}
	cmd.Flags().String("name", "", "Inventory name")
	cmd.Flags().String("type", "static", "Inventory type (static, file, etc.)")
	cmd.Flags().String("inventory", "", "Inventory content (\\n, \\t, \\r escapes are interpreted)")
	cmd.Flags().String("inventory-file", "", "Read inventory content from a file")
	addKeyFlags(cmd)
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <INVENTORY>",
		Short: "Update an inventory",
		Long:  `Update an inventory. Accepts an inventory ID or name. Only changed fields are sent.`,
		Example: `  semctl inventory update prod-hosts --type file
  semctl inventory update 3 --inventory-file hosts.ini`,
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
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			body := map[string]any{
				"id":         inventoryID,
				"project_id": projectID,
			}
			if cmd.Flags().Changed("name") {
				name, _ := cmd.Flags().GetString("name")
				body["name"] = name
			}
			if cmd.Flags().Changed("type") {
				invType, _ := cmd.Flags().GetString("type")
				body["type"] = invType
			}
			content, hasContent, err := readInventoryContent(cmd)
			if err != nil {
				return err
			}
			if hasContent {
				body["inventory"] = content
			}
			if err := applyKeyFlags(cmd, body); err != nil {
				return err
			}
			resp, err := ctx.Client.Do(cmd.Context(), "PUT", fmt.Sprintf("/project/%d/inventory/%d", projectID, inventoryID), body)
			if err != nil {
				return fmt.Errorf("update inventory: %w", err)
			}
			if err := api.CheckResponse(resp); err != nil {
				return fmt.Errorf("update inventory: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Updated inventory %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().String("name", "", "Inventory name")
	cmd.Flags().String("type", "static", "Inventory type (static, file, etc.)")
	cmd.Flags().String("inventory", "", "Inventory content (\\n, \\t, \\r escapes are interpreted)")
	cmd.Flags().String("inventory-file", "", "Read inventory content from a file")
	addKeyFlags(cmd)
	return cmd
}

func newDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <INVENTORY>",
		Short: "Delete an inventory",
		Long:  `Delete an inventory. Accepts an inventory ID or name.`,
		Example: `  semctl inventory delete prod-hosts
  semctl inventory delete 3`,
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
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			resp, err := ctx.Client.Do(cmd.Context(), "DELETE", fmt.Sprintf("/project/%d/inventory/%d", projectID, inventoryID), nil)
			if err != nil {
				return fmt.Errorf("delete inventory: %w", err)
			}
			if err := api.CheckResponse(resp); err != nil {
				return fmt.Errorf("delete inventory: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Deleted inventory %s\n", args[0])
			return nil
		},
	}
}
