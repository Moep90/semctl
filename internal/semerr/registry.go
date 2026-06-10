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

// Package semerr provides stable, typed error classes for semctl. Each error
// the CLI surfaces maps to a Class with a stable SEMDDDNNN code, a semantic
// name, a human title, a remediation hint, and an exit code — so users, scripts,
// and support can reason about failures beyond a raw HTTP status. The registry
// below is the single source of truth; docs/errors.md is generated from it.
package semerr

import "sort"

// Class is a registry entry: the stable public contract for one error class.
type Class struct {
	Code           string
	Name           string
	Domain         string
	Title          string
	DefaultMessage string
	Hint           string
	Retryable      bool
	ExitCode       int
	Severity       string // error | warning | fatal
	UserActionable bool
	HTTPStatuses   []int
	DocsURL        string
}

// Exit codes by domain. Process exit currently stays 1 (see formatError); these
// are surfaced in output and reserved for a future opt-in / major release.
const (
	exitGeneric  = 1
	exitUsage    = 2
	exitConfig   = 3
	exitAuth     = 4
	exitNetwork  = 5
	exitAPI      = 6
	exitParse    = 7
	exitLocal    = 8
	exitNotFound = 44
	exitTemp     = 75 // temporary failure, retry may succeed
)

var registry = map[string]Class{
	// 000xxx — generic / unknown
	"SEM000001": {Code: "SEM000001", Name: "UNKNOWN_ERROR", Domain: "generic", Title: "Unknown error", DefaultMessage: "An unexpected error occurred.", ExitCode: exitGeneric, Severity: "error"},
	"SEM000002": {Code: "SEM000002", Name: "INTERNAL_INVARIANT_VIOLATION", Domain: "generic", Title: "Internal invariant violation", DefaultMessage: "An internal invariant was violated.", ExitCode: exitGeneric, Severity: "fatal"},
	"SEM000003": {Code: "SEM000003", Name: "UNIMPLEMENTED", Domain: "generic", Title: "Not implemented", DefaultMessage: "This functionality is not implemented.", ExitCode: exitGeneric, Severity: "error"},

	// 100xxx — CLI usage
	"SEM100001": {Code: "SEM100001", Name: "INVALID_ARGUMENT", Domain: "cli", Title: "Invalid argument", DefaultMessage: "An argument was invalid.", ExitCode: exitUsage, Severity: "error", UserActionable: true},
	"SEM100002": {Code: "SEM100002", Name: "MISSING_ARGUMENT", Domain: "cli", Title: "Missing argument", DefaultMessage: "A required argument was not provided.", ExitCode: exitUsage, Severity: "error", UserActionable: true},
	"SEM100003": {Code: "SEM100003", Name: "INVALID_FLAG", Domain: "cli", Title: "Invalid flag", DefaultMessage: "A flag value was invalid.", ExitCode: exitUsage, Severity: "error", UserActionable: true},
	"SEM100004": {Code: "SEM100004", Name: "UNSUPPORTED_OUTPUT_FORMAT", Domain: "cli", Title: "Unsupported output format", DefaultMessage: "The requested output format is not supported.", ExitCode: exitUsage, Severity: "error", UserActionable: true},
	"SEM100005": {Code: "SEM100005", Name: "COMMAND_USAGE_ERROR", Domain: "cli", Title: "Command usage error", DefaultMessage: "The command was used incorrectly.", ExitCode: exitUsage, Severity: "error", UserActionable: true},

	// 200xxx — config
	"SEM200001": {Code: "SEM200001", Name: "CONFIG_NOT_FOUND", Domain: "config", Title: "Config not found", DefaultMessage: "No configuration was found.", Hint: "Run `semctl auth login` or set --host/SEMAPHORE_HOST.", ExitCode: exitConfig, Severity: "error", UserActionable: true},
	"SEM200002": {Code: "SEM200002", Name: "CONFIG_INVALID", Domain: "config", Title: "Config invalid", DefaultMessage: "The configuration is invalid.", ExitCode: exitConfig, Severity: "error", UserActionable: true},
	"SEM200003": {Code: "SEM200003", Name: "PROFILE_NOT_FOUND", Domain: "config", Title: "Profile not found", DefaultMessage: "The requested profile was not found.", ExitCode: exitConfig, Severity: "error", UserActionable: true},
	"SEM200004": {Code: "SEM200004", Name: "CONFIG_PERMISSION_DENIED", Domain: "config", Title: "Config permission denied", DefaultMessage: "The configuration file could not be read due to permissions.", ExitCode: exitConfig, Severity: "error", UserActionable: true},
	"SEM200005": {Code: "SEM200005", Name: "ENVIRONMENT_INVALID", Domain: "config", Title: "Environment invalid", DefaultMessage: "An environment variable was invalid.", ExitCode: exitConfig, Severity: "error", UserActionable: true},

	// 300xxx — auth
	"SEM300001": {Code: "SEM300001", Name: "AUTH_TOKEN_MISSING", Domain: "auth", Title: "Authentication token missing", DefaultMessage: "No authentication token was provided.", Hint: "Run `semctl auth login` or set SEMAPHORE_TOKEN.", ExitCode: exitAuth, Severity: "error", UserActionable: true},
	"SEM300002": {Code: "SEM300002", Name: "AUTH_TOKEN_EXPIRED", Domain: "auth", Title: "Authentication token expired", DefaultMessage: "The authentication token has expired.", Hint: "Re-authenticate with `semctl auth login`.", ExitCode: exitAuth, Severity: "error", UserActionable: true},
	"SEM300003": {Code: "SEM300003", Name: "AUTH_TOKEN_INVALID", Domain: "auth", Title: "Authentication token invalid", DefaultMessage: "The authentication token is invalid.", ExitCode: exitAuth, Severity: "error", UserActionable: true},
	"SEM300004": {Code: "SEM300004", Name: "AUTH_SCOPE_INSUFFICIENT", Domain: "auth", Title: "Insufficient scope", DefaultMessage: "The token lacks the required scope.", ExitCode: exitAuth, Severity: "error", UserActionable: true},
	"SEM300005": {Code: "SEM300005", Name: "AUTH_INTERACTIVE_REQUIRED", Domain: "auth", Title: "Interactive authentication required", DefaultMessage: "Interactive authentication is required.", ExitCode: exitAuth, Severity: "error", UserActionable: true},

	// 400xxx — network / transport
	"SEM400001": {Code: "SEM400001", Name: "NETWORK_DNS_FAILURE", Domain: "network", Title: "DNS resolution failed", DefaultMessage: "The host could not be resolved.", Hint: "Check the host name and your network/DNS.", ExitCode: exitNetwork, Severity: "error", Retryable: true},
	"SEM400002": {Code: "SEM400002", Name: "NETWORK_TIMEOUT", Domain: "network", Title: "Network timeout", DefaultMessage: "The request timed out.", ExitCode: exitNetwork, Severity: "error", Retryable: true},
	"SEM400003": {Code: "SEM400003", Name: "TLS_HANDSHAKE_FAILED", Domain: "network", Title: "TLS handshake failed", DefaultMessage: "The TLS handshake failed.", ExitCode: exitNetwork, Severity: "error"},
	"SEM400004": {Code: "SEM400004", Name: "PROXY_ERROR", Domain: "network", Title: "Proxy error", DefaultMessage: "A proxy error occurred.", ExitCode: exitNetwork, Severity: "error"},
	"SEM400005": {Code: "SEM400005", Name: "CONNECTION_REFUSED", Domain: "network", Title: "Connection refused", DefaultMessage: "The connection was refused.", Hint: "Check that the host is reachable and the port is correct.", ExitCode: exitNetwork, Severity: "error", Retryable: true},
	"SEM400006": {Code: "SEM400006", Name: "CONNECTION_RESET", Domain: "network", Title: "Connection reset", DefaultMessage: "The connection was reset.", ExitCode: exitNetwork, Severity: "error", Retryable: true},

	// 500xxx — API response
	"SEM500001": {Code: "SEM500001", Name: "API_BAD_REQUEST", Domain: "api", Title: "API request rejected", DefaultMessage: "The API rejected the request as invalid.", ExitCode: exitAPI, Severity: "error", UserActionable: true, HTTPStatuses: []int{400}},
	"SEM500002": {Code: "SEM500002", Name: "API_UNAUTHENTICATED", Domain: "api", Title: "API authentication required", DefaultMessage: "Authentication is missing or invalid.", Hint: "Run `semctl auth login` or check SEMAPHORE_TOKEN.", ExitCode: exitAPI, Severity: "error", UserActionable: true, HTTPStatuses: []int{401}},
	"SEM500003": {Code: "SEM500003", Name: "API_FORBIDDEN", Domain: "api", Title: "API access forbidden", DefaultMessage: "The authenticated identity is not allowed to perform this action.", Hint: "Check that your token has access to this resource.", ExitCode: exitAPI, Severity: "error", UserActionable: true, HTTPStatuses: []int{403}},
	"SEM500004": {Code: "SEM500004", Name: "API_RESOURCE_NOT_FOUND", Domain: "api", Title: "API resource not found", DefaultMessage: "The requested API resource was not found.", Hint: "Check the resource ID, endpoint path, and that your token has access to it.", ExitCode: exitNotFound, Severity: "error", UserActionable: true, HTTPStatuses: []int{404}},
	"SEM500005": {Code: "SEM500005", Name: "API_CONFLICT", Domain: "api", Title: "API conflict", DefaultMessage: "The request conflicts with the current state of the resource.", ExitCode: exitAPI, Severity: "error", UserActionable: true, HTTPStatuses: []int{409}},
	"SEM500006": {Code: "SEM500006", Name: "API_RATE_LIMITED", Domain: "api", Title: "API rate limited", DefaultMessage: "The request was rate limited.", Hint: "Wait and retry; see retry_after if present.", ExitCode: exitTemp, Severity: "error", Retryable: true, HTTPStatuses: []int{429}},
	"SEM500007": {Code: "SEM500007", Name: "API_SERVER_ERROR", Domain: "api", Title: "API server error", DefaultMessage: "The API returned an internal server error.", ExitCode: exitTemp, Severity: "error", Retryable: true, HTTPStatuses: []int{500}},
	"SEM500008": {Code: "SEM500008", Name: "API_BAD_GATEWAY", Domain: "api", Title: "API bad gateway", DefaultMessage: "The API returned a bad gateway error.", ExitCode: exitTemp, Severity: "error", Retryable: true, HTTPStatuses: []int{502}},
	"SEM500009": {Code: "SEM500009", Name: "API_SERVICE_UNAVAILABLE", Domain: "api", Title: "API service unavailable", DefaultMessage: "The API is temporarily unavailable.", ExitCode: exitTemp, Severity: "error", Retryable: true, HTTPStatuses: []int{503}},
	"SEM500010": {Code: "SEM500010", Name: "API_GATEWAY_TIMEOUT", Domain: "api", Title: "API gateway timeout", DefaultMessage: "The API gateway timed out.", ExitCode: exitTemp, Severity: "error", Retryable: true, HTTPStatuses: []int{504}},
	"SEM500011": {Code: "SEM500011", Name: "API_UNEXPECTED_STATUS", Domain: "api", Title: "Unexpected API status", DefaultMessage: "The API returned an unexpected status.", ExitCode: exitAPI, Severity: "error"},

	// 600xxx — serialization
	"SEM600001": {Code: "SEM600001", Name: "RESPONSE_DECODE_FAILED", Domain: "serialization", Title: "Response decode failed", DefaultMessage: "The API response could not be decoded.", ExitCode: exitParse, Severity: "error"},
	"SEM600002": {Code: "SEM600002", Name: "RESPONSE_SCHEMA_INVALID", Domain: "serialization", Title: "Response schema invalid", DefaultMessage: "The API response did not match the expected schema.", ExitCode: exitParse, Severity: "error"},
	"SEM600003": {Code: "SEM600003", Name: "UNSUPPORTED_CONTENT_TYPE", Domain: "serialization", Title: "Unsupported content type", DefaultMessage: "The response content type is not supported.", ExitCode: exitParse, Severity: "error"},
	"SEM600004": {Code: "SEM600004", Name: "EMPTY_RESPONSE", Domain: "serialization", Title: "Empty response", DefaultMessage: "The API returned an empty response.", ExitCode: exitParse, Severity: "error"},

	// 700xxx — local runtime
	"SEM700001": {Code: "SEM700001", Name: "FILE_NOT_FOUND", Domain: "local", Title: "File not found", DefaultMessage: "A required file was not found.", ExitCode: exitLocal, Severity: "error", UserActionable: true},
	"SEM700002": {Code: "SEM700002", Name: "FILE_PERMISSION_DENIED", Domain: "local", Title: "File permission denied", DefaultMessage: "A file could not be accessed due to permissions.", ExitCode: exitLocal, Severity: "error", UserActionable: true},
	"SEM700003": {Code: "SEM700003", Name: "COMMAND_EXECUTION_FAILED", Domain: "local", Title: "Command execution failed", DefaultMessage: "A local command failed to execute.", ExitCode: exitLocal, Severity: "error"},
	"SEM700004": {Code: "SEM700004", Name: "LOCAL_STATE_LOCKED", Domain: "local", Title: "Local state locked", DefaultMessage: "Local state is locked by another process.", ExitCode: exitLocal, Severity: "error", Retryable: true},
}

// Lookup returns the registered Class for a code.
func Lookup(code string) (Class, bool) {
	c, ok := registry[code]
	return c, ok
}

// Classes returns all registered classes sorted by code (for docs generation
// and registry validation).
func Classes() []Class {
	out := make([]Class, 0, len(registry))
	for _, c := range registry {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}
