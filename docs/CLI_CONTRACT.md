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

### `vidtrace docs`

Prints built-in product documentation.

```bash
vidtrace docs
vidtrace docs agent
vidtrace docs commands
vidtrace docs artifacts
vidtrace docs studio
```

Topics:

| Topic | Purpose |
|---|---|
| `overview` | Product summary and common workflows |
| `agent` | Agent operating contract for ticket/video analysis |
| `commands` | Command reference |
| `artifacts` | Artifact bundle reading order and schema notes |
| `studio` | Terminal Studio review workflow and keys |

### `vidtrace studio`

Opens the artifact inspection studio.

```bash
vidtrace studio
vidtrace studio /path/to/bug_artifacts_YYYYMMDD_HHMMSS
```

With a bundle path, studio shows timeline entries, metadata details, OCR text, transcript text, and frame paths.

Studio keys:

| Key | Behavior |
|---|---|
| `up`/`down`, `k`/`j` | Move through timeline entries |
| `m` | Toggle bundle metadata/details |
| `o` | Open the selected frame with the OS default opener when possible |
| `r` | Reveal the selected frame in Finder on macOS |
| `c` | Copy a concise evidence summary when clipboard tooling is available |
| `q`, `esc`, `ctrl+c` | Exit |

Open, reveal, and copy actions are best-effort. If the selected frame is missing or a platform tool is unavailable, Studio displays a short status message and continues running.

### `vidtrace validate`

Validates an artifact bundle.

```bash
vidtrace validate /path/to/bug_artifacts_YYYYMMDD_HHMMSS
vidtrace validate /path/to/bug_artifacts_YYYYMMDD_HHMMSS --json
```

Validation checks:

- bundle directory exists
- `metadata.json` parses and has `schema_version: "1"`
- `timeline.json` parses and has `schema_version: "1"`
- timeline has at least one entry
- `ocr/ocr_all_frames.txt` exists
- timeline frame paths and OCR paths exist

The command exits `0` when all checks pass and `1` when any check fails. With `--json`, stdout contains a validation report and stderr stays empty unless JSON encoding itself fails.

### `vidtrace index`

Indexes an existing artifact bundle into an optional VecLite evidence database.

```bash
vidtrace index /path/to/bug_artifacts_YYYYMMDD_HHMMSS --db /path/to/evidence.veclite
vidtrace index /path/to/bug_artifacts_YYYYMMDD_HHMMSS --db /path/to/evidence.veclite --json
```

The command validates the bundle, reads `metadata.json` and `timeline.json`, and writes one BM25 text document per timeline entry into the `evidence_entries_keyword` collection. Re-running the command for the same bundle updates existing records by `evidence_id` instead of duplicating them.

Example success JSON:

```json
{
  "ok": true,
  "bundle_dir": "/path/to/bug_artifacts_YYYYMMDD_HHMMSS",
  "db_path": "/path/to/evidence.veclite",
  "collection": "evidence_entries_keyword",
  "mode": "keyword",
  "indexed_entries": 120,
  "inserted_entries": 120,
  "updated_entries": 0,
  "summary": "Indexed 120 evidence entries into evidence_entries_keyword."
}
```

### `vidtrace search`

Searches an evidence database and returns timestamped bundle evidence.

```bash
vidtrace search /path/to/evidence.veclite "checkout button error"
vidtrace search /path/to/evidence.veclite "checkout button error" --limit 5 --json
```

The first implementation uses VecLite BM25 keyword search. Semantic and hybrid search are future additive modes and require explicit embedding-provider configuration.

Example success JSON:

```json
{
  "ok": true,
  "query": "checkout button error",
  "db_path": "/path/to/evidence.veclite",
  "collection": "evidence_entries_keyword",
  "mode": "keyword",
  "results": [
    {
      "score": 7.42,
      "evidence_id": "/path/to/bundle#12.000#frames/frame_0012.png",
      "bundle": "/path/to/bundle",
      "source_video": "/Users/example/Downloads/bug.mp4",
      "time_seconds": 12,
      "frame": "frames/frame_0012.png",
      "ocr_path": "ocr/frame_0012.txt",
      "ocr": "Checkout failed",
      "transcript": "I clicked checkout and got an error",
      "has_ocr": true,
      "has_transcript": true
    }
  ]
}
```

### `vidtrace investigate`

Creates a compact handoff from video evidence to code search.

```bash
vidtrace investigate /path/to/bug_artifacts_YYYYMMDD_HHMMSS --query "checkout button error"
vidtrace investigate /path/to/bug_artifacts_YYYYMMDD_HHMMSS --query "checkout button error" --codebase /path/to/app --json
```

The command indexes/searches the bundle evidence using the BM25 evidence-search path, then returns:

- timestamped video evidence
- suggested code-search queries
- ready-to-run vecgrep commands when `--codebase` is provided

It does not index source code inside vidtrace. Use vecgrep for codebase search.

Example success JSON:

```json
{
  "ok": true,
  "query": "checkout button error",
  "bundle_dir": "/path/to/bug_artifacts_YYYYMMDD_HHMMSS",
  "temporary_db": true,
  "codebase_dir": "/path/to/app",
  "mode": "keyword",
  "evidence": [
    {
      "score": 6.11,
      "time_seconds": 12,
      "frame": "frames/frame_0012.png",
      "ocr_path": "ocr/frame_0012.txt",
      "ocr": "Checkout failed",
      "transcript": "I clicked checkout and got an error"
    }
  ],
  "suggested_queries": [
    "checkout button error",
    "Checkout failed"
  ],
  "vecgrep_commands": [
    "cd '/path/to/app' && vecgrep search 'checkout button error' --format=json"
  ],
  "summary": "Found 1 video evidence hit(s) and 2 suggested code search(es); vecgrep command suggestions included."
}
```

### `vidtrace analyze`

Writes a Markdown evidence report for a bundle and ticket.

```bash
vidtrace analyze /path/to/bug_artifacts_YYYYMMDD_HHMMSS --ticket ticket.md
```

### `vidtrace compare`

Compares a ticket with OCR/transcript evidence from a bundle.

```bash
vidtrace compare /path/to/bug_artifacts_YYYYMMDD_HHMMSS --ticket ticket.md
vidtrace compare /path/to/bug_artifacts_YYYYMMDD_HHMMSS --ticket ticket.md --json
```

`status` is `match`, `mismatch`, or `inconclusive`.

`confidence` is `high`, `medium`, or `low`. `term_hits` identifies which matched terms appeared in OCR or transcript evidence.

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

Human output is progress-oriented and readable, with numbered step progress bars. JSON output writes only JSON to stdout.

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
