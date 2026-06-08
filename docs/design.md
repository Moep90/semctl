> **Disclaimer:** `semctl` is an independent, open-source command line interface for Semaphore UI. It is **not affiliated with, endorsed by, sponsored by, or officially connected to the Semaphore UI project or its creators**. This tool is intended for personal use, educational purposes, and operational convenience at your own risk. All product names, logos, and brands are property of their respective owners.

# Design Document: Semaphore UI CLI (`semctl`)

This document is the source of truth for product behavior, user experience, and architecture decisions for the Semaphore UI command line interface.

[`Claude.md`](../Claude.md) is the source of truth for agent behavior and repository working rules. `docs/design.md` expands on product and technical design detail that `Claude.md` states concisely. When the two documents differ, `Claude.md` governs process and `docs/design.md` governs product behavior.

---

## 1. Overview

The CLI provides a productive terminal workflow for operators, platform engineers, and automation users who interact with Semaphore UI. It is inspired by the ergonomics of `gh` and `glab`: noun-first commands, readable tables by default, stable JSON for automation, explicit authentication, profiles, and an `api` escape hatch for uncovered endpoints.

The working binary name is `semctl`.

### Target workflow

```bash
semctl auth login https://semaphore.example.com
semctl project list
semctl project use homelab
semctl template list
semctl task run deploy-prod --message "Deploy release 1.8.3"
semctl task logs --follow
```

---

## 2. Goals

1. Provide a polished terminal client for day-to-day Semaphore UI usage.
2. Support both interactive human use and non-interactive automation without separate code paths.
3. Hide REST path details behind domain commands while keeping `semctl api` for advanced usage.
4. Resolve common resources by name where practical; accept IDs everywhere.
5. Support multiple Semaphore UI instances through profiles.
6. Store credentials securely by default.
7. Make destructive actions explicit and hard to trigger accidentally.
8. Provide output formats suitable for shell scripting.
9. Stay close to the public Semaphore UI API to reduce maintenance risk.

### Non-goals

1. Replace the Semaphore UI web interface completely.
2. Implement a local scheduler or execute jobs locally.
3. Store project state outside Semaphore UI, except local CLI configuration.
4. Reimplement full project import/export editing in the first release.
5. Depend on browser-only authentication as the only login mechanism.
6. Guess sensitive values from local files unless explicitly requested.

---

## 3. Command Model

### 3.1 Noun-first groups

Commands use Semaphore UI domain nouns as top-level groups. Verbs are nested:

```bash
semctl project list
semctl template get deploy-prod
semctl task logs 812
semctl runner list
```

Avoid raw REST paths as primary commands. Do not use hyphenated verbs like `list-projects` or `get-task-logs`.

Long-term command tree:

```text
semctl
├── auth
├── config
├── api
├── project
├── user
├── key
├── repo
├── inventory
├── environment
├── template
├── task
├── schedule
├── runner
├── integration
├── event
├── info
└── alias
```

Use `repo` rather than `repository`. Use `environment` as the canonical group; `env` may be added as an alias later.

### 3.2 Global flags

All commands support a consistent set of global flags:

```text
--host              Semaphore UI host URL
--profile           Configuration profile to use
--project, -p       Default project override
--output, -o        Output format override
--json              Shorthand for --output json
--jq                Filter JSON output client-side
--yes, -y           Skip confirmation prompts
--verbose           Verbose output
--debug             Debug output (redacts secrets)
--no-color          Disable colored output
--color             Force colored output
--no-interactive    Disable interactive prompts
```

### 3.3 Flag and configuration precedence

1. Explicit command flags
2. Environment variables
3. Active profile configuration
4. Built-in defaults

Supported environment variables:

```text
SEMAPHORE_HOST
SEMAPHORE_TOKEN
SEMAPHORE_PROFILE
SEMAPHORE_PROJECT
SEMAPHORE_OUTPUT
SEMAPHORE_NO_COLOR
```

Interactive prompts are disabled automatically when stdin/stdout is not a TTY, and may be forced off with `--no-interactive`.

### 3.4 Built-in aliases

Keep built-in aliases conservative to avoid surprising users:

```text
repo    → repository (if added later)
env     → environment (post-MVP)
tpl     → template (post-MVP)
logs    → task logs (post-MVP)
```

User-defined aliases may be supported later:

```bash
semctl alias set tl 'task list'
semctl alias set tf 'task logs --follow'
```

---

## 4. Authentication

### 4.1 Primary method: API token

The first release supports API token authentication. Username/password and OIDC-assisted login may follow in later milestones.

```bash
semctl auth login https://semaphore.example.com
```

Interactive flow:

```text
? Authentication method: API token
? Token: ********
✓ Authenticated as alice
✓ Stored credentials for https://semaphore.example.com
```

