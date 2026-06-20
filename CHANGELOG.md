# Changelog

All notable changes to this project are documented here.

## [Unreleased]

No unreleased changes yet.

## [0.6.0] - 2026-06-20

### Added

- `vidtrace search` filters: `--bundle`, `--source-video`, `--source`, `--min-time`, and `--max-time` narrow results so a multi-bundle evidence database can be searched by bundle, source video, evidence source, or timestamp window. JSON output echoes active filters under a `filters` object and omits it when no filter is set.
- `vidtrace index` accepts multiple bundle paths (for example a shell glob) and indexes them into one database, validating every bundle before any write and reporting per-bundle plus aggregate totals. Single-bundle output keeps the existing JSON shape.
- Semantic and hybrid evidence search via Ollama. `vidtrace index --embed ollama --embed-model <model>` builds a vector index alongside the keyword index, and `vidtrace search --mode semantic|hybrid` embeds the query to rank paraphrased descriptions. An embedding provider is pluggable behind an `Embedder` interface; the embedding profile is stored and a mismatched provider/model/dimension is rejected. Keyword stays the default and needs no provider. `vidtrace doctor` reports Ollama as an optional tool.
- `vidtrace mcp` runs a Model Context Protocol server (official Go MCP SDK) over stdio, exposing read-only `validate`, `search`, `compare`, `analyze`, and `investigate` tools whose structured outputs mirror the `--json` contracts. No tool mutates videos or generated bundles. See ADR-0004.

### Changed

- `vidtrace investigate` suggested code searches now drop browser/OS chrome, host and domain tokens, month and day names, and four-digit years from OCR text, so suggestions surface bug-relevant terms instead of address-bar and clock noise. Code-like tokens such as ticket IDs and the verbatim user query are preserved.
- Reorganized end-to-end specs from `specs/glyphrun/` into `e2e/` (`flows/` for specs, `fixtures/` for shared sample-bundle scripts, `actions/` for reusable step snippets). The duplicated inline bundle setup is replaced by `e2e/fixtures/sample_bundle.sh`, non-interactive flows share a `wait_clean_exit` action, and `task e2e` now globs `e2e/flows/*.yml`.

### Removed

- `PLAN.md` and `docs/HANDOFF.md` after the `v0.5.0` release; the release record lives in this changelog and `BACKLOG.md`, and durable extractor gotchas moved into `AGENTS.md`.

## [0.5.0] - 2026-06-20

### Added

- `PLAN.md` for the `v0.5.0` Studio dogfood and review workflow.
- Studio metadata/details toggle with the `m` key.
- Studio frame actions for opening selected frames, revealing them in Finder on macOS, and copying concise evidence summaries.
- `vidtrace index <bundle> --db <path>` for optional VecLite BM25 evidence indexing.
- `vidtrace search <db> <query> --json` for timestamped evidence lookup.
- `vidtrace investigate <bundle> --query <text> [--codebase <path>]` for video-evidence to code-search handoffs.
- VitePress documentation site configured for Vercel deployment.
- Step progress bars for human-readable extraction output.
- Unit tests for Studio metadata formatting, evidence summary formatting, frame path resolution, and platform command selection.
- Unit and CLI tests for evidence indexing and search.
- Unit and CLI tests for investigation handoff output.
- Glyphrun coverage for the VitePress documentation build.
- Glyphrun coverage for Studio metadata toggle and action status text.

### Changed

- README, Studio docs, usage docs, CLI contract, backlog, and agent docs now describe the Studio review actions.
- README, usage docs, CLI contract, backlog, roadmap, and built-in docs now describe the evidence search workflow.
- Documentation site notes now describe the VitePress/Vercel deployment path.
- Documentation site installation and build now use Bun with `bun.lock` instead of npm with `package-lock.json`.
- Documentation site build pins the working VitePress/Vite/plugin/esbuild toolchain for Vercel.
- Studio now uses a compact top-aligned layout with responsive timeline and evidence panes.
- VecLite is pinned to `v0.15.0`.

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
