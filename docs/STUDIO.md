# Studio

`vidtrace studio` opens an artifact bundle in a terminal review interface.

Use Studio when a human needs to inspect what the extractor produced without remembering the bundle file layout.

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
4. Compare the selected timestamp against the OCR text, transcript text, and frame path.
5. Press `q`, `esc`, or `ctrl+c` to exit.

## What Studio Shows

- Bundle source video and duration.
- Timeline entry count.
- Selected timestamp.
- Selected frame path.
- OCR text for the selected frame.
- Transcript segments that overlap the selected frame time.

## Current Limits

- Studio displays frame paths but does not open images yet.
- Studio reads existing bundles; extraction still runs through `vidtrace extract`.
- The interface is intentionally keyboard-first so it can be covered by glyphrun E2E specs.
