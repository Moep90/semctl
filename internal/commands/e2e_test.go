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

//go:build e2e

package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestE2EAuthAndProject(t *testing.T) {
	host, token := setupE2E(t)
	configDir := t.TempDir()

	// semctl auth login with token
	cmd := exec.Command("go", "run", "../../cmd/semctl", "auth", "login", host, "--with-token")
	cmd.Stdin = strings.NewReader(token)
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+configDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("auth login failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Authenticated") {
		t.Fatalf("expected auth success, got: %s", out)
	}

	// semctl project list
	cmd = exec.Command("go", "run", "../../cmd/semctl", "project", "list", "--host", host, "--output", "json")
	cmd.Env = append(os.Environ(), "SEMAPHORE_TOKEN="+token)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("project list failed: %v\n%s", err, out)
	}
	var projects []map[string]any
	if err := json.Unmarshal(out, &projects); err != nil {
		t.Fatalf("invalid project list json: %v\n%s", err, out)
	}
	if len(projects) == 0 {
		t.Fatal("expected at least one project")
	}
}

func TestE2EAuthLogout(t *testing.T) {
	host, token := setupE2E(t)
	configDir := t.TempDir()

	// Login first
	cmd := exec.Command("go", "run", "../../cmd/semctl", "auth", "login", host, "--with-token")
	cmd.Stdin = strings.NewReader(token)
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+configDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("auth login failed: %v\n%s", err, out)
	}

	// Logout
	cmd = exec.Command("go", "run", "../../cmd/semctl", "auth", "logout", host)
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+configDir)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("auth logout failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Logged out") && !strings.Contains(string(out), "Removed") {
		t.Fatalf("expected logout success, got: %s", out)
	}
}

func TestE2EProjectUse(t *testing.T) {
	host, token := setupE2E(t)
	configDir := t.TempDir()

	// Ensure project exists and get its name
	ensureProject(t, host, token)

	// Login
	cmd := exec.Command("go", "run", "../../cmd/semctl", "auth", "login", host, "--with-token")
	cmd.Stdin = strings.NewReader(token)
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+configDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("auth login failed: %v\n%s", err, out)
	}

	// project use
	cmd = exec.Command("go", "run", "../../cmd/semctl", "project", "use", "test-project")
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+configDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("project use failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "test-project") {
		t.Fatalf("expected project use to reference test-project, got: %s", out)
	}
}

func TestE2ETemplateAndTask(t *testing.T) {
	host, token := setupE2E(t)

	// Ensure project exists
	ensureProject(t, host, token)

	// semctl template list
	cmd := exec.Command("go", "run", "../../cmd/semctl", "template", "list", "--host", host, "--project", "test-project", "--output", "json")
	cmd.Env = append(os.Environ(), "SEMAPHORE_TOKEN="+token)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("template list failed: %v\n%s", err, out)
	}
	var templates []map[string]any
	if err := json.Unmarshal(out, &templates); err != nil {
		t.Fatalf("invalid template list json: %v\n%s", err, out)
	}
	if len(templates) == 0 {
		t.Fatal("expected at least one template")
	}

	// semctl task run
	cmd = exec.Command("go", "run", "../../cmd/semctl", "task", "run", "test-template", "--host", host, "--project", "test-project", "--message", "e2e test", "--output", "json")
	cmd.Env = append(os.Environ(), "SEMAPHORE_TOKEN="+token)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("task run failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Queued task") {
		t.Fatalf("expected task queued, got: %s", out)
	}
}

func TestE2ETaskLogsAndStop(t *testing.T) {
	host, token := setupE2E(t)
	ensureProject(t, host, token)
	ensureTemplate(t, host, token)

	// Run a task
	cmd := exec.Command("go", "run", "../../cmd/semctl", "task", "run", "test-template", "--host", host, "--project", "test-project", "--message", "e2e logs test", "--output", "json")
	cmd.Env = append(os.Environ(), "SEMAPHORE_TOKEN="+token)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("task run failed: %v\n%s", err, out)
	}

	// Extract task ID from output heuristically
	taskID := extractTaskID(string(out))
	if taskID == "" {
		t.Fatalf("could not extract task ID from output: %s", out)
	}

	// Try task logs (may be empty for a fresh task, but should not error)
	cmd = exec.Command("go", "run", "../../cmd/semctl", "task", "logs", taskID, "--host", host, "--project", "test-project")
	cmd.Env = append(os.Environ(), "SEMAPHORE_TOKEN="+token)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("task logs failed: %v\n%s", err, out)
	}

	// Stop the task
	cmd = exec.Command("go", "run", "../../cmd/semctl", "task", "stop", taskID, "--host", host, "--project", "test-project")
	cmd.Env = append(os.Environ(), "SEMAPHORE_TOKEN="+token)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("task stop failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "stopped") && !strings.Contains(string(out), "Stop") {
		t.Fatalf("expected task stop confirmation, got: %s", out)
	}
}

