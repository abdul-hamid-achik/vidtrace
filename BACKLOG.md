# Backlog

This backlog keeps product ideas, engineering work, and integration bets visible without forcing them into the current iteration.

## Recently Completed

### v0.10.0 Release

As a maintainer, I can ship the fcheap + vecgrep integration as a tagged release.

Acceptance criteria:

- [x] `CHANGELOG.md` has a `0.10.0` section and `README.md` reports `v0.10.0`.
- [x] Tag `v0.10.0` is pushed and drives the build version.
- [x] The GitHub release workflow publishes checksums and Darwin/Linux tarballs.
- [x] No source videos, generated bundles, or build output are tracked.

### VecLite v0.17.0 Single-Collection Migration

Acceptance criteria:

- [x] Bump `go.mod` to `github.com/abdul-hamid-achik/veclite v0.17.0`.
- [x] `vidtrace index` writes one record per timeline entry into `evidence_entries`, with a named `text` vector space added when `--embed` is configured.
- [x] All three search modes (`keyword`, `semantic`, `hybrid`) run against the single collection using `TextSearch` / `SearchSpace` / `HybridSearchSpace`.
- [x] Re-indexing is idempotent by `evidence_id` via `UpsertRecordByKey`.
- [x] `vidtrace migrate-evidence <db>` converts pre-v0.17.0 databases in place (no-op on modern DBs).
- [x] ADR-0003, CLI_CONTRACT, USAGE, and CHANGELOG updated.
- [x] `task all` green; real-video dogfood passes.

### Artifact Polish and Bash Extractor Removal

As a maintainer, I can remove the legacy Bash extractor and polish artifact consistency, so that `internal/pipeline` is the sole extractor and the schema version, timestamps, and validation warnings are consistent across packages.

Acceptance criteria:

- [x] Go pipeline parity verified on synthetic and real video; decision recorded in `CHANGELOG.md`.
- [x] `scripts/extract.sh` removed; `AGENTS.md` and `docs/ROADMAP.md` updated to reflect `internal/pipeline` as the sole extractor.
- [x] Schema version centralized in `internal/artifacts` and referenced by pipeline, timeline, and validate.
- [x] Combined OCR timestamp uses the same UTC injected timestamp as `metadata.json`.
- [x] `vidtrace validate` emits soft `warnings` for empty transcript with a declared whisper model and for frame/OCR count drift.
- [x] Bundle path collision handling appends `_2`, `_3`, ... suffixes.
- [x] `vidtrace extract` supports SIGINT/SIGTERM cancellation.
- [x] Unit tests cover the glob safety, timestamp consistency, empty-OCR combined file, collision handling, and validation warnings.

### v0.8.0 Release

As a maintainer, I can ship Linux `.deb`/`.rpm` packages and the documented distribution decisions as a tagged release.

Acceptance criteria:

- [x] `CHANGELOG.md` has a `0.8.0` section and `README.md` reports `v0.8.0`.
- [x] Tag `v0.8.0` is pushed and the release publishes Linux `.deb`/`.rpm` (amd64 + arm64) alongside the archives and checksums.
- [x] CI and the release workflow pass.

### v0.7.0 Release

As a maintainer, I can ship timeline matching v2, multi-language OCR fail-fast, the live progress bar, and the studio agent guard as a tagged release.

Acceptance criteria:

- [x] `CHANGELOG.md` has a `0.7.0` section and `README.md` reports `v0.7.0`.
- [x] Tag `v0.7.0` is pushed and drives the build version.
- [x] The GitHub release workflow publishes checksums and Darwin/Linux tarballs.
- [x] No source videos, generated bundles, databases, or build output are tracked.

### Multi-Language OCR

As a QA engineer testing localized apps, I want non-English OCR to work, so that UI text in other languages is captured correctly.

Acceptance criteria:

