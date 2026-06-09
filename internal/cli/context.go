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

package cli

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/moep90/semaphore-cli/internal/api"
	"github.com/moep90/semaphore-cli/internal/auth"
	"github.com/moep90/semaphore-cli/internal/config"
	"github.com/moep90/semaphore-cli/internal/output"
	"github.com/moep90/semaphore-cli/internal/resolver"
)

// Context holds the runtime state for a command invocation.
type Context struct {
	Config  *config.Config
	Client  *api.Client
	Printer *output.Printer
	Project string
	Host    string
	IsTTY   bool
	NoColor bool
	Verbose bool
	Debug   bool
}

// BuildContext creates a CLI context from global flag overrides and configuration.
func BuildContext(cfg *config.Config, hostFlag, projectFlag, outputFlag, profileFlag string, noColor, verbose, debug bool) (*Context, error) {
	ctx := &Context{
		Config:  cfg,
		IsTTY:   isTerminal(os.Stdout),
		NoColor: noColor,
		Verbose: verbose,
		Debug:   debug,
	}

	// Resolve profile.
	profileName := firstNonEmpty(profileFlag, os.Getenv("SEMAPHORE_PROFILE"), cfg.CurrentProfile)
	var profile *config.Profile
	if profileName != "" {
		profile = cfg.Profiles[profileName]
	}
	if profile == nil && cfg.CurrentProfile != "" {
		profile = cfg.ActiveProfile()
	}

	// Resolve host.
	ctx.Host = firstNonEmpty(hostFlag, os.Getenv("SEMAPHORE_HOST"), profileField(profile, func(p *config.Profile) string { return p.Host }))
	if ctx.Host == "" {
		return nil, fmt.Errorf("no host configured; use --host or set SEMAPHORE_HOST, or run 'semctl auth login'")
	}
	if err := validateHost(ctx.Host); err != nil {
		return nil, err
	}

	// Resolve project.
	ctx.Project = firstNonEmpty(projectFlag, os.Getenv("SEMAPHORE_PROJECT"), profileField(profile, func(p *config.Profile) string { return p.Project }))

	// Resolve token and create client.
	token := auth.GetToken(ctx.Host, cfg)
	tokenSource := auth.GetTokenSource(ctx.Host, cfg)
	ctx.Client = api.NewClientWithSource(ctx.Host, token, tokenSource)
	if ctx.Debug {
		ctx.Client = ctx.Client.WithDebug(os.Stderr)
	}
	if ctx.Verbose {
		ctx.Client = ctx.Client.WithVerbose(os.Stderr)
	}

	if verbose {
		if profileName != "" {
			_, _ = fmt.Fprintf(os.Stderr, "[verbose] using profile: %s\n", profileName)
		}
		_, _ = fmt.Fprintf(os.Stderr, "[verbose] host: %s\n", ctx.Host)
		if ctx.Project != "" {
			_, _ = fmt.Fprintf(os.Stderr, "[verbose] project: %s\n", ctx.Project)
		}
		if tokenSource != "" {
			_, _ = fmt.Fprintf(os.Stderr, "[verbose] auth: %s\n", tokenSource)
		}
	}

	// Resolve output mode.
	modeStr := firstNonEmpty(outputFlag, os.Getenv("SEMAPHORE_OUTPUT"), profileField(profile, func(p *config.Profile) string { return p.DefaultOutput }))
	if modeStr == "" {
		modeStr = "table"
	}
	mode, err := output.ParseMode(modeStr)
	if err != nil {
		return nil, err
	}
	ctx.Printer = output.New(mode)

	return ctx, nil
}

// ResolveProjectID resolves the active project to a numeric ID.
func (c *Context) ResolveProjectID(ctx context.Context) (int, error) {
	if c.Project == "" {
		return 0, fmt.Errorf("no project configured; use --project or set SEMAPHORE_PROJECT, or run 'semctl project use'")
	}
	return resolver.ResolveProject(ctx, c.Client, c.Project)
}

// ResolveTemplateID resolves a template identifier to an ID.
func (c *Context) ResolveTemplateID(ctx context.Context, idOrName string) (int, error) {
	projectID, err := c.ResolveProjectID(ctx)
	if err != nil {
		return 0, err
	}
	return resolver.ResolveTemplate(ctx, c.Client, projectID, idOrName)
}

// ResolveTaskID resolves a task identifier to an ID.
func (c *Context) ResolveTaskID(ctx context.Context, idOrName string) (int, error) {
	projectID, err := c.ResolveProjectID(ctx)
	if err != nil {
		return 0, err
	}
	return resolver.ResolveTask(ctx, c.Client, projectID, idOrName)
}

// ResolveInventoryID resolves an inventory identifier to an ID.
func (c *Context) ResolveInventoryID(ctx context.Context, idOrName string) (int, error) {
	projectID, err := c.ResolveProjectID(ctx)
	if err != nil {
		return 0, err
	}
	return resolver.ResolveInventory(ctx, c.Client, projectID, idOrName)
}

// ResolveEnvironmentID resolves an environment identifier to an ID.
func (c *Context) ResolveEnvironmentID(ctx context.Context, idOrName string) (int, error) {
	projectID, err := c.ResolveProjectID(ctx)
	if err != nil {
		return 0, err
	}
	return resolver.ResolveEnvironment(ctx, c.Client, projectID, idOrName)
}

