# Handoff: vidtrace

This document summarizes the current state for the next agent or development session.

## Product

`vidtrace` turns bug screen recordings into structured evidence bundles so humans and coding agents can inspect frames, OCR, transcript text, metadata, and timeline evidence.

## Current State

- The repository is now a Git repo.
- The Go module is initialized as `github.com/abdul-hamid-achik/vidtrace`.
- `scripts/extract.sh` remains the working legacy extractor.
- `vidtrace doctor` is implemented in Go.
- `vidtrace tui` is a Bubble Tea v2 shell.
- `vidtrace extract` is implemented in Go.
- `vidtrace extract -json` emits machine-readable run summaries for agents.

## Key Files

- `AGENTS.md`: coding-agent conventions.
- `README.md`: product overview and quickstart.
- `Taskfile.yml`: development commands.
- `.tool-versions`: local tool pins.
- `docs/ARCHITECTURE.md`: component direction.
- `docs/CLI_CONTRACT.md`: stable command contract.
- `docs/ARTIFACT_SCHEMA.md`: target artifact JSON shapes.
- `docs/ROADMAP.md`: iteration plan.
- `BACKLOG.md`: prioritized product and engineering backlog.
- `docs/TESTING.md`: unit, smoke, and glyphrun verification strategy.
- `docs/INSTALL.md`: install paths and runtime dependencies.
- `docs/USAGE.md`: human and agent workflows.
- `docs/RELEASE.md`: CI, GoReleaser, and Homebrew tap release process.
- `.goreleaser.yaml`: release build and Homebrew tap configuration.
- `.github/workflows/`: CI and release workflows.
- `CLAUDE.md`: Claude-specific guidance.
- `docs/adr/`: architectural decisions.

## Known Environment Notes

- Go `1.26.4` is installed locally.
- Task `3.50.0` is installed locally, while `.tool-versions` now pins Task `3.51.1`.
- `ffmpeg`, `ffprobe`, `tesseract`, and `whisper` are on PATH locally.
- GoReleaser `v2.13.3` is installed locally, while `.tool-versions` pins GoReleaser `2.16.0` for CI.
- Local Tesseract languages currently include `eng`, `osd`, and `snum`; `spa` is not installed yet.
- Whisper `small.pt` is cached locally.
- `task smoke` creates a synthetic video and writes artifacts under `/tmp/vidtrace-smoke`.
- `task e2e` runs glyphrun specs for CLI behavior.
- `~/Downloads/bug.mp4` is available locally as a real sample video, but must not be committed.

## Next Best Iteration

Polish artifact and timeline quality.

Suggested checks:

1. Add `prompts/analyze-bundle.md`.
2. Improve `timeline.json` matching rules.
3. Add golden or structural JSON contract tests.
4. Design `vidtrace compare` for ticket-vs-video mismatch detection.
5. Decide whether `scripts/extract.sh` should remain as legacy fallback.

## Legacy Gotchas

- OCR combiner glob must stay scoped to `frame_*.txt`; otherwise it can read its own combined output.
- `whisper small` is the preferred default for English bug videos.
- 1 fps is a useful default for UI bug recordings, but fast animation bugs may need `--fps 2`.
- Whisper writes into an output directory, not to a single target file path.
