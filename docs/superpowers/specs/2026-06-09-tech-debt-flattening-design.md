# Tech-debt flattening & TDD foundation — design

Date: 2026-06-09
Status: approved

## Problem

New features in `semctl` reintroduce bugs faster than they fix them. Root causes,
confirmed by inspection:

1. **Bugs slip past tests** — no coverage measurement or gate; a new command can ship
   with two shallow tests. Baseline coverage ranges 20%–93% per package.
2. **Writing tests is painful** — `internal/testutil` exists but only provides a
   server-side `MockServer`. Each command test re-rolls its own `newTestRoot` (11 copies)
   and swaps the global `os.Stdout` with a pipe (7 occurrences) — fragile and not
   parallel-safe — even though `BuildCmdContext` already routes output through
   `cmd.OutOrStdout()` (context.go:307). The friction suppresses TDD.
3. **Hard to change safely** — `task.go` is 539 LOC; large units make edits ripple.
4. **Inconsistent patterns** — every command differs slightly; no documented, enforced
   "add a command" path. Branch sprawl (many parallel agent branches diverging from a
   moving `main`) compounds it.

## Approach

Four levers, delivered as small, independently-green, sequential PRs off `main`.

- **PR 0 — Land `feature/deepen-repo-v2`.** Squash 9 commits of unmerged feature work
  (api/cli/output/resolver/template depth, ~1300 lines) into one clean Conventional-Commit
  PR off the current trunk; verify green; merge. Flattens the divergence first.
- **PR 1 — `testutil` `RunCommand` harness + remove dead packages.** Add
  `RunCommand(t, args...) (stdout, stderr, err)` building the real root command and
  injecting buffers via `SetOut`/`SetErr` (parallel-safe, no global swap), plus config/token
  isolation helpers, building on the existing `MockServer`. Delete empty
  `internal/commands/cmd` and `internal/commands/user`. Self-test the harness.
- **PR 2 — Surface coverage in CI.** Per-package coverage printed + profile uploaded as an
  artifact. Non-blocking.
- **PR 3 — Migrate command tests to the harness.** Delete the duplicated `newTestRoot` and
  `os.Stdout` swaps; raise coverage on weak packages (`cli` 36%, `resolver` 47%,
  `config` 54%, `task` 57%) as a side effect.
- **PR 4 — Split `task.go`** into focused files (wiring + `run`/`logs`/`list`/…).
  Behavior-identical, test-guarded.
- **PR 5 — `CONTRIBUTING.md`** "adding a command" checklist anchored to the harness +
  `.github/pull_request_template.md`. Commits this design doc.
- **PR 6 — Ratcheting coverage floor.** A script fails CI if total coverage drops below the
  committed baseline. Ratchet from reality; no arbitrary target.

## Non-goals (YAGNI)

- No mock-generation framework — httptest fakes fit; the harness makes them ergonomic.
- No rewrite of the command/cli/api layering — it is sound.
- No arbitrary high coverage mandate — ratchet from the real baseline.

## Success criteria

- A new command's happy-path test is a few lines via `testutil.RunCommand` + `MockServer`.
- CI shows coverage and refuses regressions below baseline.
- No file in `internal/commands` exceeds ~300 LOC without cause.
- One documented, enforced way to add a command.