// ResolveKeystoreID resolves a keystore identifier to an ID.
func (c *Context) ResolveKeystoreID(ctx context.Context, idOrName string) (int, error) {
	projectID, err := c.ResolveProjectID(ctx)
	if err != nil {
		return 0, err
	}
	return resolver.ResolveKeystore(ctx, c.Client, projectID, idOrName)
}

// ResolveRepositoryID resolves a repository identifier to an ID.
func (c *Context) ResolveRepositoryID(ctx context.Context, idOrName string) (int, error) {
	projectID, err := c.ResolveProjectID(ctx)
	if err != nil {
		return 0, err
	}
	return resolver.ResolveRepository(ctx, c.Client, projectID, idOrName)
}

// ResolveScheduleID resolves a schedule identifier to an ID.
func (c *Context) ResolveScheduleID(ctx context.Context, idOrName string) (int, error) {
	projectID, err := c.ResolveProjectID(ctx)
	if err != nil {
		return 0, err
	}
	return resolver.ResolveSchedule(ctx, c.Client, projectID, idOrName)
}

// ResolveUserID resolves a user identifier to an ID.
func (c *Context) ResolveUserID(ctx context.Context, idOrName string) (int, error) {
	return resolver.ResolveUser(ctx, c.Client, idOrName)
}

// ValidateProjectConfigured returns an error if no project is set.
func (c *Context) ValidateProjectConfigured() error {
	if c.Project == "" {
		return fmt.Errorf("no project configured; use --project or set SEMAPHORE_PROJECT, or run 'semctl project use'")
	}
	return nil
}

// CurrentProfileName returns the active profile name.
func (c *Context) CurrentProfileName() string {
	return firstNonEmpty(c.Config.CurrentProfile, os.Getenv("SEMAPHORE_PROFILE"))
}

// WithTimeout returns a derived context with the given timeout.
func WithTimeout(ctx context.Context, duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, duration)
}

// LatestTaskID returns the latest task ID in the active project.
func (c *Context) LatestTaskID(ctx context.Context) (int, error) {
	projectID, err := c.ResolveProjectID(ctx)
	if err != nil {
		return 0, err
	}
	resp, err := c.Client.Do(ctx, "GET", fmt.Sprintf("/project/%d/tasks/last", projectID), nil)
	if err != nil {
		return 0, fmt.Errorf("fetch latest task: %w", err)
	}
	var task api.Task
	if err := api.DecodeJSON(resp, &task); err != nil {
		// Try array fallback.
		var tasks []api.Task
		resp, err2 := c.Client.Do(ctx, "GET", fmt.Sprintf("/project/%d/tasks", projectID), nil)
		if err2 != nil {
			return 0, fmt.Errorf("decode latest task: %w", err)
		}
		if err := api.DecodeJSON(resp, &tasks); err != nil {
			return 0, fmt.Errorf("decode tasks: %w", err)
		}
		if len(tasks) == 0 {
			return 0, fmt.Errorf("no tasks found in project")
		}
		return tasks[0].ID, nil
	}
	return task.ID, nil
}

// PrintError prints a structured error using the configured printer.
func (c *Context) PrintError(msg string, suggestions []string) {
	c.Printer.PrintError(msg, suggestions)
}

func validateHost(host string) error {
	u, err := url.Parse(host)
	if err != nil {
		return fmt.Errorf("invalid host URL: %w", err)
	}
	if !u.IsAbs() {
		return fmt.Errorf("host must be an absolute URL (e.g., https://semaphore.example.com)")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("host scheme must be https or http")
	}
	return nil
}

func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func profileField(p *config.Profile, fn func(*config.Profile) string) string {
	if p == nil {
		return ""
	}
	return fn(p)
}

// BuildCmdContext extracts global flags from a cobra command and builds a Context.
func BuildCmdContext(cmd *cobra.Command) (*Context, error) {
	hostFlag, _ := cmd.Flags().GetString("host")
	projectFlag, _ := cmd.Flags().GetString("project")
	outputFlag, _ := cmd.Flags().GetString("output")
	profileFlag, _ := cmd.Flags().GetString("profile")
	jsonFlag, _ := cmd.Flags().GetBool("json")
	noColor, _ := cmd.Flags().GetBool("no-color")
	verbose, _ := cmd.Flags().GetBool("verbose")
	debug, _ := cmd.Flags().GetBool("debug")

	// Only apply --json shorthand when --output was not explicitly set.
	if jsonFlag && !cmd.Flags().Changed("output") {
		outputFlag = "json"
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	ctx, err := BuildContext(cfg, hostFlag, projectFlag, outputFlag, profileFlag, noColor, verbose, debug)
	if err != nil {
		return nil, err
	}
	if out := cmd.OutOrStdout(); out != nil {
		ctx.Printer.Stdout = out
	} else {
		ctx.Printer.Stdout = os.Stdout
	}
	if errOut := cmd.ErrOrStderr(); errOut != nil {
		ctx.Printer.Stderr = errOut
	} else {
		ctx.Printer.Stderr = os.Stderr
	}
	return ctx, nil
}

// Atoi is a thin wrapper around strconv.Atoi.
func Atoi(s string) (int, error) {
	return strconv.Atoi(s)
}
