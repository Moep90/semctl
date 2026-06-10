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

//go:generate go run ../../cmd/gen-errors-doc

import (
	"fmt"
	"sort"
	"strings"
)

// domainOrder fixes the section order in the generated reference.
var domainOrder = []struct{ key, heading string }{
	{"generic", "Generic"},
	{"cli", "CLI usage"},
	{"config", "Config"},
	{"auth", "Auth"},
	{"network", "Network / transport"},
	{"api", "API response"},
	{"serialization", "Serialization"},
	{"local", "Local runtime"},
}

// RenderMarkdown renders the full error-class reference from the registry. It
// is the single source of truth for docs/errors.md (kept in sync by a test).
func RenderMarkdown() string {
	byDomain := make(map[string][]Class)
	for _, c := range Classes() {
		byDomain[c.Domain] = append(byDomain[c.Domain], c)
	}

	var b strings.Builder
	b.WriteString("# Error classes\n\n")
	b.WriteString("`semctl` errors use stable error codes in the format `SEMDDDNNN`: ")
	b.WriteString("`SEM` product prefix, `DDD` domain, `NNN` ordinal within the domain. ")
	b.WriteString("The HTTP status (when any) is carried as metadata, not encoded in the code.\n\n")
	b.WriteString("In `--output json`/`yaml` the `error` field is a structured object ")
	b.WriteString("(`code`, `name`, `title`, `message`, `hint`, `retryable`, `exit_code`, `http_status`, `metadata`). ")
	b.WriteString("The process currently exits `1` for all failures; the per-class `exit_code` is informational.\n\n")
	b.WriteString("> This file is generated from the registry. Do not edit by hand; run `go generate ./internal/semerr/...`.\n")

	for _, d := range domainOrder {
		classes := byDomain[d.key]
		if len(classes) == 0 {
			continue
		}
		sort.Slice(classes, func(i, j int) bool { return classes[i].Code < classes[j].Code })
		fmt.Fprintf(&b, "\n## %s\n\n", d.heading)
		b.WriteString("| Code | Name | Meaning | Retryable | Exit |\n")
		b.WriteString("|---|---|---|---:|---:|\n")
		for _, c := range classes {
			retry := "No"
			if c.Retryable {
				retry = "Yes"
			}
			meaning := c.Title
			if c.Hint != "" {
				meaning = c.Title + ". " + c.Hint
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %d |\n", c.Code, c.Name, meaning, retry, c.ExitCode)
		}
	}
	return b.String()
}
