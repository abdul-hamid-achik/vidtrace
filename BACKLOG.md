# Backlog

This backlog keeps product ideas, engineering work, and integration bets visible without forcing them into the current iteration.

## Recently Completed

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

### Semantic and Hybrid Evidence Search

As an agent, I want semantic and hybrid search over video evidence, so that paraphrased bug descriptions find the right timestamp even when the exact words differ.

Acceptance criteria:

- [ ] Add explicit embedding provider configuration.
- [ ] Store and validate an evidence embedding profile.
- [ ] Add semantic and hybrid search modes without changing the BM25 JSON contract.
- [ ] Keep keyword search available when no embedding provider is configured.

### MCP Server with Go SDK

As a coding agent, I want vidtrace to expose bundle validation, evidence search, and analysis through MCP tools, so that agent clients can call vidtrace without shell parsing.

Acceptance criteria:

- [ ] Use the official Go MCP SDK used by the local toolchain instead of a custom protocol layer.
- [ ] Add read-only tools for `validate`, `search`, `compare`, and `analyze`.
- [ ] Keep tool responses structured and aligned with existing `--json` contracts.
- [ ] Do not expose commands that mutate videos or generated bundles by default.
- [ ] Add tests for tool schemas and handler responses.

## Later

### Timeline Matching V2

As a coding agent, I want `timeline.json` to align transcript segments and OCR frames beyond simple frame windows, so that evidence citations stay useful for fast UI changes.

Acceptance criteria:

- [ ] Document the current overlap model and its limits.
- [ ] Add tests for fractional FPS and transcript boundary behavior.
- [ ] Consider nearest-frame matching for sparse frame rates.
- [ ] Keep schema changes additive.

### Multi-Language OCR

As a QA engineer testing localized apps, I want Spanish OCR support, so that UI text in English and Spanish is captured correctly.

Acceptance criteria:

- [ ] `vidtrace doctor` reports missing requested OCR languages.
- [ ] `--ocr-lang eng+spa` fails early when `spa` data is not installed.
- [ ] Docs explain how to install Tesseract language data.

### Distribution Hardening

As a user, I want a low-friction install path on macOS and Linux, so that I can run vidtrace without clone/build steps.

Acceptance criteria:

- [x] Release builds produce checksums.
- [x] Installation docs cover source builds and Homebrew cask installs.
- [ ] Decide whether a Homebrew formula is useful in addition to the cask.
- [ ] Evaluate Apple Developer signing and notarization.
- [ ] Add Linux package guidance after there is user demand.
