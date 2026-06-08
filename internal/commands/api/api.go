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

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	semapi "github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/cli"
)

// NewAPICommand builds the api escape hatch command.
func NewAPICommand() *cobra.Command {
	var fields []string
	var rawFields []string
	var headers []string
	var input string

	cmd := &cobra.Command{
		Use:   "api <METHOD> <PATH>",
		Short: "Make an authenticated API request",
		Long: `Make a raw authenticated request to the Semaphore UI API.

The path is relative to /api. Use --field for typed JSON values and --raw-field
for string values. Use --input to send a JSON body from a file or stdin.

This is the escape hatch for endpoints not yet covered by first-class commands.`,
		Example: `  semctl api GET /info
  semctl api GET /projects
  semctl api POST /project/1/tasks --field template_id=7
  semctl api GET /project/1/tasks/last
  echo '{"message":"deploy"}' | semctl api POST /project/1/tasks --input -`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, err := cli.BuildCmdContext(cmd)
			if err != nil {
				return err
			}

			method := strings.ToUpper(args[0])
			path := args[1]

			var body any
			if input != "" {
				var data []byte
				if input == "-" {
					data, err = io.ReadAll(os.Stdin)
				} else {
					data, err = os.ReadFile(input)
				}
				if err != nil {
					return fmt.Errorf("read input: %w", err)
				}
				if err := json.Unmarshal(data, &body); err != nil {
					return fmt.Errorf("parse input json: %w", err)
				}
			} else if len(fields) > 0 || len(rawFields) > 0 {
				m := make(map[string]any)
				for _, f := range rawFields {
					parts := strings.SplitN(f, "=", 2)
					if len(parts) != 2 {
						return fmt.Errorf("invalid raw field: %s", f)
					}
					m[parts[0]] = parts[1]
				}
				for _, f := range fields {
					parts := strings.SplitN(f, "=", 2)
					if len(parts) != 2 {
						return fmt.Errorf("invalid field: %s", f)
					}
					var v any
					if err := json.Unmarshal([]byte(parts[1]), &v); err != nil {
						v = parts[1]
					}
					m[parts[0]] = v
				}
				body = m
			}

			extraHeaders := make(http.Header)
			for _, h := range headers {
				parts := strings.SplitN(h, ":", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid header: %s", h)
				}
				extraHeaders.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
			}

			resp, err := ctx.Client.DoWithHeaders(cmd.Context(), method, path, body, extraHeaders)
			if err != nil {
				return fmt.Errorf("api request: %w", err)
			}
			defer func() { _ = resp.Body.Close() }()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("read response: %w", err)
			}

			if resp.StatusCode >= 400 {
				return &semapi.Error{StatusCode: resp.StatusCode, Body: data}
			}

			if len(data) == 0 {
				return nil
			}

			// Pretty-print JSON if possible.
			var pretty any
			if err := json.Unmarshal(data, &pretty); err == nil {
				enc := json.NewEncoder(ctx.Printer.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(pretty)
			}

			_, _ = ctx.Printer.Stdout.Write(data)
			_, _ = fmt.Fprintln(ctx.Printer.Stdout)
			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&fields, "field", "F", nil, "Typed request field")
	cmd.Flags().StringArrayVarP(&rawFields, "raw-field", "f", nil, "String request field")
	cmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "HTTP header")
	cmd.Flags().StringVar(&input, "input", "", "Read request body from file or stdin")

	return cmd
}