- [x] `--ocr-lang eng+spa` fails early when a requested language pack is not installed, naming the missing packs before any extraction work.
- [x] `vidtrace doctor` reports the available Tesseract languages.
- [x] Docs explain how to install Tesseract language data (`docs/INSTALL.md`).
- [x] Unit tests cover the language parsing and missing-language detection.

### TUI/Progress UX and Agent Safety

As a user, I want a clean live progress bar during extraction; as an agent, I want the TUI never to trap me.

Acceptance criteria:

- [x] Extraction renders a live `bubbles` progress bar that redraws in place on a TTY.
- [x] Piped/captured/`--json` output stays plain one-line-per-step (no per-frame spam).
- [x] `vidtrace studio` refuses non-interactive terminals with a message pointing to the JSON commands or `docs agent`.
- [x] Tests cover the plain/interactive progress reporter and the studio guard.

### Timeline Matching V2

As a coding agent, I want `timeline.json` to align transcript segments to frames beyond simple fixed windows, so that evidence citations stay useful for fractional and sparse frame rates.

Acceptance criteria:

- [x] Document the matching model and its limits (`docs/ARTIFACT_SCHEMA.md`).
- [x] Tile each frame to the next actual frame's time (half-open) so fractional FPS and missing frames are handled, and a boundary segment is not double-counted.
- [x] Attach trailing audio after the last frame to the last frame.
- [x] Add nearest-frame fallback so a segment overlapping no interval is never dropped.
- [x] Add tests for fractional FPS, boundary, trailing, spanning, and fallback behavior; keep the JSON schema unchanged.

### v0.6.0 Release

As a maintainer, I can ship the evidence-search filters, multi-bundle indexing, semantic/hybrid search via Ollama, investigate noise filtering, the e2e spec reorg, and the MCP server as a tagged release.

Acceptance criteria:

- [x] `CHANGELOG.md` has a `0.6.0` section and `README.md` reports `v0.6.0`.
- [x] Tag `v0.6.0` is pushed and drives the build version (GoReleaser `-X main.version`).
- [x] The GitHub release workflow publishes checksums and Darwin/Linux tarballs.
- [x] No source videos, generated bundles, local VecLite databases, or build output are tracked.

### MCP Server with Go SDK

As a coding agent, I want vidtrace to expose bundle validation, evidence search, and analysis through MCP tools, so that agent clients can call vidtrace without shell parsing.

Acceptance criteria:

- [x] Use the official Go MCP SDK instead of a custom protocol layer.
- [x] Add read-only tools for `validate`, `search`, `compare`, and `analyze` (plus `investigate`).
- [x] Keep tool responses structured and aligned with existing `--json` contracts.
- [x] Do not expose commands that mutate videos or generated bundles by default.
- [x] Add tests for tool schemas and handler responses (handlers plus an in-memory round trip).

### v0.5.0 Release

As a maintainer, I can ship the Studio review, BM25 evidence search, investigation handoff, and VitePress docs work as a tagged release.

Acceptance criteria:

- [x] Commit `e8ce88d` is pushed to `main`.
- [x] Tag `v0.5.0` is pushed.
- [x] GitHub release `v0.5.0` is published with checksums and Darwin/Linux tarballs.
- [x] CI and release workflows pass.
- [x] No source videos, generated artifact bundles, local VecLite databases, `node_modules`, or VitePress build output are tracked.

### Evidence Search Foundation

As an agent, I can search bug-video evidence by keyword, so that I can find the timestamp, frame, OCR, and transcript moments that explain a bug before inspecting the codebase.

Acceptance criteria:

