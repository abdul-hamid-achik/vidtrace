# Backlog

This backlog keeps product ideas, engineering work, and integration bets visible without forcing them into the current iteration.

## Recently Completed

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
