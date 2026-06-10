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
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"syscall"
	"testing"
)

// timeoutError is a minimal net.Error reporting a timeout.
type timeoutError struct{}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return false }

func TestClassifyTransportErrors(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		wantCode string
	}{
		{"dns", &url.Error{Op: "Get", URL: "https://h/api/x", Err: &net.DNSError{Err: "no such host", IsNotFound: true}}, "SEM400001"},
		{"timeout", &url.Error{Op: "Get", URL: "https://h/api/x", Err: timeoutError{}}, "SEM400002"},
		{"deadline", fmt.Errorf("do: %w", context.DeadlineExceeded), "SEM400002"},
		{"refused", &url.Error{Op: "Get", URL: "https://h/api/x", Err: &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}}, "SEM400005"},
		{"reset", &url.Error{Op: "Get", URL: "https://h/api/x", Err: &net.OpError{Op: "read", Err: syscall.ECONNRESET}}, "SEM400006"},
		{"tls", &url.Error{Op: "Get", URL: "https://h/api/x", Err: x509.UnknownAuthorityError{}}, "SEM400003"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			se := Classify(tc.err)
			if se.Code != tc.wantCode {
				t.Fatalf("code = %q, want %q (err: %v)", se.Code, tc.wantCode, tc.err)
			}
			if se.Domain != "network" {
				t.Fatalf("domain = %q, want network", se.Domain)
			}
		})
	}
}

func TestTransportMessageDoesNotLeakURL(t *testing.T) {
	// A url.Error stringifies to include the full URL (with any query string).
	// The user-facing message must not carry it — only the class default.
	err := &url.Error{Op: "Get", URL: "https://h/api/x?token=SECRET", Err: timeoutError{}}
	se := Classify(err)
	if strings.Contains(se.Message, "SECRET") || strings.Contains(se.Message, "token=") {
		t.Fatalf("transport message must not leak the URL/query, got %q", se.Message)
	}
}

func TestClassifySerializationErrors(t *testing.T) {
	syntax := func() error { return json.Unmarshal([]byte("{bad"), &struct{}{}) }()
	typeErr := func() error {
		return json.Unmarshal([]byte(`{"n":"x"}`), &struct {
			N int `json:"n"`
		}{})
	}()

	cases := []struct {
		name     string
		err      error
		wantCode string
	}{
		{"syntax", syntax, "SEM600001"},
		{"type-mismatch", typeErr, "SEM600002"},
		{"empty", fmt.Errorf("decode: %w", io.EOF), "SEM600004"},
		{"truncated", fmt.Errorf("decode: %w", io.ErrUnexpectedEOF), "SEM600001"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			se := Classify(tc.err)
			if se.Code != tc.wantCode {
				t.Fatalf("code = %q, want %q (err: %v)", se.Code, tc.wantCode, tc.err)
			}
			if se.Domain != "serialization" {
				t.Fatalf("domain = %q, want serialization", se.Domain)
			}
		})
	}
}
