# Agent Guidelines for filetools

## Project Overview

Go CLI toolkit for file management and analysis. Cobra based, five commands,
four output formats.

| Command    | Purpose                                                      |
| ---------- | ------------------------------------------------------------ |
| `dupfind`  | Find duplicate files by hash                                 |
| `dirstat`  | Directory statistics: file counts, sizes, per-type breakdown |
| `rename`   | Batch rename using glob matching and sed-style replacements  |
| `organize` | Move files into per-category subdirectories                   |
| `size`     | Size analysis: totals, size distribution, largest files       |

## Build/Test Commands

- **Build**: `make build` (injects version, commit, commitDate, treeState via ldflags)
- **Test**: `go test ./...`
- **Test with race**: `go test -race ./...`
- **Coverage**: `go test -cover ./...`
- **Single test**: `go test -run TestName ./cmd/`
- **Vet**: `go vet ./...`
- **Format check**: `gofmt -l .`
- **Clean**: `make clean`
- **Run**: `make run`

CI (`.github/workflows/ci.yml`) runs gofmt, vet, build and `go test -race` on
every push and pull request. Releases are built by the SLSA workflow in
`.github/workflows/go-ossf-slsa3-publish.yml`, do not change it without
understanding the SLSA requirements.

## Architecture

```
cmd/                 # One file per Cobra command, plus the shared root command
internal/output/     # OutputFormatter interface and the text/json/xml/html implementations
internal/exclusions/ # Glob and file-type exclusion matchers
```

### Adding a command

1. Create `cmd/<name>.go` with the Cobra command and its flags
2. Define the result struct in `internal/output/formatter.go`
3. Add `Format<Name>(result *<Name>Result, writer io.Writer) error` to the
   `OutputFormatter` interface
4. Implement it in all four formatters (`text.go`, `json.go`, `xml.go`, `html.go`)
5. Register the command in `init()` with `rootCmd.AddCommand()`
6. Add tests in `cmd/<name>_test.go`

### Global flags

`-o/--output` (text, json, xml, html), `-f/--file`, the `-j`, `-x` and `-w`
shortcuts, `--exclude-file`, `--exclude-dir` and `--no-progress`. The
shortcut flags take priority over `-o`.

### Conventions

- **Output**: every command builds a result struct carrying a `Metadata`
  block, then hands it to `output.NewFormatter(format)`.
- **Progress**: use `output.NewProgressIndicator(cmd)`. It returns a no-op
  indicator for non-TTY output, structured formats and file output.
- **Exclusions**: parse the patterns at command level with
  `exclusions.ParseExclusions`, check paths during the walk with
  `exclusions.CheckExclusions` and report every skipped item in the result.
- **Traversal**: use `filepath.WalkDir`, not `filepath.Walk`. Call `d.Info()`
  only when the size or modification time is actually needed.
- **Dry run**: `rename` and `organize` run as a dry run by default, `--force`
  performs the change.
- **Report paths**: source paths are relative to the scanned directory.

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
