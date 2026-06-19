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

External tools own:

- video decoding and frame extraction via `ffmpeg`
- video metadata via `ffprobe`
- OCR via `tesseract`
- speech transcription via `whisper`

## Component Map

```text
cmd/vidtrace
└── internal/cli
    ├── internal/analysis
    ├── internal/bundle
    ├── internal/doctor
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

## Studio Direction

The studio is not the primary execution path. It should help users inspect existing artifacts and monitor future pipeline runs.

Current and planned panels:

- timeline viewer
- selected transcript text
- selected OCR text
- selected frame path
- artifact metadata details
- future pipeline run monitor

Use Bubble Tea commands for async work. Do not block `Update`.

## Distribution Direction

GoReleaser owns release packaging. GitHub Actions owns CI and tagged release execution. Homebrew tap publishing writes to `abdul-hamid-achik/homebrew-tap` with a dedicated token, not the default repository token.

## Non-Goals

- Reimplementing OCR in Go.
- Reimplementing speech-to-text in Go.
- Building a web UI before the CLI artifact model is stable.
- Uploading videos to remote services by default.
