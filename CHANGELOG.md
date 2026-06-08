> **Disclaimer:** `semctl` is an independent, open-source command line interface for Semaphore UI. It is **not affiliated with, endorsed by, sponsored by, or officially connected to the Semaphore UI project or its creators**. This tool is intended for personal use, educational purposes, and operational convenience at your own risk. All product names, logos, and brands are property of their respective owners.

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Apache-2.0 license and `NOTICE` file.
- Governance documents: `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `CODEOWNERS`.
- GitHub automation: Dependabot config, issue templates, PR template.
- Expanded E2E test coverage (`auth logout`, `project use`, `task logs`, `task stop`, `api`, `ping`).
- CI E2E job that spins up Semaphore UI via Docker Compose.
- GoReleaser SBOM generation and `.deb`/`.rpm` packaging.
- Nightly snapshot workflow.
- Prominent unaffiliated disclaimer in `README.md` and `NOTICE`.

## [0.1.0] - 2025-01-15

### Added
- Initial MVP release of the Semaphore UI CLI (`semctl`).
- Commands: `auth login/logout/status`, `config profile list/use/set/get`, `project list/get/use`, `template list/get`, `task list/last/get/run/stop/logs`, `api`, `info`, `ping`.
- Multi-profile support with OS keyring integration.
- Output formats: table, JSON, YAML, text.
- Docker Compose test stack for integration and E2E testing.
- Golden output tests under `testdata/golden/`.

[unreleased]: https://github.com/moep90/semaphore-cli/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/moep90/semaphore-cli/releases/tag/v0.1.0
