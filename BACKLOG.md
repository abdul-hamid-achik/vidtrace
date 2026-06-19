# Backlog

This backlog keeps product ideas, engineering work, and integration bets visible without forcing them into the current iteration.

## Now

### Release and Docs Polish

As a maintainer, I want CI, release automation, and site-ready docs, so that the project can ship repeatable binaries without tribal knowledge.

Acceptance criteria:

- [x] CI runs formatting, module drift, unit tests, build, lint, and GoReleaser config checks.
- [x] Tagged releases use GoReleaser.
- [x] Homebrew tap publishing targets `abdul-hamid-achik/homebrew-tap`.
- [x] Install, usage, release, and site-planning docs exist as Markdown.
- [ ] First `v*` tag is published after `HOMEBREW_TAP_TOKEN` is configured in GitHub.

### Improve E2E Confidence

As a maintainer, I want `glyphrun` specs for the core CLI flows, so that CLI regressions are caught in a real PTY before release.

Acceptance criteria:

- [x] `task e2e` validates every spec before running it.
- [x] `task e2e` runs `doctor/version` and `extract -json` specs.
- [x] E2E artifacts stay under `.glyphrun/` and are not committed.

### Tighten Timeline Quality

As a coding agent, I want `timeline.json` to align transcript segments and OCR frames predictably, so that I can cite evidence by timestamp.

Acceptance criteria:

- [ ] Frame time calculation is documented.
- [x] Transcript overlap rules have unit tests.
- [ ] Empty OCR entries are preserved but clearly represented.

### Add Focused Unit Tests

As a maintainer, I want unit tests around artifact naming, metadata shape, and timeline generation, so that refactors do not break agent contracts.

Acceptance criteria:

- [x] `internal/artifacts` has tests for safe bundle names.
- [x] `internal/timeline` has tests for segment overlap.
- [ ] JSON contracts are covered with golden or structural tests.

## Next

### Agent Analysis Prompt

As a coding agent, I want a prompt template that explains how to inspect a vidtrace bundle, so that I produce consistent bug reports.

Acceptance criteria:

- [x] Add `prompts/analyze-bundle.md`.
- [x] The prompt references `metadata.json`, `timeline.json`, OCR, transcript files, and frames.
- [x] The prompt asks the agent to call out ticket/video mismatches explicitly.

### Ticket vs Video Comparison

As a support engineer, I want vidtrace to compare a ticket description against video evidence, so that I can detect mismatched attachments quickly.

Acceptance criteria:

- [ ] Add a design doc for `vidtrace compare`.
- [ ] Input accepts ticket text and artifact bundle path.
- [ ] Output includes `match`, `mismatch`, or `inconclusive` with evidence references.

### Optional VecLite Index

As an agent, I want to index extracted OCR and transcript chunks into VecLite, so that I can semantically search moments across a bundle or a set of bundles.

Acceptance criteria:

- [ ] Keep extraction independent from VecLite.
- [ ] Add `vidtrace index <bundle> --db <path>` as an optional command.
- [ ] Index records include `bundle`, `timestamp`, `source`, `frame`, and `text`.
- [ ] Support BM25/text search first; add embeddings behind explicit config.

## Later

### Studio Artifact Browser

As a human reviewer, I want a studio view that opens an artifact bundle and lets me browse timeline, transcript, OCR, and frames, so that I can inspect evidence without remembering file paths.

Acceptance criteria:

- `vidtrace studio <bundle>` opens an existing bundle.
- Timeline entries are keyboard navigable.
- Selected entries show transcript text, OCR text, and frame path.

### Multi-Language OCR

As a QA engineer testing localized apps, I want Spanish OCR support, so that UI text in English and Spanish is captured correctly.

Acceptance criteria:

- `vidtrace doctor` reports missing requested OCR languages.
- `--ocr-lang eng+spa` fails early when `spa` data is not installed.
- Docs explain how to install Tesseract language data.

### Distribution

As a user, I want a simple installation path, so that I can run vidtrace without cloning the repo.

Acceptance criteria:

- Release builds produce checksums.
- Installation docs cover source builds and binary installs.
- Homebrew formula is considered after the first tagged release.
