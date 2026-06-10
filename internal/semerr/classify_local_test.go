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
	"io/fs"
	"testing"
)

func TestClassifyLocalFileErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode string
	}{
		{"not-found", fmt.Errorf("read inventory file: %w", fs.ErrNotExist), "SEM700001"},
		{"permission", fmt.Errorf("read key file: %w", fs.ErrPermission), "SEM700002"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			se := Classify(tc.err)
			if se.Code != tc.wantCode {
				t.Fatalf("code = %q, want %q", se.Code, tc.wantCode)
			}
			if se.Domain != "local" {
				t.Fatalf("domain = %q, want local", se.Domain)
			}
		})
	}
}
