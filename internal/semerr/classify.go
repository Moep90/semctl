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

	"github.com/moep90/semaphore-cli/internal/api"
)

// httpStatusToCode maps an HTTP status to its API error class.
var httpStatusToCode = map[int]string{
	400: "SEM500001",
	401: "SEM500002",
	403: "SEM500003",
	404: "SEM500004",
	409: "SEM500005",
	429: "SEM500006",
	500: "SEM500007",
	502: "SEM500008",
	503: "SEM500009",
	504: "SEM500010",
}

// Classify maps an arbitrary error to a *SemError. An already-classified error
// is returned as-is; a wrapped *api.Error is found via errors.As; anything else
// falls back to SEM000001 (UNKNOWN_ERROR) carrying the original message.
func Classify(err error) *SemError {
	if err == nil {
		return nil
	}

	var se *SemError
	if errors.As(err, &se) {
		return se
	}

	var apiErr *api.Error
	if errors.As(err, &apiErr) {
		return fromAPIError(apiErr)
	}

	return New("SEM000001").WithMessage(err.Error()).Wrap(err)
}

func fromAPIError(e *api.Error) *SemError {
	code, ok := httpStatusToCode[e.StatusCode]
	if !ok {
		code = "SEM500011" // API_UNEXPECTED_STATUS
	}
	se := New(code).WithHTTPStatus(e.StatusCode).Wrap(e)
	if e.Method != "" {
		se.With("method", e.Method)
	}
	if e.Path != "" {
		se.With("path", e.Path)
	}
	if e.RequestID != "" {
		se.With("request_id", e.RequestID)
	}
	// Build the message from method/path/status only — never the response body.
	if e.Method != "" && e.Path != "" {
		se.WithMessagef("%s %s returned HTTP %d.", e.Method, e.Path, e.StatusCode)
	} else {
		se.WithMessagef("The API returned HTTP %d.", e.StatusCode)
	}
	return se
}
