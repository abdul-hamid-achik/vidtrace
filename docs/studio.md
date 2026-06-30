# Studio

`vidtrace studio` opens an artifact bundle in a terminal review interface.

Use Studio when a human needs to inspect what the extractor produced without remembering the bundle file layout.

Studio is compact and keyboard-first. Wide terminals show the timeline and selected evidence side by side; narrow terminals stack the panes so paths and evidence text stay readable.

## Open a Bundle

```bash
vidtrace studio /path/to/bug_artifacts_YYYYMMDD_HHMMSS
```

You can also launch the empty Studio shell:

```bash
vidtrace studio
```

## Review Workflow

1. Extract a video with `vidtrace extract`.
2. Open the generated `output_dir` with `vidtrace studio`.
3. Move through timeline entries with `up`/`down` or `k`/`j`.
4. Press `m` to toggle bundle metadata before trusting the evidence.
5. Compare the selected timestamp against the OCR text, transcript text, and frame path.
6. Use `o`, `r`, or `c` when you need to inspect the selected frame or paste evidence elsewhere.
7. Press `q`, `esc`, or `ctrl+c` to exit.

## Keys

| Key | Action |
|---|---|
| `up`/`down`, `k`/`j` | Move through timeline entries |
| `m` | Toggle bundle metadata/details |
| `o` | Open the selected frame with the OS default opener when possible |
| `r` | Reveal the selected frame in Finder on macOS |
| `c` | Copy a concise evidence summary when clipboard tooling is available |
| `q`, `esc`, `ctrl+c` | Exit |

## What Studio Shows

- A compact status header and action status line.
- Bundle source video and duration.
- Bundle metadata details, including extraction FPS, OCR languages, and Whisper model.
- Timeline entry count.
- Selected timestamp.
- Selected frame path.
- OCR text for the selected frame.
- Transcript segments that overlap the selected frame time.
- Status messages for open, reveal, and copy actions.

## Current Limits

- Open, reveal, and copy actions depend on platform tools. Studio shows a short status message when an action is unavailable or fails.
- Studio reads existing bundles; extraction still runs through `vidtrace extract`.
- The interface is intentionally keyboard-first so it can be covered by glyphrun E2E specs.
