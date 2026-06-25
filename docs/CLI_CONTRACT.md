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

The JSON output is intended for tests and automation. Required tools are `ffmpeg`, `ffprobe`, `tesseract`, and `whisper`. Optional tools include `ollama` (for semantic/hybrid evidence search), `fcheap` (for stash vault integration), and `vecgrep` (for codebase search via `fcheap connect`).

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

Indexes one or more existing artifact bundles into an optional VecLite evidence database.

```bash
vidtrace index /path/to/bug_artifacts_YYYYMMDD_HHMMSS --db /path/to/evidence.veclite
vidtrace index /path/to/bug_artifacts_YYYYMMDD_HHMMSS --db /path/to/evidence.veclite --json
vidtrace index /path/to/bug_artifacts_* --db /path/to/evidence.veclite --json
```

The command validates each bundle, reads `metadata.json` and `timeline.json`, and writes one record per timeline entry into the single `evidence_entries` collection (BM25 over the content; a named `text` vector space is added when `--embed` is configured). Re-running the command for the same bundle updates existing records by `evidence_id` instead of duplicating them. Pre-v0.17.0 databases used separate `evidence_entries_keyword` and `evidence_entries_text` collections; run `vidtrace migrate-evidence` to convert them in place.

Pass multiple bundle paths (for example a shell glob) to build one searchable database that spans many bundles; combine with the `vidtrace search` filters to narrow back to a single bundle, source video, or time window. All bundle paths are validated before any database write, so an invalid path is rejected before the database is created or modified, and duplicate paths (including symlink aliases) are indexed once. Indexing is idempotent by `evidence_id`, so re-running after an interruption is safe.

Add `--embed ollama --embed-model <model>` to also build a semantic index (vector + content) for semantic and hybrid search. This requires a running Ollama server (default `http://localhost:11434`, override with `--ollama-url`). The keyword index is always built; the embedding flags are additive.

```bash
vidtrace index /path/to/bundle --db /path/to/evidence.veclite --embed ollama --embed-model nomic-embed-text --json
```

Index flags:

| Flag | Default | Meaning |
|---|---|---|
| `--db` | none (required) | Evidence database path |
| `--embed` | none | Embedding provider for the semantic index (`ollama`) |
| `--embed-model` | none | Embedding model name (required with `--embed`) |
| `--ollama-url` | `http://localhost:11434` | Ollama base URL |
| `--json` | `false` | Emit machine-readable JSON |

When `--embed` is set, the JSON adds `semantic_entries` and an `embedding` object (`provider`, `model`, `dimensions`); both are omitted for keyword-only indexing. The embedding profile is stored in the database, and indexing or searching it later with a different provider/model is rejected.

Example single-bundle success JSON:

```json
{
  "ok": true,
  "bundle_dir": "/path/to/bug_artifacts_YYYYMMDD_HHMMSS",
  "db_path": "/path/to/evidence.veclite",
  "collection": "evidence_entries",
  "mode": "keyword",
  "indexed_entries": 120,
  "inserted_entries": 120,
  "updated_entries": 0,
  "summary": "Indexed 120 evidence entries into evidence_entries."
}
```

When more than one bundle is indexed at once, the JSON instead reports aggregate totals plus a per-bundle `bundles` array:

```json
{
  "ok": true,
  "db_path": "/path/to/evidence.veclite",
  "collection": "evidence_entries",
  "mode": "keyword",
  "indexed_entries": 214,
  "inserted_entries": 214,
  "updated_entries": 0,
  "bundles": [
    {
      "bundle_dir": "/path/to/bug_artifacts_A",
      "indexed_entries": 120,
      "inserted_entries": 120,
      "updated_entries": 0
    },
    {
      "bundle_dir": "/path/to/bug_artifacts_B",
      "indexed_entries": 94,
      "inserted_entries": 94,
      "updated_entries": 0
    }
  ],
  "summary": "Indexed 214 evidence entries from 2 bundle(s) into evidence_entries."
}
```

### `vidtrace search`

Searches an evidence database and returns timestamped bundle evidence.

```bash
vidtrace search /path/to/evidence.veclite "checkout button error"
vidtrace search /path/to/evidence.veclite "checkout button error" --limit 5 --json
vidtrace search /path/to/evidence.veclite "checkout button error" --bundle /path/to/bundle --min-time 60 --max-time 90 --json
```

The first implementation uses VecLite BM25 keyword search. Semantic and hybrid search are future additive modes and require explicit embedding-provider configuration.

