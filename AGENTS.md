# Agent Guidelines for filetools

## Project Overview

Go toolkit for file management and analysis. Cobra based, five commands, four
output formats and an optional graphical interface built with Gio.

| Command    | Purpose                                                      |
| ---------- | ------------------------------------------------------------ |
| `dupfind`  | Find duplicate files by hash                                 |
| `dirstat`  | Directory statistics: file counts, sizes, per-type breakdown |
| `rename`   | Batch rename using glob matching and sed-style replacements  |
| `organize` | Move files into per-category subdirectories                   |
| `size`     | Size analysis: totals, size distribution, largest files       |

## Build/Test Commands

- **Build**: `make build` (injects version, commit, commitDate, treeState via ldflags)
- **Build with GUI**: `make build-gui` (adds the `gui` build tag, see below)
- **Test**: `go test ./...`
- **Test with race**: `go test -race ./...`
- **Coverage**: `go test -cover ./...`
- **Single test**: `go test -run TestName ./internal/scan/`
- **Vet**: `go vet ./...`
- **Format check**: `gofmt -l .`
- **Clean**: `make clean`
- **Run**: `make run`

CI (`.github/workflows/ci.yml`) runs gofmt, vet, build and `go test -race` on
every push and pull request. Releases are built by the SLSA workflow in
`.github/workflows/go-ossf-slsa3-publish.yml`, do not change it without
understanding the SLSA requirements.

## Architecture

The dependency direction is strictly downwards. Nothing in a lower layer may
import a higher one.

```
cmd/                 # Cobra front end: flags, metadata, formatting. No analysis logic.
internal/gui/        # Gio front end. Build tag "gui".
internal/scan/       # Analysis engines. Pure Go, no Cobra, no Gio.
internal/output/     # Formatters that render a report as text/json/xml/html.
internal/report/     # Data model shared by the engines, the formatters and the GUI.
internal/progress/   # Progress reporting contract and the text reporter.
internal/exclusions/ # Glob and file-type exclusion matchers.
internal/version/    # Build metadata injected via ldflags.
```

`cmd` and `internal/gui` are two interchangeable front ends over the same
`internal/scan` engines. A change to the behaviour of a command must be made
in `internal/scan`, never in a front end.

### Adding a command

1. Add the engine to `internal/scan/<name>.go` with an options struct, a
   `progress.Indicator` parameter and a `report.*Result` return value
2. Define the result struct in `internal/report/report.go`
3. Add `Format<Name>(result *report.<Name>Result, writer io.Writer) error` to
   the `OutputFormatter` interface
4. Implement it in all four formatters (`text.go`, `json.go`, `xml.go`, `html.go`)
5. Register the Cobra command in `cmd/<name>.go` with `rootCmd.AddCommand()`
6. Add the GUI page in `internal/gui/page_<name>.go`
7. Add tests in `internal/scan/<name>_test.go`

### Global flags

`-o/--output` (text, json, xml, html), `-f/--file`, the `-j`, `-x` and `-w`
shortcuts, `--exclude-file`, `--exclude-dir` and `--no-progress`. The
shortcut flags take priority over `-o`.

### Conventions

- **Output**: every command builds a result struct carrying a `Metadata`
  block, then hands it to `output.NewFormatter(format)`.
- **Progress**: use `newProgressIndicator(cmd)` in `cmd/root.go`. It returns a
  no-op indicator for non-TTY output, structured formats and file output.
  The engines only ever see the `progress.Indicator` interface.
- **Exclusions**: the engines parse the patterns themselves with
  `exclusions.ParseExclusions`, check paths during the walk with
  `exclusions.CheckExclusions` and report every skipped item in the result.
- **Traversal**: use `filepath.WalkDir`, not `filepath.Walk`. Call `d.Info()`
  only when the size or modification time is actually needed.
- **Dry run**: `rename` and `organize` run as a dry run by default, `--force`
  performs the change. The engine takes an explicit `DryRun` option.
- **Report paths**: source paths are relative to the scanned directory.
- **Options**: engines take a plain options struct. Do not pass Cobra flags or
  command line strings into `internal/scan`.

### The GUI

The graphical interface lives in `internal/gui` and is guarded by the `gui`
build tag, so the default build stays pure Go and cross compiles everywhere.

- `go build ./...` never compiles the GUI and never pulls in Gio.
- `go build -tags gui ./...` compiles it. On Linux this needs cgo and the
  X11, Wayland and EGL development packages. On Windows it is pure Go.
- `cmd/gui.go` (tag `gui`) registers the `gui` subcommand, `cmd/gui_stub.go`
  (tag `!gui`) explains how to enable it.
- Gio must stay pinned to an exact version. Never use `go get -u gioui.org`,
  it can pull breaking minor versions of unstable dependencies.

Gio rules worth remembering:

- A `widget.List` records its scroll position internally, so it must be
  created once and kept across frames. The zero value of `layout.Axis` is
  `Horizontal`, so set `Axis: layout.Vertical` explicitly.
- Gio reports the size of a widget by the size of its content. A widget that
  must fill a fixed column has to enforce its width, see `fixedWidth` in
  `internal/gui/widgets.go`.
- Panels that hug their content measure the content first with `op.Record`,
  paint the background and replay the content with `op.Defer`.
- Long running work runs in a goroutine and reports through
  `progress.Indicator`. Never block the frame loop and never touch widget
  state from a goroutine.
- The interface is tested by rendering it offscreen with
  `gioui.org/gpu/headless`. Write a screenshot with
  `FILETOOLS_GUI_SCREENSHOT=/tmp/gui.png go test -tags gui -run Screenshot ./internal/gui/`.

## Code Style Guidelines

### Imports
- Standard library imports first
- Third-party imports second (e.g., `github.com/spf13/cobra`)
- Group imports logically with blank lines

### Formatting
- Use `gofmt` for consistent formatting
- Run `gofmt -l .` to check formatting

### Naming Conventions
- **Functions**: camelCase (e.g., `calculateHash`, `findDuplicates`)
- **Variables**: camelCase (e.g., `hashAlgorithm`, `rootDir`)
- **Constants**: camelCase
- **Structs/Types**: PascalCase for exported types

### Error Handling
- Return errors from functions rather than panicking
- Use `fmt.Fprintf(os.Stderr, ...)` for warnings/errors
- Use `os.Exit(1)` for fatal errors in CLI commands
- Handle errors immediately after function calls

### Code Structure
- Use Cobra for CLI commands
- Main package delegates to `cmd` package
- Functions should be focused and testable
- Add comments to exported functions explaining their purpose

## Git Conventions

- Conventional commit prefixes: `feat:`, `fix:`, `perf:`, `test:`, `docs:`,
  `chore:`
- Do not add `Co-authored-by` trailers
- Keep commits focused, one logical change each
