# Changelog

All notable changes to this project are documented here.

## [Unreleased]

No unreleased changes yet.

## [0.4.0] - 2026-06-19

### Added

- `vidtrace validate <bundle> [--json]` for artifact bundle structure and path checks.
- Built-in `vidtrace docs studio` topic for terminal Studio review guidance.
- `confidence` and `term_hits` fields in `vidtrace compare --json`.
- Glyphrun coverage for compare and validate flows.
- Unit tests for validation, compare JSON shape, normalized term matching, invalid FPS, frame time calculation, and empty OCR representation.

### Changed

- `vidtrace compare` now normalizes punctuation-separated terms before scoring.
- README, agent docs, backlog, install, release, testing, and site docs now reflect the published install and Studio workflow.
- Documentation now describes bundle validation, compare limitations, and timeline frame time calculation.

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
