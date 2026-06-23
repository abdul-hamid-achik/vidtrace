# Changelog

All notable changes to this project are documented here.

## [Unreleased]

## [0.10.0] - 2026-06-23

### Added

- fcheap and vecgrep integration for bundle stashing and codebase search. `vidtrace stash save|list|restore|info|search` wraps the fcheap CLI for vault persistence, sharing, and cross-stash evidence search. `vidtrace investigate --connect --codebase` runs `fcheap connect` (vecgrep) and returns real `file:line` code matches alongside video evidence; `vidtrace investigate --stash <id>` restores a stashed bundle before investigation. All new features degrade gracefully when fcheap or vecgrep are not installed. The MCP server exposes read-only `stash_list`, `stash_info`, `stash_search`, and `stash_connect` tools and enhances `investigate` with `connect`, `stash_id`, `connect_mode`, and `connect_limit` fields. `vidtrace doctor` reports `fcheap` and `vecgrep` as optional tools. See ADR-0005.

## [0.9.0] - 2026-06-21

### Removed

- `scripts/extract.sh` after Go pipeline parity was verified on synthetic and real video (same frames, OCR, and transcript outputs, plus Go-only `metadata.json` and `timeline.json`). `internal/pipeline` is the sole extractor. The parity check ran on a 3-frame synthetic clip (3 frames, 3 OCR files, 5 transcript formats, `validate` 9/9 checks passed) and on `~/Downloads/bug.mp4` (94 frames, 94 OCR files, 5 transcript formats, `validate` 9/9 checks passed).

### Added

- Artifact schema version is now a single source of truth (`artifacts.SchemaVersion`), referenced by the pipeline, timeline builder, and validator so the expected version can never drift across packages.
- `vidtrace validate --json` now emits a `warnings` array for soft issues that do not fail validation: an empty `transcript/` directory when metadata declares a whisper model (silent video or transcription failure), and a frame count that differs from the OCR frame txt count (partial extraction or manual edit).
- Bundle path collision handling: two extractions in the same second now produce distinct directories (`_2`, `_3`, ...) instead of silently overwriting each other.
- `vidtrace extract` now derives its context from SIGINT/SIGTERM so long ffmpeg/whisper runs can be interrupted cleanly; `--json` failure output stays stable.
- Pipeline concurrency: OCR frames now run in parallel with a bounded worker pool, and Whisper transcription runs concurrently with OCR. New `--concurrency` flag caps OCR workers (0 = auto, capped to 8). On a 94-frame real video this cut wall-clock time by ~40%. The `progressReporter` is now mutex-protected and safe for concurrent use.
- CI media smoke job: a new GitHub Actions `mediasmoke` job generates a tiny synthetic video and runs an end-to-end Go extraction (ffmpeg → tesseract → whisper) on `main` pushes and `workflow_dispatch`, catching regressions in the media-tool wrappers that unit tests cannot reach. Pull requests stay fast (the job is skipped on PRs).
- `vidtrace migrate-evidence <db>` converts pre-v0.17.0 evidence databases (the three-collection layout: `evidence_entries_keyword`, `evidence_entries_text`, `evidence_meta`) into the single `evidence_entries` collection with a named `text` vector space. Running it on a modern database is a no-op (`already_migrated: true`), so it is safe to run unconditionally.

### Changed

- Combined OCR file (`ocr_all_frames.txt`) header timestamp now uses the same injected UTC RFC3339 timestamp as `metadata.json`'s `generated_at`, instead of local time, for consistency and deterministic tests.
- Evidence layout collapsed to a single `evidence_entries` collection with a named `text` vector space (was three collections: `evidence_entries_keyword` + `evidence_entries_text` + `evidence_meta`). All three search modes (keyword, semantic, hybrid) now run against the one collection, eliminating content duplication and unifying the filter path. `vidtrace index --json` now reports `collection: "evidence_entries"` regardless of mode.
- Bumped `veclite` to `v0.17.0` (was `v0.16.0`) to adopt `UpsertRecordByKey` and `HybridSearchSpace`, which make the single-collection layout possible.

## [0.8.0] - 2026-06-20

### Added

- Linux `.deb` and `.rpm` packages (amd64 and arm64) are published with each release via nfpms and documented in `docs/INSTALL.md`. The Homebrew-cask-vs-formula decision and an Apple signing/notarization playbook are recorded in `docs/RELEASE.md`.

## [0.7.0] - 2026-06-20

### Added

- `vidtrace extract` fails fast when a requested `--ocr-lang` is not installed, listing the missing tesseract language packs before any frames are extracted, instead of failing partway through OCR.

### Changed

- Timeline transcript matching now tiles each frame to the next actual frame's time as a half-open interval (the last frame extends to the end of the recording). This handles fractional frame rates and missing frames, captures trailing audio on the last frame, and stops boundary segments from being double-counted. A segment that overlaps no interval falls back to the nearest frame by midpoint so no transcript is dropped. The `timeline.json` schema is unchanged.
- Human extraction progress now renders a live `bubbles` progress bar that redraws in place on an interactive terminal; piped, captured, or `--json` output stays plain one-line-per-step (no per-frame spam), so logs and agent callers are unaffected.
- `vidtrace studio` refuses to start when stdin/stdout is not an interactive terminal, returning a clear message that points automated callers to the `--json` commands or `vidtrace docs agent` instead of launching a TUI that would hang or garble their session.

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
