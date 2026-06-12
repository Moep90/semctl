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

package api

import (
	"errors"
	"time"
)

// Project is a Semaphore UI project.
type Project struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	Created          time.Time `json:"created"`
	MaxParallelTasks int       `json:"max_parallel_tasks,omitempty"`
}

// Template is a task template.
type Template struct {
	ID                        int    `json:"id"`
	Name                      string `json:"name"`
	ProjectID                 int    `json:"project_id,omitempty"`
	App                       string `json:"app,omitempty"`
	Playbook                  string `json:"playbook,omitempty"`
	Repository                string `json:"repository,omitempty"`
	Inventory                 string `json:"inventory,omitempty"`
	Environment               string `json:"environment,omitempty"`
	InventoryID               int    `json:"inventory_id,omitempty"`
	EnvironmentID             int    `json:"environment_id,omitempty"`
	RepositoryID              int    `json:"repository_id,omitempty"`
	ViewID                    int    `json:"view_id,omitempty"`
	GitBranch                 string `json:"git_branch,omitempty"`
	AllowOverrideBranchInTask bool   `json:"allow_override_branch_in_task,omitempty"`
	SuppressSuccessAlert      bool   `json:"suppress_success_alert,omitempty"`
}

// OmitZeroTime is a time.Time that marshals to JSON null when it holds the zero
// value. The Semaphore API returns Go's zero time ("0001-01-01T00:00:00Z") for
// timestamps that have not happened yet (e.g. `end` on a still-running task);
// surfacing that as null is less misleading to consumers than the zero string.
type OmitZeroTime struct {
	time.Time
}

// MarshalJSON renders the zero time as null and otherwise defers to time.Time.
func (t OmitZeroTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return t.Time.MarshalJSON()
}

// Task is a running or completed task.
type Task struct {
	ID            int          `json:"id"`
	TemplateID    int          `json:"template_id"`
	ProjectID     int          `json:"project_id"`
	Status        string       `json:"status"`
	Message       string       `json:"message,omitempty"`
	Playbook      string       `json:"playbook,omitempty"`
	Environment   string       `json:"environment,omitempty"`
	GitBranch     string       `json:"git_branch,omitempty"`
	Limit         string       `json:"limit,omitempty"`
	CommitHash    string       `json:"commit_hash,omitempty"`
	CommitMessage string       `json:"commit_message,omitempty"`
	Arguments     string       `json:"arguments,omitempty"`
	InventoryID   int          `json:"inventory_id,omitempty"`
	Created       time.Time    `json:"created"`
	Start         OmitZeroTime `json:"start"`
	End           OmitZeroTime `json:"end"`
	UserID        int          `json:"user_id,omitempty"`
}

// TaskOutput is a single log line from a task.
type TaskOutput struct {
	TaskID int    `json:"task_id,omitempty"`
	Time   string `json:"time,omitempty"`
	Output string `json:"output"`
}

// User is a Semaphore UI user.
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`
}

// Info holds server information.
type Info struct {
	Version string `json:"version,omitempty"`
}

// AuthLoginRequest is sent to the cookie-based login endpoint.
type AuthLoginRequest struct {
	Auth     string `json:"auth"`
	Password string `json:"password"`
}

// AuthLoginResponse is returned by the login endpoint.
type AuthLoginResponse struct {
	Token string `json:"token"`
}

// Inventory is a Semaphore UI inventory.
type Inventory struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	ProjectID    int    `json:"project_id,omitempty"`
	Type         string `json:"type,omitempty"`
	Inventory    string `json:"inventory,omitempty"`
	SSHKeyID     *int   `json:"ssh_key_id"`
	BecomeKeyID  *int   `json:"become_key_id"`
	RepositoryID *int   `json:"repository_id"`
	TemplateID   *int   `json:"template_id"`
}

// Environment is a Semaphore UI environment.
type Environment struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ProjectID int    `json:"project_id,omitempty"`
	JSON      string `json:"json,omitempty"`
}

// TaskRunRequest is the body sent to launch a task.
type TaskRunRequest struct {
	TemplateID    int    `json:"template_id"`
	Message       string `json:"message,omitempty"`
	GitBranch     string `json:"git_branch,omitempty"`
	EnvironmentID int    `json:"environment_id,omitempty"`
	InventoryID   int    `json:"inventory_id,omitempty"`
	Limit         string `json:"limit,omitempty"`
	Diff          bool   `json:"diff,omitempty"`
	DryRun        bool   `json:"dry_run,omitempty"`
	Tags          string `json:"tags,omitempty"`
	SkipTags      string `json:"skip_tags,omitempty"`
	// Environment carries Ansible extra variables and survey-variable answers as
	// a JSON-encoded string. Semaphore reads this field (not "extra_vars", which
	// it silently ignores) when applying variables to a run.
	Environment string `json:"environment,omitempty"`
	Check       bool   `json:"check,omitempty"`
}

// Keystore is a Semaphore UI access key / keystore entry.
type Keystore struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ProjectID int    `json:"project_id,omitempty"`
	Type      string `json:"type,omitempty"`
}

// Repository is a Semaphore UI repository.
type Repository struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ProjectID int    `json:"project_id,omitempty"`
	GitURL    string `json:"git_url,omitempty"`
	Branch    string `json:"branch,omitempty"`
}

// Schedule is a scheduled task template execution. Field names follow the
// Semaphore API body schema (`cron_format`, `active`), not the older
// `cron_expression`/`enabled` names, which the API never returns.
type Schedule struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	ProjectID      int    `json:"project_id,omitempty"`
	TemplateID     int    `json:"template_id,omitempty"`
	CronFormat     string `json:"cron_format,omitempty"`
	Active         bool   `json:"active"`
	Type           string `json:"type"`
	DeleteAfterRun bool   `json:"delete_after_run"`
	RepositoryID   *int   `json:"repository_id"`
}

// UserDetail extends User with additional metadata.
type UserDetail struct {
	User
	Admin   bool      `json:"admin,omitempty"`
	Created time.Time `json:"created"`
}

// TaskSummary is a lightweight representation of a task.
type TaskSummary struct {
	ID         int       `json:"id"`
	TemplateID int       `json:"template_id"`
	Status     string    `json:"status"`
	Message    string    `json:"message,omitempty"`
	Created    time.Time `json:"created"`
}

// ListOptions holds common list query options.
type ListOptions struct {
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
	SortField string `json:"sort_field,omitempty"`
	SortOrder string `json:"sort_order,omitempty"`
}

// ValidateProjectID returns an error if the project ID is not positive.
func ValidateProjectID(id int) error {
	if id <= 0 {
		return errors.New("project id must be greater than 0")
	}
	return nil
}

// ValidateTemplateID returns an error if the template ID is not positive.
func ValidateTemplateID(id int) error {
	if id <= 0 {
		return errors.New("template id must be greater than 0")
	}
	return nil
}
