# Usage

## Human Workflow

```bash
vidtrace extract /path/to/bug.mp4
```

Human mode prints progress and a concise final summary.

## Agent Workflow

```bash
vidtrace extract /path/to/bug.mp4 --json
```

Agents can print the built-in operating guide before extraction:

```bash
vidtrace docs agent
```

With `--json`, stdout contains JSON only. Agents should read `output_dir` from the summary and inspect:

- `metadata.json`
- `timeline.json`
- `ocr/ocr_all_frames.txt`
- `transcript/*.json`
- selected `frames/frame_*.png`

Then compare the ticket with extracted evidence:

```bash
vidtrace compare "$output_dir" --ticket ticket.md --json
vidtrace analyze "$output_dir" --ticket ticket.md
```

Open a bundle in the studio:

```bash
vidtrace studio "$output_dir"
```

## Common Options

```bash
vidtrace extract /path/to/bug.mp4 \
  --fps 1 \
  --ocr-lang eng \
  --whisper-lang en \
  --model small \
  --out ~/Downloads \
  --name bug
```

| Flag | Default | Purpose |
|---|---|---|
| `--fps` | `1` | Frames extracted per second |
| `--ocr-lang` | `eng` | Tesseract language list |
| `--whisper-lang` | `en` | Whisper audio language |
| `--model` | `small` | Whisper model |
| `--out` | `~/Downloads` | Parent output directory |
| `--name` | input basename | Artifact bundle name prefix |
| `--json` | `false` | Machine-readable run summary |

## Local Real Video

A local sample may exist at `~/Downloads/bug.mp4`. Do not commit it or generated artifact bundles.

```bash
bin/vidtrace extract ~/Downloads/bug.mp4 \
  --out /tmp/vidtrace-bug-smoke \
  --name bug \
  --json
```

## Development Wrappers

```bash
task extract VIDEO=/path/to/bug.mp4
task agent VIDEO=/path/to/bug.mp4
```

Use `task agent` when testing the JSON automation contract.
