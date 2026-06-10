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
	"errors"
	"regexp"
	"testing"
)

func TestNewKnownCode(t *testing.T) {
	e := New("SEM500004")
	if e.Code != "SEM500004" {
		t.Fatalf("code: %q", e.Code)
	}
	if e.Name != "API_RESOURCE_NOT_FOUND" {
		t.Fatalf("name: %q", e.Name)
	}
	if e.Title == "" {
		t.Fatal("title must not be empty")
	}
	// Message defaults to the class DefaultMessage until overridden.
	if e.Message != e.DefaultMessage || e.Message == "" {
		t.Fatalf("message should default to DefaultMessage, got %q", e.Message)
	}
	if e.Retryable {
		t.Fatal("404 must not be retryable")
	}
	if e.ExitCode != 44 {
		t.Fatalf("exit code: %d", e.ExitCode)
	}
}

func TestNewUnknownCodeFallsBack(t *testing.T) {
	e := New("SEM999999")
	if e.Code != "SEM000001" {
		t.Fatalf("unknown code must fall back to SEM000001, got %q", e.Code)
	}
}

func TestWithMessageAndWrapPreservesCause(t *testing.T) {
	cause := errors.New("boom")
	e := New("SEM500004").WithMessage("custom message").Wrap(cause)
	if e.Message != "custom message" {
		t.Fatalf("message: %q", e.Message)
	}
	if !errors.Is(e, cause) {
		t.Fatal("Wrap must preserve the cause for errors.Is/Unwrap")
	}
}

func TestWithDropsNonAllowlistedKey(t *testing.T) {
	e := New("SEM500004").
		With("method", "GET").
		With("authorization", "Bearer secret")
	if e.Metadata["method"] != "GET" {
		t.Fatalf("allowlisted key dropped: %v", e.Metadata)
	}
	if _, ok := e.Metadata["authorization"]; ok {
		t.Fatal("non-allowlisted metadata key must be dropped, not stored")
	}
}

func TestRegistryInvariants(t *testing.T) {
	re := regexp.MustCompile(`^SEM\d{6}$`)
	seen := make(map[string]bool)
	for _, c := range Classes() {
		if !re.MatchString(c.Code) {
			t.Fatalf("code %q does not match SEMDDDNNN", c.Code)
		}
		if seen[c.Code] {
			t.Fatalf("duplicate code %q", c.Code)
		}
		seen[c.Code] = true
		if c.Name == "" || c.Title == "" || c.DefaultMessage == "" || c.Severity == "" {
			t.Fatalf("class %q has empty required field: %+v", c.Code, c)
		}
		if c.ExitCode == 0 {
			t.Fatalf("class %q has zero exit code", c.Code)
		}
	}
	if len(seen) == 0 {
		t.Fatal("registry is empty")
	}
	// The flagship classes must exist.
	for _, code := range []string{"SEM000001", "SEM500004", "SEM500011"} {
		if !seen[code] {
			t.Fatalf("registry missing required class %q", code)
		}
	}
}
