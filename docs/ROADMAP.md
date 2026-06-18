# Roadmap

## Iteration 1: Repository Foundation

- Initialize Git.
- Add Go module.
- Add Taskfile.
- Add `AGENTS.md`.
- Add `vidtrace doctor`.
- Add Charm v2 TUI shell.

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

- Add `glyphrun` tests. Done for doctor/version and extract JSON flows.
- Test exit codes, stdout, stderr, and generated files. Started.
- Add small synthetic media fixtures or fixture-generation tasks.

See `BACKLOG.md` for prioritized work beyond the roadmap.

## Iteration 5: Inspection TUI

- Browse artifact bundles.
- View transcript and OCR side by side.
- Jump from timeline entries to frames.
- Monitor long-running extraction jobs.

## Iteration 6: Distribution

- Add release builds. Done with GoReleaser config.
- Add checksums. Done with GoReleaser config.
- Document install paths. Done.
- Publish a Homebrew cask through `abdul-hamid-achik/homebrew-tap` after the first `v*` tag.
