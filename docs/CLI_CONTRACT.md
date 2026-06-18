# CLI Contract

This document describes the intended stable command surface for `vidtrace`.

## Exit Codes

| Code | Meaning |
|---:|---|
| 0 | Success |
| 1 | Runtime failure or missing requirement |
| 2 | Usage error |

## Implemented Commands

### `vidtrace doctor`

Checks local dependencies.

```bash
vidtrace doctor
vidtrace doctor -json
```

The JSON output is intended for tests and automation.

### `vidtrace version`

Prints the CLI version.

```bash
vidtrace version
```

### `vidtrace tui`

Opens the terminal UI shell.

```bash
vidtrace tui
```

### `vidtrace extract`

```bash
vidtrace extract /path/to/bug.mp4 \
  --fps 1 \
  --ocr-lang eng \
  --whisper-lang en \
  --model small \
  --out ~/Downloads
```

Flags:

| Flag | Default | Meaning |
|---|---|---|
| `--fps` | `1` | Frame extraction rate |
| `--ocr-lang` | `eng` | Tesseract language list |
| `--whisper-lang` | `en` | Whisper audio language |
| `--model` | `small` | Whisper model |
| `--out` | `~/Downloads` | Parent output directory |
| `--name` | input basename | Artifact bundle name prefix |
| `--json` | `false` | Emit machine-readable run summary |

Human output is progress-oriented and readable. JSON output writes only JSON to stdout.

Example success JSON:

```json
{
  "ok": true,
  "source_video": "/path/to/bug.mp4",
  "output_dir": "/Users/example/Downloads/bug_artifacts_20260618_120000",
  "frames": 120,
  "ocr_files": 120,
  "transcript_files": [
    "transcript/bug.json",
    "transcript/bug.srt",
    "transcript/bug.tsv",
    "transcript/bug.txt",
    "transcript/bug.vtt"
  ],
  "metadata_path": "metadata.json",
  "timeline_path": "timeline.json",
  "combined_ocr_path": "ocr/ocr_all_frames.txt",
  "duration_seconds": 120
}
```

Example failure JSON:

```json
{
  "error": "source video not found: /path/to/missing.mp4",
  "ok": false
}
```

## Planned Commands

### `vidtrace timeline`

Regenerates `timeline.json` for an existing artifact bundle.

```bash
vidtrace timeline /path/to/bug_artifacts_YYYYMMDD_HHMMSS
```
