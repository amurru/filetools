# Roadmap

This document tracks planned and candidate features for filetools. Items are
grouped by priority and effort. Status markers: `[ ]` not started, `[~]` in
progress, `[x]` done.

## Near term (high value, low effort)

- [ ] `search` command: full-text search across a tree (literal/regex, globs,
      context lines, binary detection, match counts). Already listed as planned.
- [ ] CSV output formatter (`internal/output/csv.go`).
- [ ] Markdown output formatter.
- [ ] Config file (`~/.config/filetools/config.yaml` or env) for default
      exclusions, hash algorithm and output format.
- [ ] `.gitignore` / `.filetoolsignore` aware exclusions during walks.
- [ ] Shell completions and man pages via Cobra generators.
- [ ] Progress ETA in the text reporter.

## Medium term

- [ ] dupfind actions: `--delete`, `--move-to <dir>`, `--hardlink` to reclaim
      space, with confirmation prompt and an undo journal.
- [ ] `empty` command: find empty files and directories, optional deletion.
- [ ] `stale` command: files not modified in N days, combinable with size
      thresholds.
- [ ] `brokenlinks` command: detect broken symlinks.
- [ ] `verify` command: generate and check SHA256SUMS-style manifests.
- [ ] `diff` command: compare two directories (added/removed/changed).
- [ ] `owner` command: disk usage aggregated by file owner (Unix; gated on
      Windows).
- [ ] Undo journal for rename/organize with `filetools undo <journal>`.
- [ ] Interactive confirmation for destructive CLI operations.
- [ ] Trash instead of permanent delete (OS trash integration).
- [ ] Organize by MIME type via magic bytes; recursive mode; collision-safe
      naming.
- [ ] Rename template tokens (`{date}`, `{counter}`, `{hash}`) and case
      transforms.
- [ ] Size/dirstat breakdowns by extension and directory; modification-time
      histograms.

## Longer term

- [ ] Archive support: scan inside `.zip`/`.tar`/`.tar.gz` (read-only).
- [ ] Watch mode (`--watch`) re-running on filesystem changes.
- [ ] GUI: drag-and-drop directories, recent directories, multi-select rows,
      export button, charts for dirstat/size.
- [ ] Dark/light toggle exposed in GUI settings.
- [ ] Parallel walk/hashing worker pool with `--concurrency`; benchmarks.
- [ ] Streaming output for very large trees.
- [ ] JSON Schema for report structs.
- [ ] Memory cap / max-files guard.

## Notes

- Every new command follows the documented checklist: scan engine, report
  struct, four (soon six) formatters, Cobra command, GUI page, tests.
- Destructive operations must stay dry-run by default and require an explicit
  confirmation or `--force`.
- The dependency direction stays strictly downwards; nothing in a lower layer
  may import a higher one.