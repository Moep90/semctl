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
	"fmt"
	"io"
	"sort"
)

// Payload returns the structured representation emitted under --output
// json/yaml. Empty optional fields are omitted.
func (e *SemError) Payload() map[string]any {
	p := map[string]any{
		"code":      e.Code,
		"name":      e.Name,
		"title":     e.Title,
		"message":   e.Message,
		"domain":    e.Domain,
		"retryable": e.Retryable,
		"exit_code": e.ExitCode,
	}
	if e.Hint != "" {
		p["hint"] = e.Hint
	}
	if e.HTTPStatus > 0 {
		p["http_status"] = e.HTTPStatus
	}
	if len(e.Metadata) > 0 {
		p["metadata"] = e.Metadata
	}
	return p
}

// WriteHuman renders the error for a terminal. The default form shows code,
// title, message, an optional hint, and allowlisted details (method/path/
// status). Verbose adds request_id and retryable.
func (e *SemError) WriteHuman(w io.Writer, verbose bool) {
	fmt.Fprintf(w, "error %s: %s\n", e.Code, e.Title)
	if e.Message != "" {
		fmt.Fprintf(w, "\n%s\n", e.Message)
	}
	if e.Hint != "" {
		fmt.Fprintf(w, "\nHint:\n  %s\n", e.Hint)
	}

	// Details: a stable subset first, then any remaining allowlisted metadata.
	type kv struct{ k, v string }
	var details []kv
	for _, k := range []string{"method", "path", "status"} {
		if v, ok := e.Metadata[k]; ok {
			details = append(details, kv{k, v})
		}
	}
	if verbose {
		if v, ok := e.Metadata["request_id"]; ok {
			details = append(details, kv{"request_id", v})
		}
		details = append(details, kv{"retryable", fmt.Sprintf("%t", e.Retryable)})
	}
	// Any other allowlisted metadata keys not already shown.
	shown := map[string]bool{"method": true, "path": true, "status": true, "request_id": true}
	var extra []string
	for k := range e.Metadata {
		if !shown[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	for _, k := range extra {
		details = append(details, kv{k, e.Metadata[k]})
	}

	if len(details) > 0 {
		fmt.Fprintf(w, "\nDetails:\n")
		for _, d := range details {
			fmt.Fprintf(w, "  %s: %s\n", d.k, d.v)
		}
	}
}