func TestE2EAPIInfo(t *testing.T) {
	host, token := setupE2E(t)

	cmd := exec.Command("go", "run", "../../cmd/semctl", "api", "GET", "/info", "--host", host)
	cmd.Env = append(os.Environ(), "SEMAPHORE_TOKEN="+token)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("api GET /info failed: %v\n%s", err, out)
	}
	var info map[string]any
	if err := json.Unmarshal(out, &info); err != nil {
		t.Fatalf("invalid api info json: %v\n%s", err, out)
	}
	if _, ok := info["version"]; !ok {
		t.Fatalf("expected 'version' in /info response, got: %s", out)
	}
}

func TestE2EPing(t *testing.T) {
	host, token := setupE2E(t)

	cmd := exec.Command("go", "run", "../../cmd/semctl", "ping", "--host", host)
	cmd.Env = append(os.Environ(), "SEMAPHORE_TOKEN="+token)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ping failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "pong") && !strings.Contains(string(out), "ok") && !strings.Contains(string(out), "OK") {
		t.Fatalf("expected pong/ok in ping output, got: %s", out)
	}
}

func setupE2E(t *testing.T) (string, string) {
	host := os.Getenv("SEMAPHORE_HOST")
	if host == "" {
		host = "http://localhost:3000"
	}
	token := os.Getenv("SEMAPHORE_TOKEN")
	if token == "" {
		token = createE2EToken(t, host)
	}
	// Wait for server to be ready
	for i := 0; i < 30; i++ {
		resp, err := http.Get(host + "/api/ping")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
	return host, token
}

func createE2EToken(t *testing.T, host string) string {
	// First, login with admin credentials to get a session
	resp, err := http.Post(host+"/api/auth/login", "application/json", strings.NewReader(`{"auth":"admin","password":"changeme"}`))
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("login unexpected status %d: %s", resp.StatusCode, body)
	}

	cookies := resp.Cookies()
	jar, _ := http.NewRequest("POST", host+"/api/user/tokens", strings.NewReader(`{"name":"e2e-test-token"}`))
	jar.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		jar.AddCookie(c)
	}

	client := &http.Client{}
	resp2, err := client.Do(jar)
	if err != nil {
		t.Fatalf("create token failed: %v", err)
	}
	defer resp2.Body.Close()
	body, _ := io.ReadAll(resp2.Body)
	if resp2.StatusCode >= 400 {
		t.Fatalf("create token failed %d: %s", resp2.StatusCode, body)
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode token response: %v", err)
	}
	return result.ID
}

func ensureProject(t *testing.T, host, token string) {
	req, _ := http.NewRequest("GET", host+"/api/projects", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	defer resp.Body.Close()
	var projects []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		t.Fatalf("decode projects: %v", err)
	}
	for _, p := range projects {
		if p["name"] == "test-project" {
			return
		}
	}
	// Create project
	req, _ = http.NewRequest("POST", host+"/api/projects", strings.NewReader(`{"name":"test-project","max_parallel_tasks":5}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	defer resp.Body.Close()
}

func ensureTemplate(t *testing.T, host, token string) {
	// Find test-project ID
	req, _ := http.NewRequest("GET", host+"/api/projects", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("list projects for template: %v", err)
	}
	defer resp.Body.Close()
	var projects []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		t.Fatalf("decode projects for template: %v", err)
	}
	var projectID float64
	for _, p := range projects {
		if p["name"] == "test-project" {
			projectID = p["id"].(float64)
			break
		}
	}
	if projectID == 0 {
		t.Fatal("test-project not found for template creation")
	}

	// Check if template already exists
	req, _ = http.NewRequest("GET", fmt.Sprintf("%s/api/project/%d/templates", host, int(projectID)), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("list templates: %v", err)
	}
	defer resp.Body.Close()
	var templates []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&templates); err != nil {
		t.Fatalf("decode templates: %v", err)
	}
	for _, tpl := range templates {
		if tpl["name"] == "test-template" {
			return
		}
	}

	// Create a minimal template
	body := fmt.Sprintf(`{"name":"test-template","playbook":"ping.yml","project_id":%d,"inventory_id":1,"repository_id":1,"environment_id":1}`, int(projectID))
	req, _ = http.NewRequest("POST", fmt.Sprintf("%s/api/project/%d/templates", host, int(projectID)), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create template: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("create template failed %d: %s", resp.StatusCode, b)
	}
}

func extractTaskID(output string) string {
	// Heuristic: look for a numeric ID in common output patterns
	fields := strings.Fields(output)
	for _, f := range fields {
		f = strings.TrimRight(f, ".")
		if _, err := fmt.Sscanf(f, "%d", new(int)); err == nil && len(f) < 12 {
			return f
		}
	}
	// Try to parse JSON
	var task struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(output), &task); err == nil && task.ID != "" {
		return task.ID
	}
	return ""
}
