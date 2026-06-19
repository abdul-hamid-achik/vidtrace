# Handoff: vidtrace

This document summarizes the current state for the next agent or development session.

## Product

`vidtrace` turns bug screen recordings into structured evidence bundles so humans and coding agents can inspect frames, OCR, transcript text, metadata, and timeline evidence.

## Current State

- The repository is now a Git repo.
- The Go module is initialized as `github.com/abdul-hamid-achik/vidtrace`.
- `scripts/extract.sh` remains the working legacy extractor.
- `vidtrace doctor` is implemented in Go.
- `vidtrace docs` prints built-in product and agent usage documentation.
- `vidtrace studio <bundle>` opens a Bubble Tea v2 bundle browser.
- `vidtrace validate <bundle> --json` checks bundle structure and referenced paths.
- `vidtrace compare <bundle> --ticket <path> --json` emits heuristic ticket/video comparison.
- `vidtrace analyze <bundle> --ticket <path>` emits a Markdown evidence report.
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
- `docs/ANALYSIS.md`: ticket-vs-video comparison workflow.
- `docs/STUDIO.md`: terminal Studio review workflow.
- `.goreleaser.yaml`: release build and Homebrew tap configuration.
- `.github/workflows/`: CI and release workflows.
- `CLAUDE.md`: Claude-specific guidance.
- `docs/adr/`: architectural decisions.

## Known Environment Notes

- Development tool versions are pinned in `.tool-versions`.
- `ffmpeg`, `ffprobe`, `tesseract`, and `whisper` are on PATH locally.
- Local Tesseract languages currently include `eng`, `osd`, and `snum`; `spa` is not installed yet.
- Whisper `small.pt` is cached locally.
- `task smoke` creates a synthetic video and writes artifacts under `/tmp/vidtrace-smoke`.
- `task e2e` runs glyphrun specs for CLI behavior.
- `~/Downloads/bug.mp4` is available locally as a real sample video, but must not be committed.

## Next Best Iteration

Build richer review and search workflows on top of the artifact bundle.

Suggested checks:

1. Add optional VecLite indexing as `vidtrace index <bundle> --db <path>`.
2. Add richer studio panes and frame opening actions.
3. Add a generated docs site from the Markdown docs.
4. Improve `timeline.json` matching rules beyond frame-window overlap.
5. Decide whether `scripts/extract.sh` should remain as legacy fallback.

## Legacy Gotchas

- OCR combiner glob must stay scoped to `frame_*.txt`; otherwise it can read its own combined output.
- `whisper small` is the preferred default for English bug videos.
- 1 fps is a useful default for UI bug recordings, but fast animation bugs may need `--fps 2`.
- Whisper writes into an output directory, not to a single target file path.
