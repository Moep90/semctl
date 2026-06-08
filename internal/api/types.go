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

import "time"

// Project is a Semaphore UI project.
type Project struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	Created          time.Time `json:"created"`
	MaxParallelTasks int       `json:"max_parallel_tasks,omitempty"`
}

// Template is a task template.
type Template struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ProjectID   int    `json:"project_id,omitempty"`
	App         string `json:"app,omitempty"`
	Playbook    string `json:"playbook,omitempty"`
	Repository  string `json:"repository,omitempty"`
	Inventory   string `json:"inventory,omitempty"`
	Environment string `json:"environment,omitempty"`
}

// Task is a running or completed task.
type Task struct {
	ID         int       `json:"id"`
	TemplateID int       `json:"template_id"`
	ProjectID  int       `json:"project_id"`
	Status     string    `json:"status"`
	Message    string    `json:"message,omitempty"`
	Created    time.Time `json:"created"`
	Start      time.Time `json:"start,omitempty"`
	End        time.Time `json:"end,omitempty"`
	UserID     int       `json:"user_id,omitempty"`
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

// AuthLoginResponse is returned by the login endpoint.
type AuthLoginResponse struct {
	Token string `json:"token"`
}
