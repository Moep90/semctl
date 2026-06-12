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

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/moep90/semaphore-cli/internal/commands"
	"github.com/moep90/semaphore-cli/internal/semerr"
)

var version = "dev"

func main() {
	root := newRootCommand()
	if err := root.Execute(); err != nil {
		code, _ := formatError(root, err, os.Stderr)
		os.Exit(resolveExitCode(root, code))
	}
}

// newRootCommand builds the fully wired root command. It delegates to
// internal/commands so the binary and the docs generator share one definition.
func newRootCommand() *cobra.Command {
	return commands.NewRootCommand(version)
}

// resolveExitCode decides the process exit code. By default every failure exits
// 1 (backward compatible). When --rich-exit-codes or SEMCTL_RICH_EXIT is set,
// the per-class exit code is used instead, so automation can branch on it.
func resolveExitCode(cmd *cobra.Command, classCode int) int {
	rich, _ := cmd.PersistentFlags().GetBool("rich-exit-codes")
	if !rich && os.Getenv("SEMCTL_RICH_EXIT") == "" {
		return 1
	}
	if classCode <= 0 {
		return 1
	}
	return classCode
}

// formatError renders a top-level error as a structured semerr class, honoring
// the --json / --output / --verbose flags. It reads the flags off the command
// so it stays in sync with how the rest of the CLI resolves output mode.
//
// The returned int is the error class's exit code. The process currently always
// exits 1 (see main) for backward compatibility; surfacing the class exit code
// here lets a future opt-in or major release activate richer exit codes in one
// place without touching call sites.
func formatError(cmd *cobra.Command, err error, w io.Writer) (int, error) {
	se := semerr.Classify(err)
	if se == nil {
		return 0, nil
	}
	jsonFlag, _ := cmd.PersistentFlags().GetBool("json")
	outputFlag, _ := cmd.PersistentFlags().GetString("output")
	verbose, _ := cmd.PersistentFlags().GetBool("verbose")
	debug, _ := cmd.PersistentFlags().GetBool("debug")
	switch {
	case jsonFlag || outputFlag == "json":
		return se.ExitCode, json.NewEncoder(w).Encode(map[string]any{"error": se.Payload()})
	case outputFlag == "yaml":
		return se.ExitCode, yaml.NewEncoder(w).Encode(map[string]any{"error": se.Payload()})
	default:
		se.WriteHuman(w, verbose)
		// --debug surfaces the underlying cause (which may carry the response
		// body) that the default message deliberately omits.
		if debug {
			if cause := se.Cause(); cause != nil {
				fmt.Fprintf(w, "\nCause (debug):\n  %v\n", cause)
			}
		}
		return se.ExitCode, nil
	}
}