- [x] Keep extraction independent from VecLite indexing.
- [x] Keep ADR 0003 as the evidence-search architecture record.
- [x] Add `vidtrace index <bundle> --db <path>` as an optional command.
- [x] Add `vidtrace search <db> <query> --json` for evidence lookup.
- [x] Index one document per timeline entry with content built from timestamp, OCR text, transcript text, and frame path.
- [x] Index records include `schema_version`, `bundle`, `source_video`, `time_seconds`, `source`, `frame`, `ocr_path`, `has_ocr`, and `has_transcript`.
- [x] Support BM25 keyword search first with VecLite `v0.15.0`.
- [x] Add unit tests and CLI JSON tests for index/search behavior.
- [x] Document that vecgrep is the companion codebase search tool after `vidtrace` finds relevant video evidence.

### Semantic and Hybrid Evidence Search

As an agent, I want semantic and hybrid search over video evidence, so that paraphrased bug descriptions find the right timestamp even when the exact words differ.

Acceptance criteria:

- [x] Add explicit embedding provider configuration (`--embed ollama --embed-model`, `--ollama-url`).
- [x] Store and validate an evidence embedding profile; reject mixing providers/models/dimensions.
- [x] Add semantic and hybrid search modes (`--mode`) without changing the BM25 JSON contract.
- [x] Keep keyword search available as the default when no embedding provider is configured.
- [x] Orchestrate Ollama over HTTP behind an `Embedder` interface; report Ollama as optional in `doctor`.
- [x] Cover the embed client, semantic/hybrid index and search, profile guard, and CLI wiring with tests.

### Evidence Search Filters

As an agent, I can search one evidence database that holds many bundles and narrow results to a specific bundle, source video, evidence source, or time window.

Acceptance criteria:

- [x] Add `--bundle`, `--source-video`, `--source`, `--min-time`, and `--max-time` flags to `vidtrace search`.
- [x] Apply filters with VecLite payload filters without changing the BM25 ranking contract.
- [x] Echo active filters in JSON under a `filters` object and omit it when no filter is set.
- [x] Reject a `--min-time` greater than `--max-time` before opening the database.
- [x] Cover cross-bundle filtering and CLI flag behavior with tests.

### Agent Investigation Handoff

As a coding agent, I can generate a compact handoff from video evidence to code search, so that I can move from "the user clicked a ticket and it failed" to likely files, routes, handlers, tests, and documentation.

Acceptance criteria:

- [x] Add `vidtrace investigate <bundle> --query <text> [--codebase <path>]`.
- [x] Return timestamped video evidence plus suggested code-search queries.
- [x] When a codebase path is provided, recommend vecgrep commands rather than indexing code inside `vidtrace`.
- [x] Keep the output useful as Markdown and JSON.

### Multi-Bundle Evidence Indexing

As an agent, I can index many bundles into one database in a single command, so that cross-bundle search and filters work over a whole set of recordings.

Acceptance criteria:

- [x] `vidtrace index` accepts multiple bundle paths (including shell globs).
- [x] Validate every bundle before any write so an invalid path fails fast.
- [x] De-duplicate repeated paths and re-index idempotently by `evidence_id`.
- [x] Report per-bundle and aggregate totals; keep single-bundle JSON unchanged.
- [x] Cover multi-bundle indexing, dedup, and fail-fast with tests.

### Investigate Suggestion Noise Reduction

As a coding agent, I want suggested code searches to skip browser chrome and dates, so that suggestions point at bug-relevant terms instead of address-bar and clock text.

Acceptance criteria:

- [x] Drop host/domain tokens, `http`/`https`/`www`/`localhost`, and browser chrome from suggestions.
- [x] Drop month and day names and four-digit years from suggestions.
- [x] Preserve code-like tokens (for example ticket IDs) and the verbatim user query.
- [x] Cover noise filtering and code-token preservation with tests.

### v0.5.0 Studio Dogfood and Review Workflow

As a human reviewer, I can inspect bundle metadata, open or reveal selected frame paths, and copy timestamped evidence from Studio.

Acceptance criteria:

