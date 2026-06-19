# Changelog

All notable changes to this project are documented here.

## [Unreleased]

No unreleased changes yet.

## [0.3.0] - 2026-06-19

### Added

- `vidtrace analyze` for Markdown evidence reports from a ticket and artifact bundle.
- `vidtrace compare` for heuristic ticket-vs-video comparison, including JSON output.
- `vidtrace studio <bundle>` timeline/OCR/transcript browser.
- Glyphrun coverage for the interactive studio.
- MIT license.

## [0.2.0] - 2026-06-18

### Added

- `vidtrace docs` built-in product and agent documentation.
- `vidtrace studio` command name.
- Reusable agent prompt in `prompts/analyze-bundle.md`.

### Changed

- Renamed the public `tui` command to `studio`.

## [0.1.1] - 2026-06-18

### Fixed

- Added Homebrew cask quarantine cleanup guidance and hook.

## [0.1.0] - 2026-06-18

### Added

- Initial Go CLI with `doctor`, `extract`, glyphrun specs, CI, GoReleaser, and Homebrew cask publishing.
