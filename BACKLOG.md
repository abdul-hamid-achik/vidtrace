# Backlog

This backlog keeps product ideas, engineering work, and integration bets visible without forcing them into the current iteration.

## Recently Completed

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

### Tighten Timeline Quality

As a coding agent, I want `timeline.json` to align transcript segments and OCR frames predictably, so that I can cite evidence by timestamp.

Acceptance criteria:

- [ ] Frame time calculation is documented.
- [x] Transcript overlap rules have unit tests.
- [ ] Empty OCR entries are preserved but clearly represented.

### Add Focused Unit Tests

As a maintainer, I want unit tests around artifact naming, metadata shape, timeline generation, and JSON contracts, so that refactors do not break agent workflows.

Acceptance criteria:

- [x] `internal/artifacts` has tests for safe bundle names.
- [x] `internal/timeline` has tests for segment overlap.
- [x] `internal/analysis` has tests for ticket/video comparison decisions.
- [ ] CLI JSON output is covered with golden or structural tests.

### Improve Comparison Scoring

As an agent, I want `vidtrace compare` to explain confidence clearly, so that I know when to trust the match assessment and when to inspect the bundle manually.

Acceptance criteria:

- [ ] Normalize terms before scoring.
- [ ] Include strongest matched and missing terms in JSON output.
- [ ] Document known limitations in `docs/ANALYSIS.md`.
- [ ] Keep the command deterministic and offline.

## Next

### Optional VecLite Index

As an agent, I want to index extracted OCR and transcript chunks into VecLite, so that I can semantically search moments across a bundle or a set of bundles.

Acceptance criteria:

- [ ] Keep extraction independent from VecLite.
- [ ] Add `vidtrace index <bundle> --db <path>` as an optional command.
- [ ] Index records include `bundle`, `timestamp`, `source`, `frame`, and `text`.
- [ ] Support plain text search first; add embeddings behind explicit config.

### Studio Review Workflow

As a human reviewer, I want Studio actions for common review tasks, so that I can move from evidence to a useful bug report faster.

Acceptance criteria:

- [ ] Add a details pane for bundle metadata.
- [ ] Add an action to open or reveal the selected frame path.
- [ ] Add a copyable evidence summary for the selected timeline entry.
- [ ] Keep keyboard navigation covered by glyphrun.

### Documentation Site

As a user, I want a small documentation site generated from the Markdown docs, so that install and workflow guidance is easy to browse outside GitHub.

Acceptance criteria:

- [ ] Reuse Markdown files instead of duplicating content.
- [ ] Include install, usage, analysis, Studio, CLI contract, artifact schema, and release pages.
- [ ] Exclude local videos, generated bundles, `.glyphrun/`, and `dist/`.

## Later

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
