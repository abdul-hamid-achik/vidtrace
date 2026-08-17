---
title: Clip
description: Cut video clips, make animated GIFs, and stitch videos from timestamp ranges with vidtrace clip. Supports labeled ranges and fcheap stashing.
---
# Clip

`vidtrace clip` cuts video clips, makes animated GIFs, and stitches clips from timestamp ranges. It runs entirely on `ffmpeg`. `cut`, `gif`, and `stitch` take a source video or existing clip files. `from-evidence` searches an evidence database and cuts around the hit timestamps.

Use Clip when you need a shareable slice of a bug recording: a per-issue MP4 for a ticket, a lightweight GIF for a README or chat, or a single summary video stitched from several clips.

## Subcommands

| Subcommand | Purpose |
|---|---|
| `clip cut` | Cut one or more sub-clips from a video at timestamp ranges |
| `clip gif` | Create animated GIF(s) from timestamp ranges |
| `clip stitch` | Join multiple clip files into one concatenated video |
| `clip from-evidence` | Cut clips or GIFs around evidence-search hits |
| `clip help` | Show clip help |

## Timestamps, Ranges, and Labels

Timestamps accept three forms:

| Format | Example | Seconds |
|---|---|---|
| `SS` | `45` | 45 |
| `MM:SS` | `3:40` | 220 |
| `HH:MM:SS` | `1:23:45` | 5025 |

A range is `START-END`, for example `0:18-3:40` or `14:50-16:14`.

A labeled range is `LABEL=START-END`, for example `issue1-blank-row=0:18-3:40`. The label sanitizes into the clip filename and the `clips.json` manifest. When you pass `--label`, it takes precedence over `--range`; unnamed ranges fill in only when there are more ranges than labels.

## Cut Clips

```bash
vidtrace clip cut /path/to/video.mp4 --label "issue1=0:18-3:40" --json
vidtrace clip cut /path/to/video.mp4 \
  --label "issue1=0:18-3:40" \
  --label "issue2=3:40-4:05" \
  --out /tmp/clips --json
```

By default `cut` uses ffmpeg stream copy (fast, lossless). Pass `--reencode` when the source needs accurate keyframe alignment and a clean cut at an arbitrary timestamp.

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

## Make GIFs

```bash
vidtrace clip gif /path/to/video.mp4 --label "issue1=0:18-3:40" --fps 10 --width 480 --json
```

GIFs are always re-encoded. Lower `--fps` and `--width` keep GIFs small enough for tickets and chat.

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

Example `clip gif` JSON:

```json
{
  "ok": true,
  "source_video": "/path/to/video.mp4",
  "output_dir": "/tmp/clips/intel-session_gifs_20260625_140000",
  "gifs": [
    {
      "label": "issue1-blank-row",
      "start_seconds": 18,
      "end_seconds": 220,
      "duration_seconds": 202,
      "fps": 10,
      "width": 480,
      "path": "issue1-blank-row.gif"
    }
  ]
}
```

## Stitch Clips

```bash
vidtrace clip stitch clip1.mp4 clip2.mp4 --name summary --json
```

Stitch concatenates clip files with the ffmpeg concat demuxer. Clips cut from the same source video share codec parameters and stitch cleanly. Clips with mixed codecs or resolutions may fail or produce a broken output; re-encode them to a common format first.

`clip stitch` flags:

| Flag | Default | Meaning |
|---|---|---|
| `--out` | `~/Downloads` | Parent output directory |
| `--name` | `stitched` | Output filename (without extension) |
| `--json` | `false` | Emit machine-readable JSON |

Example `clip stitch` JSON:

```json
{
  "ok": true,
  "inputs": ["/tmp/clips/clip1.mp4", "/tmp/clips/clip2.mp4"],
  "output_path": "/tmp/clips/summary/summary.mp4",
  "duration_seconds": 42.5
}
```

## From Evidence Hits

```bash
vidtrace clip from-evidence --db /tmp/evidence.veclite --query "login failed" --pad 2 --json
vidtrace clip from-evidence --db /tmp/evidence.veclite --query "login failed" --gif --json
```

`from-evidence` runs keyword search on the VecLite database, then cuts a window around each hit (`--pad` seconds on each side, default 2). Pass a video path if the source is no longer at the path stored in the index.

| Flag | Default | Meaning |
|---|---|---|
| `--db` | required | Evidence database path |
| `--query` | required | Search text |
| `--limit` | `3` | Maximum hits to cut |
| `--pad` | `2` | Seconds of context before and after each hit |
| `--gif` | `false` | Emit GIFs instead of MP4 clips |
| `--out` | `~/Downloads` | Parent output directory |
| `--name` | derived | Prefix for output filenames |
| `--reencode` | `false` | Force re-encoding for `cut` |
| `--fps` / `--width` | `10` / `480` | GIF settings when `--gif` is set |
| `--stash` | `false` | Stash the output directory to fcheap |
| `--json` | `false` | Emit machine-readable JSON |

Search is keyword-only. Use `vidtrace search` with `--mode` when you need semantic or hybrid ranking first.

## Output Layout and Manifest

Each `cut` and `gif` run writes into a timestamped, collision-free output directory under `--out`, named `<name>_clips_<YYYYMMDD_HHMMSS>` (or `<name>_gifs_<YYYYMMDD_HHMMSS>` for GIFs). A `clips.json` manifest is written alongside the clips or GIFs describing every produced file. Stitch writes `<name>.mp4` into its own timestamped directory.

## Stash to fcheap

`cut` and `gif` accept `--stash` to push the output directory into the fcheap vault after writing. Combine with repeatable `--tag` and a `--tool` tag so stashed clips are discoverable alongside stashed bundles. Stashing degrades gracefully: if `fcheap` is not installed, the clip or GIF still succeeds and the stash error is recorded in the JSON `stash_error` field.

## Current Limits

- Clip requires `ffmpeg`. `from-evidence` reads an evidence database and the source video; it does not mutate the artifact bundle.
- Stream-copy cuts (`cut` without `--reencode`) snap to the nearest keyframe, so the clip may start slightly before the requested timestamp. Use `--reencode` for frame-accurate starts.
- Stitch uses the concat demuxer and expects consistent codec parameters across inputs.
- Stash features require the optional `fcheap` CLI. `vidtrace doctor` reports whether `ffmpeg` is installed.