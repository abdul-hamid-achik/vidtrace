# ADR-0002: Orchestrate External Media Tools

## Status

Accepted

## Context

`vidtrace` needs frame extraction, metadata extraction, OCR, and speech transcription. Mature local tools already exist for these jobs.

## Decision Drivers

- Avoid reimplementing complex media and ML systems.
- Keep output reproducible and inspectable.
- Allow users to manage tool versions locally.
- Keep Go code focused on orchestration and artifact contracts.

## Decision

Use external tools:

- `ffmpeg` for frame extraction
- `ffprobe` for metadata
- `tesseract` for OCR
- `whisper` for transcription

The Go CLI validates these tools through `vidtrace doctor`, runs them as subprocesses, and turns their outputs into stable artifact bundles.

## Consequences

Good:

- Faster delivery.
- Uses proven tooling.
- Keeps the Go codebase smaller.

Tradeoffs:

- Users must install runtime dependencies.
- Errors from external tools need careful normalization.
- Cross-platform support depends on tool availability.

