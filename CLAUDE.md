# CLAUDE.md

This file gives Claude Code and Claude-style agents project-specific operating guidance.

## Start Here

Read `AGENTS.md` first. It contains the canonical coding-agent instructions for this repo.

## Project Summary

`vidtrace` is a Go CLI that turns bug videos into structured evidence bundles for humans and agents. It orchestrates external tools instead of reimplementing media processing:

- `ffmpeg` for frames
- `ffprobe` for metadata
- `tesseract` for OCR
- `whisper` for transcript generation

## Common Commands

```bash
task check
task smoke
task run -- docs agent
task run -- validate /path/to/bundle --json
task run -- compare /path/to/bundle --ticket ticket.md --json
task run -- analyze /path/to/bundle --ticket ticket.md
task run -- studio /path/to/bundle
task e2e
```

Run the extractor for a local real video without committing artifacts:

```bash
bin/vidtrace extract ~/Downloads/bug.mp4 --out /tmp/vidtrace-bug-smoke --name bug --json
```

## Files Not To Commit

- Videos such as `~/Downloads/bug.mp4`
- Generated artifact bundles
- `.glyphrun/`
- `bin/`

## Agent Contract

When `--json` is used, stdout must remain parseable JSON. Progress logs and human summaries must not be mixed into JSON output.

Use `vidtrace docs agent` for the fastest in-CLI product guide. For ticket/video work, inspect `metadata.json`, `timeline.json`, OCR text, transcript files, and selected frame images before deciding whether the ticket matches the video.
