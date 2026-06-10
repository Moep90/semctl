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
	"bytes"
	"strings"
	"testing"

	"github.com/moep90/semaphore-cli/internal/api"
)

func TestPayloadShape(t *testing.T) {
	se := Classify(&api.Error{StatusCode: 404, Method: "GET", Path: "/project/1/tasks/last", RequestID: "req_abc"})
	p := se.Payload()

	if p["code"] != "SEM500004" {
		t.Fatalf("code: %v", p["code"])
	}
	if p["name"] != "API_RESOURCE_NOT_FOUND" {
		t.Fatalf("name: %v", p["name"])
	}
	if p["retryable"] != false {
		t.Fatalf("retryable: %v", p["retryable"])
	}
	if p["http_status"] != 404 {
		t.Fatalf("http_status: %v (%T)", p["http_status"], p["http_status"])
	}
	if p["exit_code"] != 44 {
		t.Fatalf("exit_code: %v", p["exit_code"])
	}
	md, ok := p["metadata"].(map[string]string)
	if !ok || md["method"] != "GET" || md["path"] != "/project/1/tasks/last" {
		t.Fatalf("metadata: %v", p["metadata"])
	}
	if _, ok := p["title"]; !ok {
		t.Fatal("payload missing title")
	}
}

func TestPayloadOmitsEmptyHTTPStatus(t *testing.T) {
	p := New("SEM000001").Payload()
	if _, ok := p["http_status"]; ok {
		t.Fatal("http_status must be omitted when there is no HTTP status")
	}
}

func TestWriteHumanDefault(t *testing.T) {
	se := Classify(&api.Error{StatusCode: 404, Method: "GET", Path: "/project/1/tasks/last"})
	var buf bytes.Buffer
	se.WriteHuman(&buf, false)
	out := buf.String()
	for _, want := range []string{"SEM500004", "API resource not found", "HTTP 404", "Hint:", "method: GET"} {
		if !strings.Contains(out, want) {
			t.Fatalf("human output missing %q, got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "request_id") {
		t.Fatalf("default output should not include request_id, got:\n%s", out)
	}
}

func TestWriteHumanVerboseAddsRequestID(t *testing.T) {
	se := Classify(&api.Error{StatusCode: 404, Method: "GET", Path: "/x", RequestID: "req_abc"})
	var buf bytes.Buffer
	se.WriteHuman(&buf, true)
	out := buf.String()
	if !strings.Contains(out, "request_id: req_abc") {
		t.Fatalf("verbose output should include request_id, got:\n%s", out)
	}
	if !strings.Contains(out, "retryable:") {
		t.Fatalf("verbose output should include retryable, got:\n%s", out)
	}
}
