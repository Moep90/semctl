# CLAUDE.md

## Project

Build a command line interface for Semaphore UI.

Working binary name: `semctl`.

The CLI should feel similar to `gh` and `glab`: noun-first commands, readable tables by default, stable JSON for automation, explicit auth, profiles, aliases, and an `api` escape hatch.

Primary API reference:

* https://semaphoreui.com/api-docs

The API is expected to expose `/api` endpoints for auth, projects, templates, tasks, repositories, inventories, environments, keys, schedules, users, integrations, events, and runners.

## Agent guidance

Before implementing or changing behavior:

1. Identify the relevant Semaphore UI API endpoint.
2. Confirm request and response shapes from the API docs.
3. Add or update API client code.
4. Add or update command wiring.
5. Add output rendering.
6. Add tests.
7. Update examples or docs when behavior changes.

Use these working modes:

* API work: inspect the Semaphore UI Swagger/OpenAPI docs first.
* CLI work: follow `gh` and `glab` style conventions.
* Auth work: prioritize secure token handling and redaction.
* Output work: preserve stable JSON and test tables with golden files.
* Task/log work: optimize for operators and CI usage.
* Refactoring: keep command handlers thin and move logic into internal packages.

Do not add broad CRUD coverage before the MVP task workflow is excellent.

## Product goals

* Provide a polished terminal client for Semaphore UI operators.
* Optimize for authenticate, select project, list templates, run tasks, and follow logs.
* Support interactive human usage and non-interactive automation.
* Prefer domain commands over raw REST paths.
* Keep `semctl api` for unsupported endpoints.
* Support multiple Semaphore UI instances through profiles.
* Be secure by default. Never leak secrets.

## Non-goals

* Do not reimplement Semaphore UI locally.
* Do not execute Ansible, Terraform, Shell, PowerShell, or Python tasks locally.
* Do not build a local scheduler.
* Do not hide API errors behind vague messages.
* Do not store secrets in plaintext unless the user explicitly accepts it.

## Implementation

Use Go unless the repository already uses another language.

Recommended libraries:

* `cobra` for commands
* `viper` or a small custom config layer
* OS keyring library for credential storage
* table renderer for human output
* YAML and JSON encoders
* optional jq-compatible filtering later

Keep implementation boring, explicit, and testable.

Suggested layout:

```text
cmd/semctl
internal/api
internal/auth
internal/cli
internal/commands
internal/config
internal/output
internal/resolver
```

Rules:

* Keep command handlers thin.
* Put HTTP behavior in an API client package.
* Put config/profile logic in one package.
* Put output rendering in one package.
* Avoid global mutable state except command context.
* Return structured errors where practical.
* Prefer explicit structs for MVP resources.
* Do not generate a huge API client unless it clearly improves maintenance.

## Development environment

Prefer `mise` for local developer tooling.

Use `mise.toml` to pin development tools and expose common tasks. Do not make `mise` mandatory for end users.

Expected workflow:

```bash
mise install
mise run fmt
mise run lint
mise run test
mise run vuln
mise run build
mise run test:integration
mise run test:e2e
```

Every important `mise` task must map to an obvious plain command.

Example tasks:

```toml
[tools]
go = "1.23"
golangci-lint = "latest"
goreleaser = "latest"
govulncheck = "latest"

[env]
CGO_ENABLED = "0"

[tasks.fmt]
run = "go fmt ./..."

[tasks.lint]
run = "golangci-lint run ./..."

[tasks.test]
run = "go test ./..."

[tasks.vuln]
run = "govulncheck ./..."

[tasks.build]
run = "go build -o bin/semctl ./cmd/semctl"

[tasks.compose-up]
run = "docker compose up -d --wait"

[tasks.compose-down]
run = "docker compose down -v"

[tasks.test-integration]
depends = ["compose-up"]
env = { SEMAPHORE_INTEGRATION = "1" }
run = "go test -tags=integration ./... -count=1"

[tasks.test-e2e]
depends = ["compose-up"]
env = { SEMAPHORE_E2E = "1" }
run = "go test -tags=e2e ./... -count=1"
```

## CLI style

Use noun-first command groups.

Good:

