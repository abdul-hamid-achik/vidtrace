# Usage

## Human Workflow

```bash
vidtrace extract /path/to/bug.mp4
```

Human mode prints progress and a concise final summary.

Progress is shown as numbered steps with bars for bundle creation, metadata, frames, OCR, transcript, timeline, and completion. JSON mode suppresses this text so stdout remains parseable.

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

Validate the bundle before deeper analysis:

```bash
vidtrace validate "$output_dir" --json
```

Index and search timestamped evidence:

```bash
vidtrace index "$output_dir" --db /tmp/vidtrace-evidence.veclite --json
vidtrace search /tmp/vidtrace-evidence.veclite "clicking a ticket does not work" --json
```

Index several bundles into one database (a shell glob expands to multiple paths):

```bash
vidtrace index /tmp/vidtrace-real/bug_artifacts_* --db /tmp/vidtrace-evidence.veclite --json
```

For semantic and hybrid search, also build an embedding index with a running [Ollama](https://ollama.com) server, then search with `--mode`:

```bash
vidtrace index "$output_dir" --db /tmp/vidtrace-evidence.veclite --embed ollama --embed-model nomic-embed-text --json
vidtrace search /tmp/vidtrace-evidence.veclite "a task click does nothing" --mode hybrid --embed ollama --embed-model nomic-embed-text --json
```

Keyword search remains the default and needs no embedder. `vidtrace doctor` reports whether Ollama is installed.

One database can index many bundles. Narrow a search to a single bundle, source video, evidence source, or time window:

```bash
vidtrace search /tmp/vidtrace-evidence.veclite "clicking a ticket does not work" \
  --bundle "$output_dir" \
  --min-time 60 --max-time 90 \
  --json
```

Create a handoff from video evidence to code search:

```bash
vidtrace investigate "$output_dir" \
  --query "clicking a ticket does not work" \
  --codebase /path/to/app \
  --json
```

Then compare the ticket with extracted evidence:

```bash
vidtrace compare "$output_dir" --ticket ticket.md --json
vidtrace analyze "$output_dir" --ticket ticket.md
```

Open a bundle in the studio:

```bash
vidtrace studio "$output_dir"
```

Use `up`/`down` or `k`/`j` to move through timeline entries. Press `m` for metadata, `o` to open the selected frame, `r` to reveal it in Finder on macOS, and `c` to copy a concise evidence summary when clipboard tooling is available. Press `q` to exit. See `docs/STUDIO.md` for the review workflow.

Studio is compact by default. It shows timeline and selected evidence side by side when the terminal is wide enough, and stacks them on narrow terminals.

## Documentation Site

Build the VitePress documentation site:

```bash
task site
```

The build output is `docs/.vitepress/dist`, which is the Vercel output directory.

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
  --out /tmp/vidtrace-real \
  --name bug \
  --json
```

## Development Wrappers

```bash
task extract VIDEO=/path/to/bug.mp4
task agent VIDEO=/path/to/bug.mp4
task run -- validate /path/to/bundle --json
task run -- index /path/to/bundle --db /tmp/vidtrace-evidence.veclite --json
task run -- search /tmp/vidtrace-evidence.veclite "ticket click" --json
task run -- investigate /path/to/bundle --query "ticket click" --codebase /path/to/app --json
task site
```

Use `task agent` when testing the JSON automation contract.
