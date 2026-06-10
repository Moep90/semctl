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

// Command gen-errors-doc writes docs/errors.md from the semerr registry.
// Invoked via `go generate ./internal/semerr/...`.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/moep90/semaphore-cli/internal/semerr"
)

func main() {
	root, err := moduleRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen-errors-doc:", err)
		os.Exit(1)
	}
	path := filepath.Join(root, "docs", "errors.md")
	if err := os.WriteFile(path, []byte(semerr.RenderMarkdown()), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "gen-errors-doc:", err)
		os.Exit(1)
	}
	fmt.Println("wrote", path)
}

// moduleRoot walks up from the cwd to the directory containing go.mod, so the
// generator works whether invoked from the repo root or via `go generate`
// (which runs in the package directory).
func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found above %q", dir)
		}
		dir = parent
	}
}