```bash
semctl auth login https://semaphore.example.com
semctl project list
semctl project use infra
semctl template list
semctl task run deploy-prod
semctl task logs --follow
semctl api GET /info
```

Avoid:

```bash
semctl list-projects
semctl get-task-logs
```

Global flags:

```text
--host
--profile
--project, -p
--output, -o
--json
--jq
--yes, -y
--verbose
--debug
--no-color
--color
--no-interactive
```

Precedence:

1. Explicit flags
2. Environment variables
3. Active profile config
4. Defaults

Environment variables:

```text
SEMAPHORE_HOST
SEMAPHORE_TOKEN
SEMAPHORE_PROFILE
SEMAPHORE_PROJECT
SEMAPHORE_OUTPUT
SEMAPHORE_NO_COLOR
```

## Configuration

Use platform-native config locations.

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

Example:

```yaml
current_profile: prod
profiles:
  prod:
    host: https://semaphore.example.com
    project: infra
    token_source: keyring
    default_output: table
```

## Authentication

Support API token auth first.

Required commands:

```bash
semctl auth login [HOST]
semctl auth logout [HOST]
semctl auth status
```

Support:

```bash
semctl auth login https://semaphore.example.com
semctl auth login https://semaphore.example.com --with-token
SEMAPHORE_TOKEN=... semctl project list --host https://semaphore.example.com
```

Prefer OS credential storage (macOS Keychain, Windows Credential Manager, Linux Secret Service). If the keyring is unavailable, fail hard unless the user passes `--plaintext` to opt into config-file storage. Never print tokens. Interactive prompts must suppress terminal echo (e.g., `term.ReadPassword`). Validate host URLs as absolute `https://` or `http://` before making requests.

## MVP scope

Implement these first:

```text
auth login, logout, status
config get, set, profile list, profile use
project list, get, use
template list, get
task list, last, get, run, stop, logs
api
info
ping
```

Target workflow:

```bash
semctl auth login https://semaphore.example.com
semctl project list
semctl project use infra
semctl template list
semctl task run deploy-prod --message "Deploy release 1.8.3"
semctl task logs --follow
```

CI workflow:

```bash
export SEMAPHORE_HOST=https://semaphore.example.com
export SEMAPHORE_TOKEN="$SEMAPHORE_API_TOKEN"
export SEMAPHORE_PROJECT=infra
TASK_ID="$(semctl task run deploy-prod --json | jq -r '.id')"
semctl task logs "$TASK_ID" --follow
semctl task watch "$TASK_ID" --exit-code
```

## Command groups

Long-term command tree:

```text
semctl
  auth
  config
  api
  project
  user
  key
  repo
  inventory
  environment
  template
  task
  schedule
  runner
  integration
  event
  info
  alias
```

Use `repo` rather than `repository`. Use `environment` as canonical and optionally add `env` later.

## API escape hatch

Implement:

```bash
semctl api <METHOD> <PATH>
```

Examples:

```bash
semctl api GET /info
semctl api GET /projects
semctl api POST /project/1/tasks --field template_id=7
semctl api GET /project/1/tasks/lastsemctl api GET /projects
semctl api POST /project/1/tasks --field template_id=7
semctl api GET /project/1/tasks/last --jq '.[0]'
```

The path is relative to `/api`.

Support `--field`, `--raw-field`, `--input`, `--header`, and `--jq`.

`--header` flags must be forwarded through `api.Client.DoWithHeaders` so custom headers (e.g., `X-CSRF-Token`, request signatures) reach the actual HTTP request.

## Resource resolution

Accept numeric IDs everywhere.

For common resources, also resolve names:

```bash
semctl project get infra
semctl template get deploy-prod
semctl task run deploy-prod
```

Resolution order:

1. Numeric ID
2. Exact name
3. Case-insensitive exact name
4. Unique prefix
5. Fail with candidates

If ambiguous, show matches and ask for an ID.

## Output

Default to tables when stdout is a TTY.

Support:

```bash
semctl project list
semctl project list --json
semctl project list --output yaml
semctl task get 812 --jq '.status'
```

JSON output is a compatibility contract. Do not change JSON field names for cosmetic reasons.

## Error handling

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

Map HTTP responses:

```text
400 invalid request
401 not authenticated, suggest semctl auth login
403 not authorized
404 resource not found
409 conflict
500 server error
```