Token from stdin for scripts and password managers:

```bash
pass show semaphore/prod-token | semctl auth login https://semaphore.example.com --with-token
```

Non-persistent token for one-off commands:

```bash
SEMAPHORE_TOKEN=... semctl project list --host https://semaphore.example.com
```

### 4.2 Auth commands

MVP:

```bash
semctl auth login [HOST]
semctl auth logout [HOST]
semctl auth status
```

Post-MVP:

```bash
semctl auth token list
semctl auth token create [NAME]
semctl auth token revoke <TOKEN_ID>
```

### 4.3 Credential storage hierarchy

1. OS keychain or credential manager.
2. Encrypted local storage if available.
3. Plaintext config only with explicit user confirmation and a clear warning.

The CLI must never print stored tokens.

---

## 5. Configuration

### 5.1 Platform-native paths

Linux:

```text
$XDG_CONFIG_HOME/semctl/config.yml
~/.config/semctl/config.yml
```

macOS:

```text
~/Library/Application Support/semctl/config.yml
```

Windows:

```text
%AppData%\semctl\config.yml
```

### 5.2 Profile structure

```yaml
current_profile: prod

profiles:
  prod:
    host: https://semaphore.example.com
    project: infra
    token_source: keyring
    default_output: table

  lab:
    host: https://semaphore-lab.example.com
    project: homelab
    token_source: env
```

### 5.3 Configuration commands

```bash
semctl config get <KEY>
semctl config set <KEY> <VALUE>
semctl config list
semctl config profile list
semctl config profile use <NAME>
semctl config profile create <NAME> --host <URL>
semctl config profile delete <NAME>
```

---

## 6. Output Formats

### 6.1 Default behavior

Default to compact tables when stdout is a TTY. Default to JSON when `--json` is passed. Automation should not need to detect TTY.

```bash
semctl project list
semctl project list --json
semctl project list --jq '.[] | select(.name == "infra").id'
```

### 6.2 Supported modes

```text
--output table        Human-readable tables (TTY default)
--output json         Stable JSON for automation
--output yaml         Review and config-like workflows
--output text         Single values
--json                Shorthand for --output json
--jq <FILTER>         Client-side JSON filtering
--template <TEMPLATE> Go template output (post-MVP)
```

### 6.3 JSON as a compatibility contract

JSON field names and structure are a compatibility contract. Do not change JSON field names or nesting for cosmetic reasons. Additions are acceptable; removals or renames require versioning consideration.

### 6.4 Golden output tests

CLI output formatting must be stable. Use golden files for table, JSON, YAML, and error output fixtures. This ensures contributors detect unintended formatting changes.

---

## 7. API Client Architecture

### 7.1 Package responsibilities

```text
cmd/semctl/
  main.go               Entry point only

internal/
  cli/                  Command wiring, global flags, TTY detection
  config/               Config loading, profile management, path resolution
  auth/                 Token storage abstraction, login/logout/status
  api/                  HTTP client, request building, response parsing,
                        error mapping, retry logic
  resolver/             Name-to-ID resolution, project context injection
  output/               Table, JSON, YAML, JQ, and template renderers
  commands/             Per-domain command handlers (project, task, template, ...)
```

Design rules:

- Keep command handlers thin. They validate flags, call internal packages, and render output.
- Put all HTTP behavior in the `api` package.
- Put all config/profile logic in the `config` package.
- Put all output rendering in the `output` package.
- Avoid global mutable state except for command context.
- Return structured errors where practical so that `output` can render them consistently.

### 7.2 Hand-coded models vs. generated models

Do not generate a huge API client for the MVP. A pragmatic two-tier approach reduces maintenance risk:

1. **Hand-code models** for high-value resources used in the core workflow: `Project`, `Template`, `Task`, `TaskOutput`, `Repository`, `Inventory`, `Environment`, `AccessKey`, `Runner`, `User`.
2. **Use `semctl api`** for everything else. This lets the CLI remain useful even when the API evolves faster than first-class commands.
3. **Generate models later** only if the OpenAPI spec becomes stable enough that code generation clearly improves maintenance.

### 7.3 API escape hatch

The `api` command exposes authenticated raw API access. The path is relative to `/api`.

```bash
semctl api <METHOD> <PATH> [flags]
```

Examples:

```bash
semctl api GET /info
semctl api GET /projects
semctl api POST /project/1/tasks --field template_id=7
semctl api GET /project/1/tasks/last --jq '.[0]'
```

Supported flags:

```text
--field, -F       Typed field
--raw-field, -f   String field
--input           Read JSON request body from file or stdin
--header, -H      Add HTTP header
--paginate        Follow pagination (post-MVP)
--jq              Filter JSON output
```

---

## 8. Resource Resolution

