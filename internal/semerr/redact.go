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

// allowedMetadataKeys is the allowlist of metadata keys that may appear in
// user-facing errors. Anything not listed here (tokens, cookies, auth headers,
// bodies, signed URLs, env vars, ...) is dropped at construction time.
var allowedMetadataKeys = map[string]bool{
	"method":        true,
	"path":          true,
	"status":        true,
	"request_id":    true,
	"operation":     true,
	"resource_type": true,
	"resource_id":   true,
	"retry_after":   true,
}

func allowedMetadataKey(key string) bool {
	return allowedMetadataKeys[key]
}
