# Contributing to filetools

Thank you for your interest in contributing. This document describes how to
get started, the project conventions and the pull request process.

## Getting started

1. Fork the repository and clone your fork.
2. Install Go 1.24.5 or later.
3. Build and test:
   ```sh
   make build
   make test
   ```
4. For the graphical interface build, install the platform dependencies listed
   in the README and run `make build-gui`.

## Project layout

The dependency direction is strictly downwards. Nothing in a lower layer may
import a higher one:

- `cmd/` and `internal/gui/` are front ends.
- `internal/scan/` holds the analysis engines.
- `internal/output/` renders reports.
- `internal/report/` is the shared data model.
- `internal/progress/`, `internal/exclusions/`, `internal/version/` are
  supporting packages.

A behaviour change must be made in `internal/scan`, never in a front end.

## Adding a command

Follow the checklist in `AGENTS.md` / README:

1. Add the engine in `internal/scan/<name>.go`.
2. Define the result struct in `internal/report/report.go`.
3. Add `Format<Name>` to the `OutputFormatter` interface and implement it in
   all formatters.
4. Register the Cobra command in `cmd/<name>.go`.
5. Add the GUI page in `internal/gui/page_<name>.go`.
6. Add tests in `internal/scan/<name>_test.go`.

## Code style

- Run `gofmt` (and `gofmt -l .` must be empty) before committing.
- Run `go vet ./...` and `golangci-lint run`.
- Keep commits focused, one logical change each.
- Use conventional commit prefixes: `feat:`, `fix:`, `perf:`, `test:`,
  `docs:`, `chore:`.
- Do not add `Co-authored-by` trailers.

## Tests

- Add tests for new functionality and bug fixes.
- Run the full suite with `go test -race ./...` before opening a PR.
- GUI tests need the EGL software renderer; see the README.

## Pull requests

- Open an issue or discussion first for large changes.
- Keep PRs small and reviewable.
- Update `README.md` and `ROADMAP.md` when behaviour changes.
- Ensure CI is green.