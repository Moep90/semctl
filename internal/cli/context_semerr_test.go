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
	"errors"
	"testing"

	"github.com/moep90/semaphore-cli/internal/config"
	"github.com/moep90/semaphore-cli/internal/semerr"
)

func TestBuildContextNoHostReturnsConfigClass(t *testing.T) {
	t.Setenv("SEMAPHORE_HOST", "")
	t.Setenv("SEMAPHORE_PROFILE", "")
	t.Setenv("SEMAPHORE_PROJECT", "")

	cfg := config.DefaultConfig()
	_, err := BuildContext(cfg, "", "", "", "", false, false, false)
	if err == nil {
		t.Fatal("expected an error when no host is configured")
	}
	var se *semerr.SemError
	if !errors.As(err, &se) {
		t.Fatalf("expected a *semerr.SemError, got %T: %v", err, err)
	}
	if se.Code != "SEM200001" {
		t.Fatalf("code = %q, want SEM200001", se.Code)
	}
}
