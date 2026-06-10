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

import "fmt"

// SemError is a runtime error carrying a registry Class plus contextual data.
// Its cause is preserved for --debug logging and errors.Is/As, but is never
// rendered into user-facing output.
type SemError struct {
	Class
	Message    string
	Metadata   map[string]string
	HTTPStatus int // 0 when the error has no HTTP status
	cause      error
}

// New constructs a SemError for the given code. An unknown code falls back to
// SEM000001 (UNKNOWN_ERROR) so a typo can never produce a code-less error. The
// message defaults to the class DefaultMessage until WithMessage overrides it.
func New(code string) *SemError {
	class, ok := Lookup(code)
	if !ok {
		class = registry["SEM000001"]
	}
	return &SemError{
		Class:    class,
		Message:  class.DefaultMessage,
		Metadata: make(map[string]string),
	}
}

// WithMessage sets the contextual, user-facing message.
func (e *SemError) WithMessage(msg string) *SemError {
	e.Message = msg
	return e
}

// WithMessagef sets the message from a format string.
func (e *SemError) WithMessagef(format string, args ...any) *SemError {
	e.Message = fmt.Sprintf(format, args...)
	return e
}

// With attaches an allowlisted metadata key. Keys outside the allowlist are
// dropped (defense in depth against leaking secrets into user output).
func (e *SemError) With(key, value string) *SemError {
	if !allowedMetadataKey(key) {
		return e
	}
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[key] = value
	return e
}

// WithHTTPStatus records the originating HTTP status (also added to metadata).
func (e *SemError) WithHTTPStatus(status int) *SemError {
	e.HTTPStatus = status
	return e.With("status", fmt.Sprintf("%d", status))
}

// Wrap records the underlying cause for debugging and errors.Is/As chains.
func (e *SemError) Wrap(cause error) *SemError {
	e.cause = cause
	return e
}

// Error implements error. It intentionally exposes only the code and message,
// never the cause body or metadata.
func (e *SemError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap exposes the cause for errors.Is/As.
func (e *SemError) Unwrap() error { return e.cause }
