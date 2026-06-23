# Architecture

## Overview

`vidtrace` is a local-first CLI that orchestrates media tools to transform a bug video into an evidence bundle.

The Go application owns:

- command parsing
- dependency checks
- pipeline orchestration
- artifact layout
- metadata and timeline JSON
- human-readable status output
- terminal Studio for bundle review
- ticket-vs-video analysis
- artifact bundle validation

External tools own:

- video decoding and frame extraction via `ffmpeg`
- video metadata via `ffprobe`
- OCR via `tesseract`
- speech transcription via `whisper`
- bundle stashing and cross-stash search via `fcheap` (optional)
- codebase search via `vecgrep` through `fcheap connect` (optional)

## Component Map

```text
cmd/vidtrace
└── internal/cli
    ├── internal/analysis
    ├── internal/bundle
    ├── internal/doctor
    ├── internal/evidence
    ├── internal/embed
    ├── internal/fcheap
    ├── internal/investigate
    ├── internal/studio
    ├── internal/pipeline
    ├── internal/artifacts
    ├── internal/ffmpeg
    ├── internal/tesseract
    ├── internal/whisper
    └── internal/timeline
```

## Pipeline Target

1. Validate input video and options.
2. Create a deterministic artifact directory.
3. Capture video metadata with `ffprobe`.
4. Extract frames with `ffmpeg`.
5. OCR frames with `tesseract`.
6. Transcribe audio with `whisper`.
7. Combine OCR files.
8. Generate `timeline.json`.
9. Write `README.txt`.

## Human and Agent Interfaces

The same CLI command supports both users:

- Human mode prints progress and a concise final summary.
- Agent mode uses `-json` and writes parseable JSON only to stdout.

Human-readable logs should not be required for automation. Prefer JSON fields and generated artifact files as automation contracts.

Human extraction progress is intentionally coarse and step-based. It should help a reviewer understand which external tool is running without becoming a machine contract.

`vidtrace validate <bundle> --json` is the quickest way to check whether a generated or fixture bundle is structurally useful before analysis.

## Search Boundary

Evidence search is optional and separate from extraction. `internal/evidence` reads validated bundles and writes a local VecLite database for BM25 keyword search over timeline entries. Source-code search stays outside the core pipeline and should use vecgrep as the companion tool.

`internal/investigate` builds on evidence search to create a compact handoff: timestamped video evidence, suggested code-search queries, and vecgrep command suggestions when a codebase path is provided. With `--connect`, it runs `fcheap connect` (vecgrep) to return real `file:line` code matches. With `--stash`, it restores a stashed bundle from the fcheap vault before investigation.

`internal/fcheap` wraps the fcheap CLI for bundle stashing (`Save`, `Restore`), vault search (`List`, `Info`, `Search`), and codebase connect (`Connect`). It mirrors the `internal/ffmpeg`/`internal/tesseract`/`internal/whisper` pattern of shelling out to external CLI tools.

## Documentation Site

The documentation site is a VitePress app rooted at `docs/` and built with Bun scripts. This keeps public docs deployable on Vercel without adding a docs generator to the Go CLI. The build output is `docs/.vitepress/dist`, and local media or generated bundles stay outside the site root.

## Studio Direction

The studio is not the primary execution path. It should help users inspect existing artifacts in a compact terminal layout and monitor future pipeline runs.

Current and planned panels:

- timeline viewer
- selected transcript text
- selected OCR text
- selected frame path
- artifact metadata details
- frame open/reveal and evidence copy actions
- future pipeline run monitor

Use Bubble Tea commands for async work. Do not block `Update`.

## Distribution Direction

GoReleaser owns release packaging. GitHub Actions owns CI and tagged release execution. Homebrew tap publishing writes to `abdul-hamid-achik/homebrew-tap` with a dedicated token, not the default repository token.

## Non-Goals

- Reimplementing OCR in Go.
- Reimplementing speech-to-text in Go.
- Building a web UI before the CLI artifact model is stable.
- Uploading videos to remote services by default.
