> **Disclaimer:** `semctl` is an independent, open-source command line interface for Semaphore UI. It is **not affiliated with, endorsed by, sponsored by, or officially connected to the Semaphore UI project or its creators**. This tool is intended for personal use, educational purposes, and operational convenience at your own risk. All product names, logos, and brands are property of their respective owners.

# Contributing to semctl

Thank you for your interest in contributing! This document outlines the workflow and expectations for contributions.

## Getting Started

1. **Fork** the repository on GitHub.
2. **Clone** your fork locally.
3. **Create a branch** for your change:
   ```bash
   git checkout -b feat/my-feature
   ```
4. **Make your changes** with clear, focused commits.
5. **Run the checks** (see below).
6. **Push** and open a **Pull Request** against `main`.

## Development Environment

This project uses [mise](https://mise.jdx.dev) to manage tooling versions. If you have `mise` installed:

```bash
mise install
mise run build
mise run test
```

Without `mise`:

```bash
go build -o bin/semctl ./cmd/semctl
go test ./...
```

## Pre-Submission Checklist

Before opening a PR, please ensure the following pass locally:

```bash
# Formatting
go fmt ./...

# Linting
golangci-lint run ./...

# Unit tests
go test -race -count=1 ./...

# Vulnerability check
govulncheck ./...
```

## Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` new feature
- `fix:` bug fix
- `docs:` documentation only
- `test:` adding or correcting tests
- `refactor:` code change that neither fixes a bug nor adds a feature
- `chore:` tooling, dependencies, CI

Example:

```
feat: add runner list command

Adds the `semctl runner list` command with table and JSON output.
```

## Automated Releases

This project uses automated releases. After a PR is merged to `main`:

1. **Release Please** analyzes the conventional commits since the last release and opens a *Release PR* with the changelog and proposed version bump.
2. A maintainer reviews and merges the Release PR.
3. Merging the Release PR creates a new `v*` git tag, which triggers the existing `release.yaml` CI workflow.
4. **GoReleaser** builds binaries, generates SBOMs, signs artifacts with cosign, publishes `.deb`/`.rpm` packages, and updates the Homebrew tap.

**Important:** Pull request titles are validated by CI and must follow Conventional Commits. The title is used as the squash-merge commit message, which Release Please reads to determine the next version.

## Testing

- **Unit tests:** cover pure logic, config precedence, auth, output rendering, and error mapping.
- **Golden tests:** use fixtures under `testdata/golden/` for stable output formatting.
- **Integration / E2E tests:** run against a disposable Semaphore UI instance via Docker Compose. See `mise run test-e2e`.

Please add or update tests for any behavior change.

## Code Review

All submissions require review before merging. Maintainers will review PRs as time permits. Small, focused PRs are reviewed faster than large, sweeping changes.

## Security

If you discover a security vulnerability, please see [SECURITY.md](SECURITY.md) for responsible disclosure guidelines. Do **not** open public issues for security bugs.

## Code of Conduct

This project adheres to the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## Questions?

Feel free to open a [Discussion](https://github.com/moep90/semaphore-cli/discussions) for questions, ideas, or general support.