## Task UX

Tasks are the core workflow.

Required commands:

```bash
semctl task list
semctl task last
semctl task get <TASK>
semctl task run <TEMPLATE>
semctl task stop <TASK>
semctl task logs [TASK]
```

`semctl task logs` without a task ID may use the latest task in the active project.

`semctl template run <TEMPLATE>` may alias to `semctl task run <TEMPLATE>`.

Support `--follow`, `--tail`, `--raw`, `--interval`, `--watch`, `--exit-code`, and `--escape-sanitize`. Sanitize ANSI escape sequences from task output by default when writing to a TTY to prevent malicious playbooks from hiding text, altering terminal titles, or injecting hyperlinks.

Suggested exit codes:

```text
0 task succeeded
1 task failed
2 task stopped or canceled
3 timeout
4 CLI or API error
```

## Security

Never print:

* API tokens
* Passwords
* SSH private keys
* Passphrases
* Runner registration tokens after initial display
* Secret environment values

Redact in debug logs:

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

TLS verification is enabled by default. Only allow insecure TLS through `--insecure-skip-tls-verify` and warn when used.

Validate all external input at trust boundaries: host URLs must be absolute with `https://` or `http://` scheme. Reject bare hostnames.

Protect credential storage:

* Prefer OS keyring. Fail hard on keyring failure unless `--plaintext` is passed.
* Config directory must not be world-writable (`mode & 0002 == 0`); refuse to write secrets otherwise.
* Config file written atomically (`tmp` → `rename`) with `0600` permissions.
* Suppress terminal echo when reading tokens interactively.

Sanitize error disclosure:

* API error bodies must be truncated before display (e.g., 200 chars max). Full body available via `BodyString()` for debug.
* Do not echo raw server error pages, SQL fragments, or stack traces to the terminal.

Sanitize output surfaces:

* Strip ANSI escape sequences (CSI, OSC, and other control codes) from user-generated content before printing to a TTY. Provide `--escape-sanitize` toggle (default `true` in TTY mode).
* Forward `--header` flags from `semctl api` to the actual HTTP request. Do not parse them into a dummy request that is discarded.

Retry policy for resilience:

* Client-side retry with exponential backoff (e.g., 1s → 2s → 4s, max 3 retries).
* Retry on network errors and 5xx / 429 / 408. Do not retry on 4xx client errors.
* Drain response bodies between retries for connection reuse.
* Respect context cancellation during backoff.

## Testing

Use a layered testing strategy like mature CLI projects:

1. Unit tests for pure logic
2. Golden tests for CLI output
3. Integration tests against local Semaphore UI
4. End-to-end tests for critical workflows

The repository must provide a `compose.yaml` that starts Semaphore UI and all dependencies required for integration and E2E tests.

The compose stack is the canonical test target. Do not run tests against a developer's personal Semaphore UI instance by default.

Fast tests:

```bash
go test ./...
go test ./... -short
```

Integration and E2E tests require explicit opt-in:

```bash
SEMAPHORE_INTEGRATION=1 go test -tags=integration ./... -count=1
SEMAPHORE_E2E=1 go test -tags=e2e ./... -count=1
```

Tests must read target settings from env vars:

```text
SEMAPHORE_HOST
SEMAPHORE_TOKEN
SEMAPHORE_USERNAME
SEMAPHORE_PASSWORD
SEMAPHORE_PROJECT
```

Test requirements:

* Seed known test data into the compose Semaphore UI server.
* Use unique resource names for mutating tests.
* Clean up created resources.
* Avoid test order dependencies.
* Avoid sleeps when polling with deadlines or health checks works.
* Keep timeouts explicit.
* Never print test credentials or tokens.

Coverage areas:

* Config precedence
* Profile selection
* Auth token selection (env > keyring > profile)
* URL construction and host validation
* API error mapping and body truncation
* Resource resolution
* Secret redaction
* Stable JSON output
* Table golden files
* `semctl api` request construction and header forwarding
* Project listing
* Template listing
* Task creation
* Task log retrieval and ANSI sanitization
* Task stop behavior
* Watched task exit codes
* Client retry behavior (5xx/429 retry, 4xx no retry, backoff timing)
* Config directory world-writable rejection