### 8.1 Resolution order

For resources where names are supported, resolve in this order:

1. Numeric ID.
2. Exact name match.
3. Case-insensitive exact name match.
4. Prefix match only if unique.
5. Fail with a clear error and show matching candidates.

Project resolution should use the active profile project unless `--project` is supplied.

### 8.2 Example: task run

```bash
semctl task run deploy-prod
```

Resolution steps:

1. Determine active host from flag, env var, or profile.
2. Determine active project from flag, env var, or profile.
3. Resolve `deploy-prod` as a template in that project.
4. Create a task request using the resolved template ID.
5. Print the queued task ID and next-step commands.

Output:

```text
✓ Queued task 812 from template deploy-prod

View logs:
  semctl task logs 812 --follow
```

### 8.3 Ambiguity handling

If a name is ambiguous, fail with candidates. Do not guess.

```text
error: template name is ambiguous: deploy

Matches:
  7   deploy-dev
  8   deploy-prod

Use an ID or a more specific name.
```

---

## 9. Task and Log UX

Tasks are the highest-frequency operational commands and the core of the operator workflow.

### 9.1 Core task commands

```bash
semctl task list
semctl task last
semctl task get <TASK>
semctl task run <TEMPLATE>
semctl task stop <TASK>
semctl task logs [TASK]
```

`semctl template run <TEMPLATE>` aliases to `semctl task run <TEMPLATE>`.

### 9.2 Default-to-latest shortcut

When no task ID is supplied and an active project is set, `logs`, `status`, and `watch` may default to the latest task. This is convenient but must fail with a helpful message if there is no active project.

```bash
semctl task logs
semctl task status
semctl task watch
```

### 9.3 Follow and watch modes

`semctl task logs --follow` should poll the task output endpoint. The first implementation prefers polling over websockets because polling is easier to debug and behaves consistently behind proxies.

Recommended flags:

```text
--follow, -f     Stream new output
--raw            Skip formatting
--since          Start from a specific time
--tail           Limit initial output lines
--interval       Polling interval (default sensible, e.g. 2s)
```

`semctl task watch` waits for a task to complete and returns an exit code.

```bash
semctl task run deploy-prod --watch --exit-code
```

### 9.4 Exit codes for automation

`semctl task watch` and `semctl task run --watch --exit-code` should return predictable exit codes:

```text
0  task succeeded
1  task failed
2  task stopped or canceled
3  timeout
4  CLI or API error
```

This makes the CLI usable in CI pipelines without parsing JSON.

---

## 10. Security

### 10.1 Secrets handling

The CLI must never print:

- API tokens.
- Passwords.
- SSH private keys.
- Runner registration tokens after initial display.
- Environment secret values, unless the API explicitly returns them and the user explicitly requests them.

Commands that accept secrets should prefer stdin or file flags over direct arguments:

```bash
--password-stdin
--token-stdin
--private-key-file
```

Warn if secret input is passed directly as a shell argument.

### 10.2 Debug log redaction

`--debug` must redact the following fields and headers:

```text
Authorization
Cookie
password
private_key
passphrase
token
secret
registration_token
```

### 10.3 TLS

TLS verification is enabled by default. Optional insecure mode:

```bash
--insecure-skip-tls-verify
```

This must print a warning and must not be persisted to config unless explicitly configured.

---

## 11. Error Handling

Errors must be actionable.

Bad:

```text
Error: 404
```

Good:

```text
error: template not found: deploy-prod

Searched project:
  infra

Try:
  semctl template list --project infra
```

### 11.1 HTTP error mapping

```text
400  invalid request — show validation details
401  not authenticated — suggest semctl auth login
403  authenticated but not authorized
404  resource not found
409  conflict — show server message
500  server error — suggest --debug if needed
```

### 11.2 Ambiguous names

See Resource Resolution for the ambiguity error format.

---

## 12. Testing Strategy

Use a layered testing strategy. The repository provides a `compose.yaml` that starts Semaphore UI and all dependencies; the compose stack is the canonical test target. Do not run tests against a developer's personal Semaphore UI instance by default.

### 12.1 Unit tests

Cover pure logic:

- Config precedence.
- Profile selection.
- Auth token selection.
- URL construction.
- API error mapping.
- Output rendering.
- Name resolution.
- Secret redaction.

### 12.2 Golden output tests

Use fixtures for table, JSON, YAML, and error output. This is important because CLI users rely on stable output formatting. Store fixtures under `testdata/golden/`.

### 12.3 Integration tests

Run against the disposable Semaphore UI instance from `compose.yaml`.

Test:

- Login or token auth.
- Project listing.
- Template listing.
- Task creation.
- Task log retrieval.
- Stop task.
- API escape hatch.

Opt-in via environment variable and build tag:

