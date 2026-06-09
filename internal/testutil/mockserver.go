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

package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// expectation records a single expected request and its canned response.
type expectation struct {
	method string
	path   string
	status int
	body   string
}

// MockServer is a test HTTP server that returns pre-registered responses.
type MockServer struct {
	*httptest.Server

	mu           sync.Mutex
	expectations []expectation
	calls        []call
}

// call records a single HTTP request received by the server.
type call struct {
	method string
	path   string
}

// NewMockServer creates and starts a new MockServer.
func NewMockServer() *MockServer {
	m := &MockServer{}
	m.Server = httptest.NewServer(http.HandlerFunc(m.handler))
	return m
}

// Expect registers a response for a matching method and path.
func (m *MockServer) Expect(method, path string, status int, body string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expectations = append(m.expectations, expectation{
		method: method,
		path:   path,
		status: status,
		body:   body,
	})
}

// ExpectJSON marshals bodyObj as JSON and registers the response.
func (m *MockServer) ExpectJSON(method, path string, status int, bodyObj any) {
	b, err := json.Marshal(bodyObj)
	if err != nil {
		panic(err)
	}
	m.Expect(method, path, status, string(b))
}

// AssertCalled verifies that at least one request matched the given method and path.
func (m *MockServer) AssertCalled(t testing.TB, method, path string) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.calls {
		if c.method == method && c.path == path {
			return
		}
	}
	t.Fatalf("expected call to %s %s, but no matching call was recorded", method, path)
}

// Reset clears all expectations and recorded calls.
func (m *MockServer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.expectations = m.expectations[:0]
	m.calls = make([]call, 0)
}

// URL returns the base URL of the test server.
func (m *MockServer) URL() string {
	return m.Server.URL
}

// Close shuts down the test server.
func (m *MockServer) Close() {
	m.Server.Close()
}

// handler matches incoming requests against expectations in order.
func (m *MockServer) handler(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	m.calls = append(m.calls, call{method: r.Method, path: r.URL.Path})

	for _, exp := range m.expectations {
		if r.Method == exp.method && r.URL.Path == exp.path {
			m.mu.Unlock()
			w.WriteHeader(exp.status)
			_, _ = w.Write([]byte(exp.body))
			return
		}
	}
	m.mu.Unlock()

	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{"error":"no matching expectation"}`))
}
