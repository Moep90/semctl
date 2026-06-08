# semctl — Semaphore UI CLI

[![CI](https://github.com/moep90/semaphore-cli/actions/workflows/ci.yaml/badge.svg)](https://github.com/moep90/semaphore-cli/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/moep90/semaphore-cli)](https://goreportcard.com/report/github.com/moep90/semaphore-cli)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Release](https://img.shields.io/github/v/release/moep90/semaphore-cli)](https://github.com/moep90/semaphore-cli/releases)

A command line interface for [Semaphore UI](https://semaphoreui.com). It feels similar to `gh` and `glab`: noun-first commands, readable tables by default, stable JSON for automation, explicit auth, profiles, and an `api` escape hatch.

> **Disclaimer:** `semctl` is an independent, open-source command line interface for Semaphore UI. It is **not affiliated with, endorsed by, sponsored by, or officially connected to the Semaphore UI project or its creators**. This tool is intended for personal use, educational purposes, and operational convenience at your own risk. All product names, logos, and brands are property of their respective owners.

## Installation

### Homebrew (macOS and Linux)

```bash
brew install moep90/tap/semctl
```

Or:

```bash
brew tap moep90/tap
brew install semctl
```

### From source

```bash
go install github.com/moep90/semaphore-cli/cmd/semctl@latest
```

### Pre-built binaries

Download from [GitHub Releases](https://github.com/moep90/semaphore-cli/releases).

## Quick start

```bash
# Authenticate (API token from your Semaphore UI profile)
semctl auth login https://semaphore.example.com

# Select a default project
semctl project list
semctl project use homelab

# Discover templates and run tasks
semctl template list
semctl task run deploy-prod --message "Deploy release 1.8.3"
semctl task logs --follow
```

## Authentication

```bash
# Interactive login (prompts for token, no echo)
semctl auth login https://semaphore.example.com

# Login with token from stdin
pass show semaphore/prod-token | semctl auth login https://semaphore.example.com --with-token

# One-off command with token from environment
SEMAPHORE_TOKEN=... semctl project list --host https://semaphore.example.com
```

Credentials are stored in your OS keychain when possible. If the keychain is unavailable, the CLI fails unless you pass `--plaintext`, which stores the token in your config file. See `semctl auth login --help`.

### Token precedence

When a command needs to authenticate, the token is resolved in this order:

1. **`SEMAPHORE_TOKEN`** environment variable
2. **OS keyring** (macOS Keychain, Windows Credential Manager, Linux Secret Service)
3. **Active profile** `token` field in `config.yml`

This means `SEMAPHORE_TOKEN` overrides everything, making it ideal for CI pipelines. For daily use, the keyring is preferred because it keeps the token out of shell history and environment dumps (`/proc/*/environ`).

**Security tip:** Avoid `export SEMAPHORE_TOKEN=...` in your shell profile. Use `--with-token` for one-off scripts, or run `semctl auth login` once to store the token in the keyring.

## Configuration

Platform-native config paths:

- **Linux**: `~/.config/semctl/config.yml`
- **macOS**: `~/Library/Application Support/semctl/config.yml`
- **Windows**: `%AppData%\semctl\config.yml`

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

## Environment variables

```text
SEMAPHORE_HOST
SEMAPHORE_TOKEN
SEMAPHORE_PROFILE
SEMAPHORE_PROJECT
SEMAPHORE_OUTPUT
SEMAPHORE_NO_COLOR
```

Precedence: explicit flags > environment variables > active profile > defaults.

## Output formats

```bash
semctl project list
semctl project list --json
semctl project list --output yaml
semctl task get 812 --output json | jq '.status'
```

## Task workflow

```bash
# Run a template
semctl task run deploy-prod --message "Deploy release 1.8.3"

# Watch a task and return its exit code
semctl task run deploy-prod --watch --exit-code

# Follow logs
semctl task logs 812 --follow

# Stop a task
semctl task stop 812
```

## API escape hatch

```bash
semctl api GET /info
semctl api GET /projects
semctl api POST /project/1/tasks --field template_id=7
semctl api GET /project/1/tasks/last | jq '.[0]'
```

## Security

- Tokens and cookies are never printed.
- `--debug` redacts secrets from logs.
- TLS verification is enabled by default.

## Shell completion

```bash
semctl completion bash > /etc/bash_completion.d/semctlctl
semctl completion zsh > "${fpath[1]}/_sem"
semctl completion fish > ~/.config/fish/completions/semctl.fish
```

## Development

Requires [mise](https://mise.jdx.dev) (optional but recommended).

```bash
mise install
mise run build
mise run test
mise run test-e2e
```

Without `mise`:

```bash
go build -o bin/semctl ./cmd/semctl
go test ./...
```

### Integration and E2E tests

Tests run against a disposable Semaphore UI instance via Docker Compose:

```bash
docker compose up -d --wait
SEMAPHORE_E2E=1 go test -tags=e2e ./... -count=1
docker compose down -v
```

## License

Licensed under the [Apache License, Version 2.0](LICENSE).