```bash
SEMAPHORE_INTEGRATION=1 go test -tags=integration ./... -count=1
```

### 12.4 End-to-end tests

Prefer black-box CLI execution for E2E tests:

```go
cmd := exec.Command("semctl", "project", "list", "--json")
```

Avoid internal package calls in E2E tests except for fixture setup.

Critical E2E path:

```bash
semctl auth login
semctl project list
semctl project use
semctl template list
semctl task run
semctl task logs
semctl task stop
semctl api GET /info
```

Opt-in via:

```bash
SEMAPHORE_E2E=1 go test -tags=e2e ./... -count=1
```

### 12.5 Test requirements

- Seed known test data into the compose Semaphore UI server.
- Use unique resource names for mutating tests.
- Clean up created resources.
- Avoid test order dependencies.
- Avoid sleeps when polling with deadlines or health checks works.
- Keep timeouts explicit.
- Never print test credentials or tokens.

### 12.6 CI pipeline

CI should run formatting, linting, unit tests, golden tests, integration tests against `compose.yaml`, and E2E tests for the critical path.

---

## 13. MVP Scope

### 13.1 MVP commands

Focus on the high-frequency operator path. Do not add broad CRUD coverage before the task workflow is excellent.

```text
semctl auth login
semctl auth logout
semctl auth status

semctl config profile list
semctl config profile use
semctl config set
semctl config get

semctl project list
semctl project get
semctl project use

semctl template list
semctl template get

semctl task list
semctl task last
semctl task get
semctl task run
semctl task stop
semctl task logs

semctl api
semctl info
semctl ping
```

### 13.2 Target workflows

First-time setup:

```bash
semctl auth login https://semaphore.example.com
semctl project list
semctl project use infra
semctl template list
```

Run a deployment:

```bash
semctl task run deploy-prod \
  --message "Deploy release 1.8.3" \
  --branch release/1.8 \
  --watch \
  --exit-code
```

Inspect a failed task:

```bash
semctl task last
semctl task logs --tail 200
semctl task get --output yaml
```

Scripted run in CI:

```bash
export SEMAPHORE_HOST=https://semaphore.example.com
export SEMAPHORE_TOKEN="$SEMAPHORE_API_TOKEN"
export SEMAPHORE_PROJECT=infra

TASK_ID="$(semctl task run deploy-prod --json | jq -r '.id')"
semctl task logs "$TASK_ID" --follow
semctl task watch "$TASK_ID" --exit-code
```

Raw API call:

```bash
semctl api GET /project/1/tasks/last --jq '.[0]'
```

### 13.3 MVP acceptance criteria

1. A user can authenticate to a Semaphore UI instance with an API token.
2. A user can configure a default host and project.
3. A user can list projects and templates.
4. A user can run a template and follow task logs.
5. A user can stop a task.
6. A CI job can perform the same workflow using only environment variables and JSON output.
7. Unsupported endpoints work through `semctl api`.
8. The CLI works on Linux, macOS, and Windows.
9. Secrets are not printed in normal or debug output.
10. `semctl api` can call endpoints not yet covered by first-class commands.

---

## 14. Post-MVP Roadmap

### Milestone 2: Resource management

Add create, edit, delete support for:

```text
projects
repositories
inventories
environments
keys
templates
schedules
```

### Milestone 3: Admin and runner operations

Add:

```text
users
project users
project runners
global runners
runner tags
runner registration tokens
runner cache clear
```

### Milestone 4: Integrations

Add:

```text
integrations
integration aliases
matchers
extracted values
```

### Milestone 5: Advanced UX

Add:

```text
user-defined aliases
dynamic shell completion
template-driven output
interactive create flows
OIDC assisted login
task websocket follow mode
```

---

## 15. Packaging and Distribution

### 15.1 Target platforms

```text
Linux amd64
Linux arm64
macOS amd64
macOS arm64
Windows amd64
```

### 15.2 Distribution channels

```text
GitHub Releases
Homebrew tap
Linux packages: deb, rpm
Container image
Scoop or Winget for Windows
go install for developers
```

Example developer install:

```bash
go install github.com/moep90/semaphore-cli/cmd/semctl@latest
```

### 15.3 Shell completion

Support static completions at minimum:

```bash
semctl completion bash
semctl completion zsh
semctl completion fish
semctl completion powershell
```

Dynamic completion for project, template, and task names can be added in a later milestone.

---

## 16. Open Questions

1. Should project backup and restore be MVP or post-MVP?
2. Should the CLI generate local YAML manifests for templates and inventories?
3. Should the CLI support plugin extensions, or is the `api` escape hatch sufficient?
4. Should the project name be globally unique in common deployments, or should the CLI always display IDs prominently?
5. Should `semctl task run` accept template name prefixes, or only exact names?
