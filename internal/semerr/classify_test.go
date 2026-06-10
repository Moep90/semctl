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
	"fmt"
	"strings"
	"testing"

	"github.com/moep90/semaphore-cli/internal/api"
)

func TestClassifyHTTPStatuses(t *testing.T) {
	cases := []struct {
		status    int
		wantCode  string
		retryable bool
	}{
		{400, "SEM500001", false},
		{401, "SEM500002", false},
		{403, "SEM500003", false},
		{404, "SEM500004", false},
		{409, "SEM500005", false},
		{429, "SEM500006", true},
		{500, "SEM500007", true},
		{502, "SEM500008", true},
		{503, "SEM500009", true},
		{504, "SEM500010", true},
		{418, "SEM500011", false}, // unmapped status
	}
	for _, tc := range cases {
		apiErr := &api.Error{StatusCode: tc.status, Method: "GET", Path: "/project/1/tasks/last"}
		se := Classify(apiErr)
		if se.Code != tc.wantCode {
			t.Errorf("status %d: code = %q, want %q", tc.status, se.Code, tc.wantCode)
		}
		if se.Retryable != tc.retryable {
			t.Errorf("status %d: retryable = %v, want %v", tc.status, se.Retryable, tc.retryable)
		}
		if se.HTTPStatus != tc.status {
			t.Errorf("status %d: HTTPStatus = %d", tc.status, se.HTTPStatus)
		}
		if se.Metadata["method"] != "GET" || se.Metadata["path"] != "/project/1/tasks/last" {
			t.Errorf("status %d: metadata = %v", tc.status, se.Metadata)
		}
		if !strings.Contains(se.Message, "GET") || !strings.Contains(se.Message, fmt.Sprintf("%d", tc.status)) {
			t.Errorf("status %d: message should mention method and status, got %q", tc.status, se.Message)
		}
	}
}

func TestClassifyWrappedAPIError(t *testing.T) {
	wrapped := fmt.Errorf("stop task: %w", &api.Error{StatusCode: 404, Method: "POST", Path: "/project/1/tasks/9/stop"})
	se := Classify(wrapped)
	if se.Code != "SEM500004" {
		t.Fatalf("wrapped api error not classified: %q", se.Code)
	}
}

func TestClassifyAPIErrorBodyNotInMessage(t *testing.T) {
	apiErr := &api.Error{StatusCode: 404, Method: "GET", Path: "/x", Body: []byte(`{"token":"SECRET"}`)}
	se := Classify(apiErr)
	if strings.Contains(se.Message, "SECRET") {
		t.Fatalf("response body must never leak into the user message: %q", se.Message)
	}
}

func TestClassifyUnknownFallsBack(t *testing.T) {
	se := Classify(errors.New("something weird"))
	if se.Code != "SEM000001" {
		t.Fatalf("code: %q", se.Code)
	}
	if se.Message != "something weird" {
		t.Fatalf("unknown error message should be preserved, got %q", se.Message)
	}
}

func TestClassifyNil(t *testing.T) {
	if Classify(nil) != nil {
		t.Fatal("Classify(nil) must return nil")
	}
}
