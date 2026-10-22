# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-10-23

### Added

- Graphical interface built with Gio, with preview before changes are applied.
- Pause and stop controls for long running GUI operations.
- `organize` and `size` commands.
- Per-operation apply failure reporting in the GUI.
- Duplicate file fingerprinting on the first 64 KB before full hashing.
- Size range filters for `dupfind`.
- Four output formats: text, JSON, XML, HTML, plus file output.
- File and directory exclusion patterns.

### Changed

- Analysis engines split out of the Cobra layer into `internal/scan`.
- Module path set to `github.com/amurru/filetools`.

### Fixed

- Navigation in the graphical interface.
- Bar chart track colour and corner radius in the GUI.
- Exclusion of the scanned root directory itself.