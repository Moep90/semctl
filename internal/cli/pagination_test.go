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
	"testing"

	"github.com/spf13/cobra"
)

func TestPageHint(t *testing.T) {
	cases := []struct {
		name        string
		total       int
		limit, page string
		want        string
	}{
		{"no limit", 10, "", "", ""},
		{"first page", 4, "2", "", "page 1 of 2"},
		{"last page", 6, "3", "2", "page 2 of 2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			AddPaginationFlags(cmd)
			if tc.limit != "" {
				_ = cmd.Flags().Set("limit", tc.limit)
			}
			if tc.page != "" {
				_ = cmd.Flags().Set("page", tc.page)
			}
			if got := PageHint(tc.total, cmd); got != tc.want {
				t.Fatalf("PageHint(%d) = %q, want %q", tc.total, got, tc.want)
			}
		})
	}
}
