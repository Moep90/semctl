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
	"os"
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
		Long:  `List, inspect, create, update, and delete access keys within the active project.`,
		Example: `  semctl keystore list
  semctl keystore get deploy-key
  semctl keystore create --name deploy-key --type ssh --login git --private-key-file id_ed25519
  semctl keystore update deploy-key --login deploy
  semctl keystore delete deploy-key`,
	}
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newGetCommand())
	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newUpdateCommand())
	cmd.AddCommand(newDeleteCommand())
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

// keystoreSecretFlags registers the shared secret flags on a command.
func keystoreSecretFlags(cmd *cobra.Command) {
	cmd.Flags().String("name", "", "Key name")
	cmd.Flags().String("type", "none", "Key type (ssh, login_password, none)")
	cmd.Flags().String("login", "", "Login / username")
	cmd.Flags().String("password", "", "Password (login_password type)")
	cmd.Flags().String("private-key", "", "SSH private key contents (ssh type)")
	cmd.Flags().String("private-key-file", "", "Read SSH private key from a file (ssh type)")
}

// buildKeystoreSecret populates the nested secret block expected by Semaphore
// for the given key type. NOTE: the exact secret body shape must be validated
// against a live Semaphore instance.
func buildKeystoreSecret(cmd *cobra.Command, keyType string, body map[string]any) error {
	switch keyType {
	case "ssh":
		login, _ := cmd.Flags().GetString("login")
		privateKey, _ := cmd.Flags().GetString("private-key")
		if file, _ := cmd.Flags().GetString("private-key-file"); file != "" {
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("read private key file: %w", err)
			}
			privateKey = string(data)
		}
		body["ssh"] = map[string]any{
			"login":       login,
			"private_key": privateKey,
		}
	case "login_password":
		login, _ := cmd.Flags().GetString("login")
		password, _ := cmd.Flags().GetString("password")
		body["login_password"] = map[string]any{
			"login":    login,
			"password": password,
		}
	}
	return nil
}

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an access key",
		Long:  `Create a new access key in the active project.`,
		Example: `  semctl keystore create --name deploy-key --type ssh --login git --private-key-file id_ed25519
  semctl keystore create --name vault-pass --type login_password --login admin --password s3cret`,
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
			keyType, _ := cmd.Flags().GetString("type")
			body := map[string]any{
				"name":       name,
				"project_id": projectID,
				"type":       keyType,
			}
			if err := buildKeystoreSecret(cmd, keyType, body); err != nil {
				return err
			}
			_, err = ctx.Client.Do(cmd.Context(), "POST", fmt.Sprintf("/project/%d/keys", projectID), body)
			if err != nil {
				return fmt.Errorf("create keystore: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Created keystore %s\n", name)
			return nil
		},
	}
	keystoreSecretFlags(cmd)
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <KEYSTORE>",
		Short: "Update an access key",
		Long:  `Update an access key. Accepts a keystore ID or name. Only changed fields are sent.`,
		Example: `  semctl keystore update deploy-key --login deploy
  semctl keystore update 9 --type login_password --login admin --password s3cret`,
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
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			body := map[string]any{
				"id":         keystoreID,
				"project_id": projectID,
			}
			if cmd.Flags().Changed("name") {
				name, _ := cmd.Flags().GetString("name")
				body["name"] = name
			}
			keyType, _ := cmd.Flags().GetString("type")
			if cmd.Flags().Changed("type") {
				body["type"] = keyType
			}
			if cmd.Flags().Changed("login") || cmd.Flags().Changed("password") ||
				cmd.Flags().Changed("private-key") || cmd.Flags().Changed("private-key-file") {
				if err := buildKeystoreSecret(cmd, keyType, body); err != nil {
					return err
				}
			}
			_, err = ctx.Client.Do(cmd.Context(), "PUT", fmt.Sprintf("/project/%d/keys/%d", projectID, keystoreID), body)
			if err != nil {
				return fmt.Errorf("update keystore: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Updated keystore %s\n", args[0])
			return nil
		},
	}
	keystoreSecretFlags(cmd)
	return cmd
}

func newDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <KEYSTORE>",
		Short: "Delete an access key",
		Long:  `Delete an access key. Accepts a keystore ID or name.`,
		Example: `  semctl keystore delete deploy-key
  semctl keystore delete 9`,
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
			projectID, err := ctx.ResolveProjectID(cmd.Context())
			if err != nil {
				return err
			}
			_, err = ctx.Client.Do(cmd.Context(), "DELETE", fmt.Sprintf("/project/%d/keys/%d", projectID, keystoreID), nil)
			if err != nil {
				return fmt.Errorf("delete keystore: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Deleted keystore %s\n", args[0])
			return nil
		},
	}
}
