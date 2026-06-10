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

package cli

import "github.com/spf13/cobra"

// RegisterGlobalFlags registers the persistent global flags that every command
// reads via BuildCmdContext. It is the single source of truth shared by the
// real root command (cmd/semctl) and the test harness (internal/testutil), so
// the two can never drift out of sync.
func RegisterGlobalFlags(cmd *cobra.Command) {
	f := cmd.PersistentFlags()
	f.String("host", "", "Semaphore UI host URL")
	f.StringP("project", "p", "", "Default project")
	f.StringP("output", "o", "", "Output format (table, json, yaml, text)")
	f.String("profile", "", "Configuration profile")
	f.Bool("json", false, "Output in JSON format")
	f.Bool("no-color", false, "Disable colored output")
	f.Bool("verbose", false, "Verbose output")
	f.Bool("debug", false, "Debug output")
	f.Bool("no-interactive", false, "Disable interactive prompts")
	f.Bool("rich-exit-codes", false, "Exit with the error class's exit code instead of 1 (also via SEMCTL_RICH_EXIT)")
}
