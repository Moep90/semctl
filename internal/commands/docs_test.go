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

package commands

import (
	"strings"
	"testing"
)

func TestRenderMarkdownDocs(t *testing.T) {
	doc := RenderMarkdownDocs(NewRootCommand("test"))

	checks := []struct {
		desc string
		want string
	}{
		{"top-level title", "# semctl"},
		{"a global flags section", "Global flags"},
		{"a global flag is documented", "--profile"},
		{"a leaf command heading", "## semctl task run"},
		{"the leaf command's examples", "semctl task run deploy-prod"},
		{"the leaf command's local flag", "--extra-vars"},
		{"a sibling top-level command", "## semctl project use"},
	}
	for _, c := range checks {
		if !strings.Contains(doc, c.want) {
			t.Errorf("expected doc to contain %s (%q)\n--- doc ---\n%s", c.desc, c.want, doc)
		}
	}
}

func TestRenderMarkdownDocsSkipsHiddenAndHelpCommands(t *testing.T) {
	doc := RenderMarkdownDocs(NewRootCommand("test"))
	// The auto-generated "help" and "completion" helper commands are noise in a
	// reference; they must not get their own sections.
	if strings.Contains(doc, "## semctl help") {
		t.Errorf("doc must not document the auto-generated help command:\n%s", doc)
	}
}