One database can hold many bundles. The filter flags narrow results so a multi-bundle database can be searched for a single bundle, source video, evidence source, or timestamp window.

Flags:

| Flag | Default | Meaning |
|---|---|---|
| `--mode` | `keyword` | Search mode: `keyword`, `semantic`, or `hybrid` |
| `--embed` | none | Embedding provider for semantic/hybrid search (`ollama`) |
| `--embed-model` | none | Embedding model name (required with `--embed`) |
| `--ollama-url` | `http://localhost:11434` | Ollama base URL |
| `--limit` | `10` | Maximum results |
| `--bundle` | none | Restrict to one bundle directory (resolved to an absolute path) |
| `--source-video` | none | Restrict to records whose `source_video` matches exactly |
| `--source` | none | Restrict to an evidence source, for example `timeline` |
| `--min-time` | none | Keep results at or after this time in seconds |
| `--max-time` | none | Keep results at or before this time in seconds |
| `--json` | `false` | Emit machine-readable JSON |

`keyword` (the default) uses BM25 and needs no embedder. `semantic` and `hybrid` embed the query and require `--embed` plus a semantic index built with the same provider and model; a mismatch is rejected. The `mode` field in the JSON output reflects the mode used, and all filters above apply in every mode.

```bash
vidtrace search /path/to/evidence.veclite "clicking a task fails" --mode hybrid --embed ollama --embed-model nomic-embed-text --json
```

When any filter is active, JSON output adds a `filters` object that echoes the applied filters. It is omitted when no filter is set, so the unfiltered BM25 contract is unchanged. A `--min-time` greater than `--max-time` fails before opening the database.

Example success JSON:

