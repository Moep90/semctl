> **Disclaimer:** `semctl` is an independent, open-source command line interface for Semaphore UI. It is **not affiliated with, endorsed by, sponsored by, or officially connected to the Semaphore UI project or its creators**. This tool is intended for personal use, educational purposes, and operational convenience at your own risk. All product names, logos, and brands are property of their respective owners.

## Summary

Brief description of what this PR does and why.

> ⚠️ **Please make sure the PR title follows [Conventional Commits](https://www.conventionalcommits.org/)** (e.g. `feat: add runner list command`, `fix: resolve panic on nil pointer`). The title is used as the merge commit message and determines the next release version.

## Checklist

- [ ] I have read the [Contributing Guidelines](CONTRIBUTING.md).
- [ ] `go fmt ./...` passes.
- [ ] `golangci-lint run ./...` passes.
- [ ] `go test -race -count=1 ./...` passes.
- [ ] `govulncheck ./...` passes (or no new reachable vulnerabilities).
- [ ] I have added/updated tests for my changes.
- [ ] I have updated documentation (`README.md`, `docs/`) if needed.

## Type of Change

- [ ] Bug fix
- [ ] New feature
- [ ] Documentation update
- [ ] Refactor / chore
- [ ] Test improvement

## Additional Notes

Anything else reviewers should know?