For E2E tests, prefer black-box CLI execution:

```go
cmd := exec.Command("semctl", "project", "list", "--json")
```

Avoid internal package calls in E2E tests except for fixture setup.

Suggested fixtures:

```text
testdata/golden
testdata/fixtures
testdata/compose
```

CI should run formatting, linting, unit tests, golden tests, integration tests against `compose.yaml`, and E2E tests for the critical path.

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

## Acceptance criteria

MVP is acceptable when:

* A user can log in with an API token (interactive, no echo; or `--with-token` for pipes).
* A user can configure a default host and project.
* A user can list projects and templates.
* A user can run a template and follow task logs.
* A user can stop a task.
* CI can use env vars and JSON output.
* Unsupported endpoints work through `semctl api`.
* Secrets are redacted in normal and debug output.
* Token storage prefers OS keyring; plaintext fallback requires explicit `--plaintext`.
* Host URLs are validated as absolute `https://` or `http://`.
* API errors truncate response bodies; full body available via `BodyString()`.
* Client-side retry handles transient 5xx / 429 with exponential backoff.
* Task logs strip ANSI escape sequences by default in TTY mode.

# Repository Coding Instructions

This repository is mainly Go. Optimize for correctness, maintainability, operability, and idiomatic Go. Prefer small, boring, explicit code over clever abstractions.

## 1. Formatting, naming, and layout

1. Always run `gofmt` or `goimports` before committing. Do not hand-format Go code.
2. Keep imports clean and grouped by standard library, external dependencies, then internal packages when tooling supports it.
3. Use clear package names: short, lowercase, no underscores, no generic names like `common`, `util`, or `helpers` unless unavoidable.
4. Do not repeat the package name in exported identifiers. Prefer `cache.New()` over `cache.NewCache()`.
5. Use MixedCaps or mixedCaps. Preserve common initialisms such as `ID`, `HTTP`, `URL`, `API`, and `TLS`.
6. Keep names proportional to scope: short names for tiny local scopes, descriptive names for wider scopes.
7. Prefer simple file organization by domain or responsibility. Do not create excessive package fragmentation.
8. Keep generated code clearly marked and separate from hand-written code where practical.
9. Avoid unnecessary line-length rules. Wrap only when readability improves.
10. Avoid import dot except in rare, justified test scenarios.

## 2. API and package design

11. Keep package APIs small. Export only what must be used by other packages.
12. Document every exported package, type, function, method, const, and var with proper Go doc comments.
13. Accept interfaces, return concrete types, unless there is a specific reason to return an interface.
14. Define interfaces at the consumer side, not the provider side, unless the abstraction is fundamental to the package.
15. Avoid premature abstraction. Add interfaces, generics, or factories only when they remove real duplication or enable real substitution.
16. Prefer constructors for types with invariants, required dependencies, or non-obvious defaults.
17. Keep zero values useful where reasonable. Avoid types that panic or behave dangerously when zero-valued.
18. Do not use global mutable state unless there is a strong reason. Prefer explicit dependencies.
19. Avoid `init()` for application logic. Use explicit initialization paths.
20. Keep public APIs backward-compatible unless the change is intentionally breaking and documented.

## 3. Error handling

21. Always check errors. Do not discard errors with `_` unless the reason is obvious and documented.
22. Do not use `panic` for normal control flow. Return errors.
23. Wrap errors with useful context using `%w` when callers may need `errors.Is` or `errors.As`.
24. Keep error strings lowercase and without trailing punctuation.
25. Prefer sentinel errors only when callers need stable comparison behavior.
26. Use typed errors only when callers need structured information.
27. Do not log and return the same error unless there is a clear ownership boundary. Usually return the error and let the boundary log it.
28. Avoid in-band errors such as returning `-1`, empty string, or nil to indicate failure when an error return is appropriate.
29. Do not expose internal implementation details, secrets, SQL fragments, stack traces, or credentials in user-facing errors.
30. Make failure modes testable.

## 4. Context, concurrency, and cancellation