- [x] `PLAN.md` defines the `v0.5.0` Studio dogfood plan and keeps real-video artifacts outside the repo.
- [x] `vidtrace studio <bundle>` has a metadata/details toggle.
- [x] Studio can attempt to open or reveal the selected frame path and reports failures without crashing.
- [x] Studio can copy a concise evidence summary when clipboard tooling is available.
- [x] Studio uses a compact responsive layout for wide and narrow terminals.
- [x] Human extraction output includes step progress bars without changing JSON output.
- [x] Unit tests cover metadata formatting, evidence summary formatting, frame path resolution, and platform command selection.
- [x] Glyphrun covers Studio metadata toggle, navigation, and action status text.

### VitePress Documentation Site

As a user, I can browse the Markdown docs on a Vercel-hosted VitePress site, so that install and workflow guidance is easy to read outside GitHub.

Acceptance criteria:

- [x] Reuse Markdown files instead of duplicating content.
- [x] Include install, usage, analysis, Studio, CLI contract, artifact schema, and release pages.
- [x] Configure VitePress under `docs/.vitepress`.
- [x] Add Vercel build settings.
- [x] Exclude local videos, generated bundles, `.glyphrun/`, and root `dist/`.
- [x] Add Glyphrun coverage for the docs build.

### v0.4.0 Bundle Validation and Comparison Confidence

As an agent or reviewer, I can validate an artifact bundle and understand why `compare` matched or did not match a ticket.

Acceptance criteria:

- [x] `vidtrace validate <bundle> --json` checks required files, JSON schemas, timeline entries, and referenced frame/OCR paths.
- [x] `vidtrace compare` normalizes punctuation-separated terms before scoring.
- [x] `vidtrace compare --json` includes `confidence` and `term_hits`.
- [x] Frame time calculation is documented in `docs/ARTIFACT_SCHEMA.md`.
- [x] Empty OCR entries are represented as empty strings and counted by validation.
- [x] CLI JSON behavior is covered by structural unit tests.
- [x] `task e2e` includes a glyphrun spec for `compare --json` and `validate --json`.

### v0.3.0 Agent and Human Review Loop

As a reviewer, I can extract a video, compare it with a ticket, and inspect the resulting evidence bundle from the CLI or Studio.

Acceptance criteria:

- [x] `vidtrace analyze <bundle> --ticket <path>` emits a Markdown evidence report.
- [x] `vidtrace compare <bundle> --ticket <path> --json` emits a machine-readable match assessment.
- [x] `vidtrace studio <bundle>` opens an existing bundle and shows timeline, OCR, transcript, and frame paths.
- [x] `task e2e` validates and runs `doctor/version`, docs, studio, and `extract --json` specs.
- [x] GitHub Releases and the Homebrew cask publish from tagged releases.
- [x] Install, usage, release, site-planning, analysis, and agent docs exist as Markdown.

## Now

### codemap Integration

As an agent, I can resolve code matches to enclosing symbols, list callers, compute blast radius, and pin vidtrace evidence findings to code symbols, so that the workflow from bug video to code fix includes structural code graph context.

Acceptance criteria:

- [x] `internal/codemap` wraps the codemap CLI (`SymbolAt`, `Callers`, `Callees`, `Impact`, `Semantic`, `Find`, `Context`, `Source`, `Annotate`, `Available`) with tests.
- [x] `vidtrace doctor` reports `codemap` as an optional tool.
- [x] `vidtrace investigate --connect --codemap` resolves code matches to enclosing symbols, lists callers, and computes blast radius.
- [x] `vidtrace investigate --codemap-annotate` pins vidtrace evidence findings to resolved symbols with `source="vidtrace"`.
- [x] `vidtrace investigate` without `--codemap` produces output identical to the previous version (backward compatible).
- [x] MCP server exposes `codemap_symbol_at`, `codemap_callers`, `codemap_impact`, `codemap_semantic`, `codemap_find`, `codemap_context` tools (read-only per ADR-0004).
- [x] MCP `investigate` tool accepts `codemap`, `codemap_depth`, `codemap_annotate` and returns `codemap_expansion`.
- [x] `codemap_annotate` is CLI-only (not in MCP) to respect the read-only constraint.
- [x] All codemap features degrade gracefully when codemap is not installed.
- [x] CLI docs, AGENTS.md, CLAUDE.md, and CHANGELOG.md updated.
- [x] Unit tests cover codemap wrapper, doctor, investigate codemap expansion, and MCP tools.
- [ ] E2E glyphrun spec for `investigate --connect --codemap`.
- [x] `task check` and `task e2e` pass.