```json
{
  "ok": true,
  "query": "checkout button error",
  "db_path": "/path/to/evidence.veclite",
  "collection": "evidence_entries",
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

Example filtered JSON (the `filters` object only appears when a filter is active):

```json
{
  "ok": true,
  "query": "checkout button error",
  "db_path": "/path/to/evidence.veclite",
  "collection": "evidence_entries",
  "mode": "keyword",
  "filters": {
    "bundle": "/path/to/bundle",
    "min_time": 60,
    "max_time": 90
  },
  "results": []
}
```

### `vidtrace migrate-evidence`

Converts a pre-v0.17.0 evidence database (the three-collection layout: `evidence_entries_keyword`, `evidence_entries_text`, `evidence_meta`) into the single `evidence_entries` collection with a named `text` vector space. Running it on a modern database is a no-op (`already_migrated: true`), so it is safe to run unconditionally.

```bash
vidtrace migrate-evidence /path/to/evidence.veclite --json
```

```json
{
  "ok": true,
  "db_path": "/path/to/evidence.veclite",
  "collection": "evidence_entries",
  "keyword_records": 120,
  "semantic_records": 94,
  "migrated_records": 214,
  "dropped_legacy": true,
  "summary": "Migrated 214 records into evidence_entries."
}
```

When the database is already on the single-collection layout, `already_migrated` is `true`, `migrated_records` is `0`, and `dropped_legacy` is `false`.

### `vidtrace investigate`

Creates a compact handoff from video evidence to code search.

```bash
vidtrace investigate /path/to/bug_artifacts_YYYYMMDD_HHMMSS --query "checkout button error"
vidtrace investigate /path/to/bug_artifacts_YYYYMMDD_HHMMSS --query "checkout button error" --codebase /path/to/app --json
vidtrace investigate /path/to/bug_artifacts_YYYYMMDD_HHMMSS --query "checkout button error" --codebase /path/to/app --connect --json
vidtrace investigate --stash fcheap_stash_id --query "checkout button error" --json
```

The command indexes/searches the bundle evidence using the BM25 evidence-search path, then returns:

- timestamped video evidence
- suggested code-search queries
- ready-to-run vecgrep commands when `--codebase` is provided
- real `file:line` code matches when `--connect` is used with `--codebase` (via fcheap connect + vecgrep)

It does not index source code inside vidtrace. Use vecgrep for codebase search.

When `--stash <id>` is provided, the bundle is restored from the fcheap vault before investigation. This allows investigating a bundle without a local copy.

When `--connect` is used with `--codebase`, vidtrace saves the bundle to fcheap (if not already stashed), runs `fcheap connect` which invokes vecgrep over the codebase, and returns real code matches. If fcheap or vecgrep are not installed, the connect error is recorded in `connect_error` and the report still succeeds with evidence and suggested queries. Using `--connect` without `--codebase` is a usage error (exit code 2).

Flags:

| Flag | Default | Meaning |
|---|---|---|
| `--query` | (required) | Bug or evidence query |
| `--codebase` | none | Optional codebase path for vecgrep command suggestions |
| `--db` | none (temp) | Optional reusable evidence database path |
| `--limit` | `5` | Maximum evidence results |
| `--connect` | `false` | Run fcheap connect for real code matches (requires `--codebase`) |
| `--stash` | none | fcheap stash ID to restore and investigate instead of a local bundle |
| `--connect-mode` | none (hybrid) | vecgrep search mode: `semantic`, `keyword`, or `hybrid` |
| `--connect-limit` | `10` | Maximum code matches from `--connect` |
| `--json` | `false` | Emit machine-readable JSON |

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

When `--connect` is used, the JSON adds `code_matches` and optionally `connect_error`:

```json
{
  "ok": true,
  "query": "checkout button error",
  "bundle_dir": "/path/to/bug_artifacts_YYYYMMDD_HHMMSS",
  "codebase_dir": "/path/to/app",
  "mode": "keyword",
  "evidence": [...],
  "suggested_queries": [...],
  "vecgrep_commands": [...],
  "code_matches": [
    {
      "file": "src/checkout/handler.go",
      "score": 0.85,
      "text": "func handleCheckoutSubmit(w http.ResponseWriter, r *http.Request)"
    }
  ],
  "summary": "Found 1 video evidence hit(s) and 2 suggested code search(es); vecgrep command suggestions included; 1 code match(es) found via fcheap connect."
}
```

When `--stash` is used, the JSON adds `stash_id`:

```json
{
  "ok": true,
  "query": "checkout button error",
  "bundle_dir": "/tmp/fcheap-restore-abc123",
  "stash_id": "checkout_bug_evidence_20260623_150000",
  "mode": "keyword",
  "evidence": [...],
  "summary": "..."
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

### `vidtrace stash`

Stashes, lists, restores, inspects, and searches artifact bundles in the fcheap vault. Requires the `fcheap` CLI to be installed.

```bash
vidtrace stash save /path/to/bundle --name "bug-evidence" --json
vidtrace stash list [--tool vidtrace] [--tag bug] [--json]
vidtrace stash restore <stash-id> [--to /path/to/dir] [--json]
vidtrace stash info <stash-id> [--json]
vidtrace stash search "query" [--mode hybrid] [--limit 20] [--json]
```

Subcommands:

| Subcommand | Purpose |
|---|---|
| `save` | Save a bundle to the fcheap vault for sharing or archival |
| `list` | List stashes, optionally filtered by tool or tag |
| `restore` | Restore a stashed bundle to a local directory |
| `info` | Get metadata and file list for a stash |
| `search` | Search across all indexed stashes |

Flags:

| Flag | Default | Meaning |
|---|---|---|
| `--name` | none | Display name for the stash (save) |
| `--tool` | none | Filter by tool name, e.g. `vidtrace` (list) |
| `--tag` | none | Filter by tag (list) or tag the stash (save) |
| `--to` | none (temp) | Target directory for restore |
| `--mode` | none (hybrid) | Search mode: `keyword`, `semantic`, or `hybrid` (search) |
| `--limit` | `20` | Maximum search results |
| `--json` | `false` | Emit machine-readable JSON |

Example `stash save` JSON:

```json
{
  "ok": true,
  "id": "bug_evidence_20260623_150000",
  "name": "bug-evidence"
}
```

Example `stash list` JSON:

```json
{
  "ok": true,
  "stashes": [
    {
      "id": "bug_evidence_20260623_150000",
      "name": "bug-evidence",
      "tool": "vidtrace",
      "file_count": 120
    }
  ]
}
```

### `vidtrace clip`

Cuts video clips, makes GIFs, and stitches clips from timestamp ranges. Requires `ffmpeg`.

```bash
vidtrace clip cut /path/to/video.mp4 --label "issue1=0:18-3:40" --json
vidtrace clip gif /path/to/video.mp4 --label "issue1=0:18-3:40" --fps 10 --width 480 --json
vidtrace clip stitch clip1.mp4 clip2.mp4 --name summary --json
```

Subcommands:

| Subcommand | Purpose |
|---|---|
| `cut` | Cut one or more clips from a video at timestamp ranges |
| `gif` | Create animated GIF(s) from timestamp ranges |
| `stitch` | Join multiple clips into one concatenated video |
| `help` | Show clip help |

Timestamp formats:

| Format | Example | Seconds |
|---|---|---|
| `SS` | `45` | 45 |
| `MM:SS` | `3:40` | 220 |
| `HH:MM:SS` | `1:23:45` | 5025 |

Range format: `START-END` (e.g. `0:18-3:40`).
Label format: `LABEL=START-END` (e.g. `issue1-blank-row=0:18-3:40`).

`clip cut` flags:

| Flag | Default | Meaning |
|---|---|---|
| `--range` | none (repeatable) | Timestamp range `START-END` |
| `--label` | none (repeatable) | Named range `LABEL=START-END` (overrides `--range`) |
| `--out` | `~/Downloads` | Parent output directory |
| `--name` | video basename | Prefix for clip filenames and output directory |
| `--reencode` | `false` | Force re-encoding instead of stream copy |
| `--stash` | `false` | Stash the clips directory to fcheap after cutting |
| `--tag` | none (repeatable) | Tag for the fcheap stash |
| `--tool` | `vidtrace` | Tool tag for the fcheap stash |
| `--json` | `false` | Emit machine-readable JSON |

`clip gif` flags:

| Flag | Default | Meaning |
|---|---|---|
| `--range` | none (repeatable) | Timestamp range `START-END` |
| `--label` | none (repeatable) | Named range `LABEL=START-END` |
| `--out` | `~/Downloads` | Parent output directory |
| `--name` | video basename | Prefix for GIF filenames and output directory |
| `--fps` | `10` | GIF frame rate |
| `--width` | `480` | GIF width in pixels (height auto-scales) |
| `--stash` | `false` | Stash the GIFs directory to fcheap |
| `--tag` | none (repeatable) | Tag for the fcheap stash |
| `--tool` | `vidtrace` | Tool tag for the fcheap stash |
| `--json` | `false` | Emit machine-readable JSON |

`clip stitch` flags:

| Flag | Default | Meaning |
|---|---|---|
| `--out` | `~/Downloads` | Parent output directory |
| `--name` | `stitched` | Output filename (without extension) |
| `--json` | `false` | Emit machine-readable JSON |

Example `clip cut` JSON:

```json
{
  "ok": true,
  "source_video": "/path/to/video.mp4",
  "output_dir": "/tmp/clips/intel-session_clips_20260625_140000",
  "clips": [
    {
      "label": "issue1-blank-row",
      "start_seconds": 18,
      "end_seconds": 220,
      "duration_seconds": 202,
      "path": "issue1-blank-row.mp4"
    }
  ]
}
```

A `clips.json` manifest is written to each output directory alongside the clips or GIFs.

### `vidtrace mcp`

Runs a Model Context Protocol server over stdio so agent clients can call vidtrace's read-only evidence tools without shell parsing. Built on the official Go MCP SDK.

```bash
vidtrace mcp
```

It exposes these read-only tools, whose inputs and structured outputs mirror the corresponding `--json` contracts:

| Tool | Purpose |
|---|---|
| `validate` | Validate an artifact bundle (`bundle_dir`). |
| `search` | Search an evidence database (`db_path`, `query`, optional `mode`, filters, and `embed`/`embed_model`/`ollama_url`). |
| `compare` | Structured ticket-vs-bundle comparison (`bundle_dir`, `ticket_path`). |
| `analyze` | Markdown evidence report (`bundle_dir`, `ticket_path`). |
| `investigate` | Video-evidence to code-search handoff (`bundle_dir` or `stash_id`, `query`, optional `codebase_dir`, `connect`, `connect_mode`, `connect_limit`). |
| `stash_list` | List fcheap stashes (`tool`, `tag`). |
| `stash_info` | Get stash metadata (`stash_id`). |
| `stash_search` | Search across stashes (`query`, `mode`, `limit`). |
| `stash_connect` | Connect a stash to a codebase via vecgrep (`stash_id`, `codebase`, `query`, `mode`, `limit`, `index`). |

No tool mutates source videos or generated artifact bundles. `stash_save` is intentionally excluded from MCP to respect the read-only constraint. Tool failures are returned as MCP tool errors (visible to the model), not protocol errors. A client disconnect (stdin EOF) is a clean shutdown.

Example client registration (Claude Desktop / MCP client config):

```json
{
  "mcpServers": {
    "vidtrace": { "command": "vidtrace", "args": ["mcp"] }
  }
}
```

## Planned Commands

### `vidtrace timeline`

Regenerates `timeline.json` for an existing artifact bundle.

```bash
vidtrace timeline /path/to/bug_artifacts_YYYYMMDD_HHMMSS
```