31. Pass `context.Context` explicitly as the first parameter for request-scoped operations.
32. Do not store `context.Context` in structs except in rare cases with strong justification.
33. Respect cancellation and deadlines in I/O, RPC, database, Kubernetes, and long-running operations.
34. Every goroutine must have a clear lifetime, cancellation path, and error-handling strategy.
35. Prefer synchronous functions unless concurrency is required by the API or materially improves behavior.
36. Avoid goroutine leaks. Use `errgroup`, channels, contexts, or wait groups deliberately.
37. Protect shared mutable state with mutexes or channels. Do not mix synchronization models casually.
38. Keep channel ownership clear. The sender usually closes the channel.
39. Run race-sensitive code with `go test -race` where practical.
40. Avoid sleeps in tests and production synchronization. Prefer contexts, clocks, conditions, or explicit signals.

## 5. Testing

41. Use table-driven tests for related input/output cases.
42. Test behavior, not implementation details.
43. Keep tests deterministic. No real network, wall-clock, external API, or filesystem dependency unless the test is explicitly integration-level.
44. Use `t.Helper()` in test helper functions.
45. Make test failures useful: include input, expected value, actual value, and relevant context.
46. Add regression tests for every bug fix.
47. Use fuzz tests for parsers, decoders, validators, protocol handling, and security-sensitive input processing.
48. Use benchmarks only for performance-sensitive code and avoid optimizing without evidence.
49. Prefer standard `testing` unless a helper library clearly improves readability.
50. Keep unit tests fast enough to run locally before pushing.

## 6. Security and data handling

51. Validate all external input at trust boundaries.
52. Do not use `math/rand` for secrets, tokens, keys, salts, or security-sensitive randomness. Use `crypto/rand`.
53. Avoid `unsafe`. Any use of `unsafe` requires a comment explaining why safe Go is insufficient and what invariant makes it safe.
54. Do not log secrets, tokens, passwords, private keys, session IDs, kubeconfigs, or credentials.
55. Use structured logging with stable keys. Avoid dumping full objects that may contain sensitive fields.
56. Use least privilege for filesystem, network, cloud, Kubernetes, and database access.
57. Prefer secure defaults: TLS verification on, authentication on, explicit allow-lists, closed-by-default behavior.
58. Do not build SQL, shell commands, YAML, JSON, or templates via unsafe string concatenation when structured APIs exist.
59. Treat dependency updates as code changes: review changelogs, run tests, and scan vulnerabilities.
60. Run `govulncheck ./...` in CI or release pipelines.

## 7. Dependencies and modules

61. Keep `go.mod` and `go.sum` committed.
62. Run `go mod tidy` after dependency changes.
63. Add dependencies only when they are justified by maintenance, correctness, security, or substantial complexity reduction.
64. Prefer the standard library when it is sufficient.
65. Avoid large framework dependencies for small problems.
66. Pin meaningful versions. Do not casually depend on `@latest` in repeatable builds.
67. Avoid long-lived `replace` directives in committed `go.mod` unless they are intentional and documented.
68. Keep module paths stable and controlled by the organization or repository.
69. Remove unused dependencies promptly.
70. Review transitive dependency risk for security-sensitive components.

## 8. Maintainability and complexity

71. Keep functions small and cohesive. Split code when branching, nesting, or cognitive load becomes hard to review.
72. Prefer early returns to deeply nested control flow.
73. Avoid cleverness. Code should be obvious to a competent Go developer under operational pressure.
74. Prefer explicit control flow over reflection, magic tags, or hidden registration.
75. Use generics only when they materially improve type safety or remove real duplication.
76. Avoid package-level side effects.
77. Keep configuration explicit, typed, validated, and documented.
78. Separate business logic from transport, storage, CLI, and framework glue.
79. Avoid circular dependencies by fixing package boundaries, not by adding abstraction hacks.
80. Delete dead code instead of preserving speculative future hooks.

## 9. Observability and operations

81. Log at service boundaries, retries, degraded behavior, and unexpected failures.
82. Do not log inside tight loops or high-cardinality paths without rate limiting or sampling.
83. Include enough context in logs to debug distributed systems, but never include secrets.
84. Emit metrics for critical paths, errors, latency, queue depth, retries, and external dependencies.
85. Use consistent metric names and labels. Avoid unbounded label cardinality.
86. Make timeouts, retry policies, and backoff explicit.
87. Use health checks that reflect actual readiness and dependency state.
88. Ensure graceful shutdown handles context cancellation, in-flight work, and cleanup.
89. Make operational defaults safe for Kubernetes/container environments.
90. Prefer reproducible builds and deterministic CI.

