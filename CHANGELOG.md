> **Disclaimer:** `semctl` is an independent, open-source command line interface for Semaphore UI. It is **not affiliated with, endorsed by, sponsored by, or officially connected to the Semaphore UI project or its creators**. This tool is intended for personal use, educational purposes, and operational convenience at your own risk. All product names, logos, and brands are property of their respective owners.

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0](https://github.com/Moep90/semctl/compare/v0.2.1...v0.3.0) (2026-06-08)


### Features

* add cookie-based authentication ([#15](https://github.com/Moep90/semctl/issues/15)) ([e2dc149](https://github.com/Moep90/semctl/commit/e2dc149f54f003d444d59be220bb8c91c2a62651))
* add task run flags ([#16](https://github.com/Moep90/semctl/issues/16)) ([e2dc149](https://github.com/Moep90/semctl/commit/e2dc149f54f003d444d59be220bb8c91c2a62651))
* implement feature requests [#14](https://github.com/Moep90/semctl/issues/14), [#15](https://github.com/Moep90/semctl/issues/15), [#16](https://github.com/Moep90/semctl/issues/16) ([#19](https://github.com/Moep90/semctl/issues/19)) ([e2dc149](https://github.com/Moep90/semctl/commit/e2dc149f54f003d444d59be220bb8c91c2a62651))
* implement inventory, environment, keystore subcommands ([#14](https://github.com/Moep90/semctl/issues/14)) ([e2dc149](https://github.com/Moep90/semctl/commit/e2dc149f54f003d444d59be220bb8c91c2a62651))

## [0.2.1](https://github.com/Moep90/semctl/compare/v0.2.0...v0.2.1) (2026-06-08)


### Bug Fixes

* address bugs [#10](https://github.com/Moep90/semctl/issues/10)-13 — respect flags and output modes ([#17](https://github.com/Moep90/semctl/issues/17)) ([6866385](https://github.com/Moep90/semctl/commit/6866385321a323ecbdffcea78a3d840ba8bd6643))
* correct action-semantic-pull-request SHA to v5.5.3 ([eab98e8](https://github.com/Moep90/semctl/commit/eab98e82a2ce7c5fb6b3dad59dfc65e92a64de3e))
* remove skip-github-release so tags are created on merge ([#9](https://github.com/Moep90/semctl/issues/9)) ([96c6d36](https://github.com/Moep90/semctl/commit/96c6d3678c28ff6658fc27b77e89666ff5a1b5c3))

## [0.2.0](https://github.com/Moep90/semctl/compare/v0.1.0...v0.2.0) (2026-06-08)


### Features

* add automated release and PR title validation ([#7](https://github.com/Moep90/semctl/issues/7)) ([ae3bbbd](https://github.com/Moep90/semctl/commit/ae3bbbdbe40a11b777d33193e4f4bba7d1371ae0))


### Bug Fixes

* address 8 bugs from user review ([4d8f47b](https://github.com/Moep90/semctl/commit/4d8f47b0704b51727d016813f6fb6cc7bebbe75c))
* gofmt alignment in task.go stop command ([d4e11cb](https://github.com/Moep90/semctl/commit/d4e11cb028edded5b6217bbca2a6efd62f51eb02))

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
