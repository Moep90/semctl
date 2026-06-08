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

package resolver

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/moep90/semaphore-cli/internal/api"
)

// ResolveProject resolves a project identifier to a project ID.
func ResolveProject(ctx context.Context, client *api.Client, idOrName string) (int, error) {
	if id, err := strconv.Atoi(idOrName); err == nil {
		return id, nil
	}

	var projects []api.Project
	resp, err := client.Do(ctx, "GET", "/projects", nil)
	if err != nil {
		return 0, fmt.Errorf("list projects: %w", err)
	}
	if err := api.DecodeJSON(resp, &projects); err != nil {
		return 0, fmt.Errorf("decode projects: %w", err)
	}

	return resolveByName(idOrName, projects, func(p api.Project) (int, string) {
		return p.ID, p.Name
	})
}

// ResolveTemplate resolves a template identifier to a template ID within a project.
func ResolveTemplate(ctx context.Context, client *api.Client, projectID int, idOrName string) (int, error) {
	if id, err := strconv.Atoi(idOrName); err == nil {
		return id, nil
	}

	path := fmt.Sprintf("/project/%d/templates", projectID)
	var templates []api.Template
	resp, err := client.Do(ctx, "GET", path, nil)
	if err != nil {
		return 0, fmt.Errorf("list templates: %w", err)
	}
	if err := api.DecodeJSON(resp, &templates); err != nil {
		return 0, fmt.Errorf("decode templates: %w", err)
	}

	return resolveByName(idOrName, templates, func(t api.Template) (int, string) {
		return t.ID, t.Name
	})
}

// ResolveTask resolves a task identifier to a task ID within a project.
func ResolveTask(ctx context.Context, client *api.Client, projectID int, idOrName string) (int, error) {
	if id, err := strconv.Atoi(idOrName); err == nil {
		return id, nil
	}

	path := fmt.Sprintf("/project/%d/tasks", projectID)
	var tasks []api.Task
	resp, err := client.Do(ctx, "GET", path, nil)
	if err != nil {
		return 0, fmt.Errorf("list tasks: %w", err)
	}
	if err := api.DecodeJSON(resp, &tasks); err != nil {
		return 0, fmt.Errorf("decode tasks: %w", err)
	}

	return resolveByName(idOrName, tasks, func(t api.Task) (int, string) {
		return t.ID, strconv.Itoa(t.ID)
	})
}

func resolveByName[T any](idOrName string, items []T, extract func(T) (int, string)) (int, error) {
	var exact []T
	var caseInsensitive []T
	var prefix []T

	for _, it := range items {
		_, name := extract(it)
		if name == idOrName {
			exact = append(exact, it)
			continue
		}
		if strings.EqualFold(name, idOrName) {
			caseInsensitive = append(caseInsensitive, it)
			continue
		}
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(idOrName)) {
			prefix = append(prefix, it)
		}
	}

	if len(exact) == 1 {
		id, _ := extract(exact[0])
		return id, nil
	}
	if len(exact) > 1 {
		return 0, ambiguousError(idOrName, exact, extract)
	}

	if len(caseInsensitive) == 1 {
		id, _ := extract(caseInsensitive[0])
		return id, nil
	}
	if len(caseInsensitive) > 1 {
		return 0, ambiguousError(idOrName, caseInsensitive, extract)
	}

	if len(prefix) == 1 {
		id, _ := extract(prefix[0])
		return id, nil
	}
	if len(prefix) > 1 {
		return 0, ambiguousError(idOrName, prefix, extract)
	}

	return 0, fmt.Errorf("not found: %s", idOrName)
}

// ResolveInventory resolves an inventory identifier to an inventory ID within a project.
func ResolveInventory(ctx context.Context, client *api.Client, projectID int, idOrName string) (int, error) {
	if id, err := strconv.Atoi(idOrName); err == nil {
		return id, nil
	}

	path := fmt.Sprintf("/project/%d/inventory", projectID)
	var inventories []api.Inventory
	resp, err := client.Do(ctx, "GET", path, nil)
	if err != nil {
		return 0, fmt.Errorf("list inventory: %w", err)
	}
	if err := api.DecodeJSON(resp, &inventories); err != nil {
		return 0, fmt.Errorf("decode inventory: %w", err)
	}

	return resolveByName(idOrName, inventories, func(i api.Inventory) (int, string) {
		return i.ID, i.Name
	})
}

// ResolveEnvironment resolves an environment identifier to an environment ID within a project.
func ResolveEnvironment(ctx context.Context, client *api.Client, projectID int, idOrName string) (int, error) {
	if id, err := strconv.Atoi(idOrName); err == nil {
		return id, nil
	}

	path := fmt.Sprintf("/project/%d/environment", projectID)
	var environments []api.Environment
	resp, err := client.Do(ctx, "GET", path, nil)
	if err != nil {
		return 0, fmt.Errorf("list environment: %w", err)
	}
	if err := api.DecodeJSON(resp, &environments); err != nil {
		return 0, fmt.Errorf("decode environment: %w", err)
	}

	return resolveByName(idOrName, environments, func(e api.Environment) (int, string) {
		return e.ID, e.Name
	})
}

// ResolveKeystore resolves a keystore identifier to a keystore ID within a project.
func ResolveKeystore(ctx context.Context, client *api.Client, projectID int, idOrName string) (int, error) {
	if id, err := strconv.Atoi(idOrName); err == nil {
		return id, nil
	}

	path := fmt.Sprintf("/project/%d/keys", projectID)
	var keystores []api.Keystore
	resp, err := client.Do(ctx, "GET", path, nil)
	if err != nil {
		return 0, fmt.Errorf("list keystore: %w", err)
	}
	if err := api.DecodeJSON(resp, &keystores); err != nil {
		return 0, fmt.Errorf("decode keystore: %w", err)
	}

	return resolveByName(idOrName, keystores, func(k api.Keystore) (int, string) {
		return k.ID, k.Name
	})
}

func ambiguousError[T any](idOrName string, items []T, extract func(T) (int, string)) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "name is ambiguous: %s\n\nMatches:\n", idOrName)
	for _, it := range items {
		id, name := extract(it)
		fmt.Fprintf(&sb, "  %d   %s\n", id, name)
	}
	sb.WriteString("\nUse an ID or a more specific name.")
	return fmt.Errorf("%s", sb.String())
}
