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

package semerr

import (
	"os"
	"strings"
	"testing"
)

func TestRenderMarkdownContainsAllClasses(t *testing.T) {
	md := RenderMarkdown()
	for _, c := range Classes() {
		if !strings.Contains(md, c.Code) {
			t.Errorf("rendered docs missing class %s", c.Code)
		}
		if !strings.Contains(md, c.Name) {
			t.Errorf("rendered docs missing name %s", c.Name)
		}
	}
	// A domain heading and the flagship row should be present.
	if !strings.Contains(md, "## API") && !strings.Contains(md, "api") {
		t.Fatalf("expected a domain section, got:\n%s", md[:200])
	}
	if !strings.Contains(md, "SEM500004") || !strings.Contains(md, "API resource not found") {
		t.Fatal("expected the flagship 404 class row")
	}
}

// TestDocsErrorsInSync fails if docs/errors.md has drifted from the registry.
// Regenerate with: go generate ./internal/semerr/...
func TestDocsErrorsInSync(t *testing.T) {
	const path = "../../docs/errors.md"
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run `go generate ./internal/semerr/...`)", path, err)
	}
	if string(onDisk) != RenderMarkdown() {
		t.Fatalf("%s is out of sync with the registry; run `go generate ./internal/semerr/...`", path)
	}
}
