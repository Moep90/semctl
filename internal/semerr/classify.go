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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net"
	"strings"
	"syscall"

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

	// Transport and serialization errors carry only the class default message —
	// the underlying error (which may include the full URL/query) stays in cause.
	if se := fromTransport(err); se != nil {
		return se
	}
	if se := fromSerialization(err); se != nil {
		return se
	}
	if se := fromLocal(err); se != nil {
		return se
	}

	return New("SEM000001").WithMessage(err.Error()).Wrap(err)
}

// fromLocal classifies local filesystem errors, or returns nil.
func fromLocal(err error) *SemError {
	if errors.Is(err, fs.ErrNotExist) {
		return New("SEM700001").Wrap(err)
	}
	if errors.Is(err, fs.ErrPermission) {
		return New("SEM700002").Wrap(err)
	}
	return nil
}

// fromTransport classifies network/transport errors, or returns nil.
func fromTransport(err error) *SemError {
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return New("SEM400001").Wrap(err)
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return New("SEM400005").Wrap(err)
	}
	if errors.Is(err, syscall.ECONNRESET) {
		return New("SEM400006").Wrap(err)
	}
	var netErr net.Error
	if (errors.As(err, &netErr) && netErr.Timeout()) || errors.Is(err, context.DeadlineExceeded) {
		return New("SEM400002").Wrap(err)
	}
	if isTLSError(err) {
		return New("SEM400003").Wrap(err)
	}
	return nil
}

// isTLSError reports whether err is a TLS/certificate failure.
func isTLSError(err error) bool {
	var (
		unknownAuthority x509.UnknownAuthorityError
		hostname         x509.HostnameError
		certInvalid      x509.CertificateInvalidError
		recordHeader     tls.RecordHeaderError
	)
	if errors.As(err, &unknownAuthority) || errors.As(err, &hostname) ||
		errors.As(err, &certInvalid) || errors.As(err, &recordHeader) {
		return true
	}
	return strings.Contains(err.Error(), "tls:")
}

// fromSerialization classifies response decode/schema errors, or returns nil.
func fromSerialization(err error) *SemError {
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return New("SEM600001").Wrap(err)
	}
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return New("SEM600002").Wrap(err)
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return New("SEM600001").Wrap(err)
	}
	if errors.Is(err, io.EOF) {
		return New("SEM600004").Wrap(err)
	}
	return nil
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
	if e.RetryAfter != "" {
		se.With("retry_after", e.RetryAfter)
	}
	// Build the message from method/path/status only — never the response body.
	if e.Method != "" && e.Path != "" {
		se.WithMessagef("%s %s returned HTTP %d.", e.Method, e.Path, e.StatusCode)
	} else {
		se.WithMessagef("The API returned HTTP %d.", e.StatusCode)
	}
	return se
}