### fcheap + vecgrep Integration

As an agent, I can stash artifact bundles to the fcheap vault, restore them on any machine, run real codebase search via `fcheap connect` (vecgrep), and access stash tools over MCP, so that the workflow from bug video to code fix is seamless across CLI and MCP.

Acceptance criteria:

- [x] `internal/fcheap` wraps the fcheap CLI (`Save`, `List`, `Info`, `Restore`, `Search`, `Connect`, `Available`) with tests.
- [x] `vidtrace doctor` reports `fcheap` and `vecgrep` as optional tools.
- [x] `vidtrace stash save|list|restore|info|search` CLI commands work with `--json` output and graceful "fcheap not installed" errors.
- [x] `vidtrace investigate --connect --codebase` returns real `code_matches` with `file:line` entries.
- [x] `vidtrace investigate --stash <id>` restores a stashed bundle before investigation.
- [x] `vidtrace investigate` without `--connect`/`--stash` produces output identical to the previous version (backward compatible).
- [x] MCP server exposes `stash_list`, `stash_info`, `stash_search`, `stash_connect` tools (read-only per ADR-0004).
- [x] MCP `investigate` tool accepts `connect`, `stash_id`, `connect_mode`, `connect_limit` and returns `code_matches`.
- [x] `stash_save` is CLI-only (not in MCP) to respect the read-only constraint.
- [x] All features degrade gracefully when fcheap/vecgrep are not installed.
- [x] ADR-0005, CLI_CONTRACT, built-in docs, and AGENTS.md updated.
- [x] Unit tests cover fcheap wrapper, doctor, stash CLI, investigate connect/stash, and MCP tools.
- [x] E2E glyphrun specs for stash commands and investigate --connect.
- [x] `task check` and `task e2e` pass.

## Later

### Distribution Hardening

As a user, I want a low-friction install path on macOS and Linux, so that I can run vidtrace without clone/build steps.

Acceptance criteria:

- [x] Release builds produce checksums.
- [x] Installation docs cover source builds and Homebrew cask installs.
- [x] Decide whether a Homebrew formula is useful in addition to the cask (decided: keep the cask, no formula; see `docs/RELEASE.md`).
- [x] Publish Linux `.deb` and `.rpm` packages (amd64 + arm64) via nfpms, documented in `docs/INSTALL.md`.
- [ ] Apple Developer signing and notarization: enrolled, Developer ID Application certificate and App Store Connect API key obtained, GitHub secrets set, and a GoReleaser `notarize.macos` (quill) path was wired up — but the release failed because the newly-issued certificate marks the Apple OID `1.2.840.113635.100.6.1.13` extension *critical*, which Go's `crypto/x509` (quill) rejects with `x509: unhandled critical extension`. This is a Go `crypto/x509` (quill) limitation, not a cert problem — that OID is critical on every Developer ID cert, so reissuing is futile. **Deferred by decision** (2026-06-20): the cask already strips quarantine so Homebrew users see no warning, and signing only benefits direct GitHub-Release downloaders, so the modest payoff does not yet justify the work. A concrete turnkey re-enable plan using `rcodesign` (runs on the existing Linux runner, bundles Apple intermediates, no critical-extension rejection — high likelihood) is recorded in `docs/RELEASE.md`; prove it on a throwaway prerelease tag before promoting.