## 10. Review and CI requirements

91. CI must run formatting checks, `go test ./...`, `go vet ./...`, linting, and vulnerability scanning.
92. Use `golangci-lint` with a focused set of linters. Prefer signal over noise.
93. Recommended baseline linters: `govet`, `staticcheck`, `ineffassign`, `errcheck`, `gosec`, `goimports`, `revive`, and complexity checks.
94. Do not silence lint warnings without a local explanation.
95. Keep pull requests small enough to review properly.
96. Every non-trivial change should include tests or a written reason why tests are not practical.
97. Review for correctness first, then API design, security, concurrency, observability, and style.
98. Prefer simple migrations over large rewrites.
99. Document architectural decisions that affect future maintainers.
100. When changing behavior, update docs, examples, configuration, and release notes as needed.

## Assistant-specific instructions

When generating or modifying code in this repository:

- Follow the rules above unless existing local code clearly establishes a different convention.
- Prefer minimal, reviewable diffs.
- Do not introduce new dependencies without explaining why the standard library or existing dependencies are insufficient.
- Do not create broad abstractions from a single use case.
- Include tests for new behavior and bug fixes.
- Preserve public API compatibility unless explicitly asked to make a breaking change.
- For Go code, return errors instead of panicking, pass context explicitly, and keep goroutine lifetimes bounded.
- For security-sensitive changes, call out input validation, secret handling, dependency risk, and logging impact.
- Before finalizing, ensure the change would pass: `goimports`, `go test ./...`, `go vet ./...`, `golangci-lint run`, and `govulncheck ./...`.

Security-specific patterns established in this codebase:

- **Token input:** Interactive prompts use `golang.org/x/term.ReadPassword` (no echo). Non-TTY falls back to `bufio.Reader.ReadString`.
- **Token storage:** Keyring is default. Plaintext config fallback requires explicit `--plaintext` opt-in; otherwise fail hard.
- **Host validation:** Always run `url.Parse` and require `u.IsAbs()` with `https` or `http` scheme. Reject bare hostnames before any request.
- **Config directory:** Before writing secrets, check `stat.Mode().Perm()&0002 != 0` and refuse if world-writable.
- **API error disclosure:** Truncate error bodies in `Error()` to ~200 chars. Expose full body only via a separate `BodyString()` method.
- **HTTP retry:** Use exponential backoff (1s → 2s → 4s, max 3 retries). Retry on network errors and 5xx/429/408. Do not retry 4xx. Drain response bodies between attempts.
- **ANSI sanitization:** Strip CSI, OSC, and control sequences from user-generated output before printing to TTY. Default `--escape-sanitize=true` in `task logs`. Use `regexp` covering `\x1b\[[0-9;]*[A-Za-z]`, `\x1b\]…\x07`, and related sequences.
- **Header forwarding:** When `semctl api` accepts `--header` flags, forward them through `DoWithHeaders(http.Header)` so they actually reach the wire. Do not build a dummy request.
- **Secret redaction:** Never include `Authorization`, `Cookie`, `token`, `password`, or `secret` values in error messages, debug logs, or test output.

## Development workflow

All feature and fix work must follow an issue-first, branch-and-merge workflow. Do not commit directly to `main`.

1. **Open an issue** describing the bug, feature, or improvement before starting work.
2. **Create a branch** from `main` with a conventional prefix:
   - `feat/<short-description>` — new features
   - `fix/<short-description>` — bug fixes
   - `docs/<short-description>` — documentation changes
   - `chore/<short-description>` — tooling, dependencies, CI
3. **Make changes** with clear, focused commits following Conventional Commits (`feat:`, `fix:`, `docs:`, `chore:`).
4. **Run checks locally** before pushing:
   ```bash
   go fmt ./...
   golangci-lint run ./...
   go test -race -count=1 ./...
   govulncheck ./...
   ```
5. **Push** the branch and open a **Pull Request** against `main`.
6. **Reference the issue** in the PR description (e.g., `Closes #123`).
7. **Ensure CI passes** before requesting review.
8. **Merge** only after maintainer approval.