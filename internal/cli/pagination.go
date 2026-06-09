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

import (
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

// AddPaginationFlags registers the --limit and --page pagination flags shared by
// the list commands.
func AddPaginationFlags(cmd *cobra.Command) {
	cmd.Flags().Int("limit", 0, "Maximum number of items to return")
	cmd.Flags().Int("page", 0, "Page number to retrieve (1-based)")
}

// PaginationQuery builds the "?count=<limit>&page=<page>" query string from the
// --limit and --page flags, including only the flags that were explicitly set.
// It returns an empty string when neither flag is set, preserving the
// unpaginated request behavior. The params are forwarded for servers that honor
// them; the CLI also enforces the limit client-side via Paginate.
func PaginationQuery(cmd *cobra.Command) string {
	q := url.Values{}
	if cmd.Flags().Changed("limit") {
		limit, _ := cmd.Flags().GetInt("limit")
		q.Set("count", strconv.Itoa(limit))
	}
	if cmd.Flags().Changed("page") {
		page, _ := cmd.Flags().GetInt("page")
		q.Set("page", strconv.Itoa(page))
	}
	if len(q) == 0 {
		return ""
	}
	return "?" + q.Encode()
}

// Paginate applies the --limit and --page flags to an already-fetched slice.
// Semaphore's list endpoints do not honor count/page query params, so the CLI
// also enforces them client-side: --limit N caps the result at N items and
// --page offsets by (page-1)*N. Without --limit the slice is returned unchanged
// (a bare --page is a no-op, since there is no page size to offset by).
func Paginate[T any](items []T, cmd *cobra.Command) []T {
	limit := 0
	if cmd.Flags().Changed("limit") {
		limit, _ = cmd.Flags().GetInt("limit")
	}
	if limit <= 0 {
		return items
	}
	page := 1
	if cmd.Flags().Changed("page") {
		if p, _ := cmd.Flags().GetInt("page"); p > 1 {
			page = p
		}
	}
	start := (page - 1) * limit
	if start >= len(items) {
		return items[:0]
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}
