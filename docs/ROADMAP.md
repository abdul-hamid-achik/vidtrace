# Roadmap

## Iteration 1: Repository Foundation

- Initialize Git.
- Add Go module.
- Add Taskfile.
- Add `AGENTS.md`.
- Add `vidtrace doctor`.
- Add Charm v2 studio shell.

## Iteration 2: Go Extract Parity

- Implement `vidtrace extract`. Done.
- Match the legacy Bash output layout. Done, plus `metadata.json` and `timeline.json`.
- Add configurable `--fps`, `--ocr-lang`, `--whisper-lang`, `--model`, and `--out`. Done.
- Preserve `scripts/extract.sh` until parity is verified.

## Iteration 3: Structured Artifacts

- Add `metadata.json` from `ffprobe`. Done.
- Add stable run summary JSON. Done.
- Add initial `timeline.json` from OCR and transcript files. Done.
- Improve timeline matching and add focused unit tests.

## Iteration 4: E2E Coverage

- Add `glyphrun` tests. Done for doctor/version, docs, compare/validate, studio, and extract JSON flows.
- Test exit codes, stdout, stderr, and generated files. Started.
- Add small synthetic media fixtures or fixture-generation tasks. Started.

See `BACKLOG.md` for prioritized work beyond the roadmap.

## Iteration 5: Inspection Studio

- Browse artifact bundles. Started.
- View transcript and OCR side by side. Started.
- Jump from timeline entries to frames. Started with frame paths.
- Monitor long-running extraction jobs.

## Iteration 6: Distribution

- Add release builds. Done with GoReleaser config.
- Add checksums. Done with GoReleaser config.
- Document install paths. Done.
- Publish a Homebrew cask through `abdul-hamid-achik/homebrew-tap`. Done.

## Iteration 7: Ticket Analysis

- Add built-in agent docs. Done.
- Add reusable agent prompt. Done.
- Add `vidtrace compare` JSON output. Done.
- Add `vidtrace analyze` Markdown output. Done.
- Improve matching with normalized terms, confidence, and term hits. Done.
- Improve matching further with optional VecLite indexing.

## Iteration 8: Documentation Site Readiness

- Keep README, install, usage, analysis, Studio, release, testing, and artifact docs aligned. In progress.
- Keep `AGENTS.md` and `CLAUDE.md` focused on current agent workflows. In progress.
- Add a generated docs site after Markdown navigation stabilizes.

## Iteration 9: Bundle Validation

- Add `vidtrace validate <bundle> --json`. Done.
- Validate required files, schema versions, timeline entries, and referenced frame/OCR paths. Done.
- Cover validation with unit tests and glyphrun. Done.
