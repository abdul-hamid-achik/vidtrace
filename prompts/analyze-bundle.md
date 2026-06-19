# Analyze a vidtrace Bundle

Use this prompt when you receive a ticket description and a vidtrace artifact bundle.

## Inputs

- Ticket title and description
- Path to a vidtrace artifact bundle
- Optional expected behavior, environment, logs, or reproduction notes

## Procedure

1. Read `metadata.json` to confirm the source video, duration, frame rate, OCR language, Whisper language, and model.
2. Read `timeline.json` to identify timestamped evidence.
3. Search `ocr/ocr_all_frames.txt` for UI text, errors, labels, routes, filenames, IDs, or states mentioned in the ticket.
4. Read `transcript/*.json` for spoken context and timestamps.
5. Open selected `frames/frame_*.png` only when text evidence is ambiguous or visual confirmation matters.
6. Compare the ticket against the video evidence.

## Output

Write a concise report with these sections:

```markdown
## Summary

One or two sentences describing what the video shows.

## Ticket Match

Status: match | mismatch | inconclusive

Explain whether the ticket and video describe the same issue.

## Evidence

- `timeline.json` timestamp: evidence
- `ocr/frame_XXXX.txt`: evidence
- `transcript/<name>.json`: evidence
- `frames/frame_XXXX.png`: evidence, when needed

## Reproduction Notes

Steps inferred from the video.

## Gaps

Missing context, unclear evidence, or differences between the ticket and video.
```

## Rules

- Cite timestamps and relative paths.
- Call out mismatches directly.
- Do not assume the video matches the ticket.
- Do not commit source videos or generated bundles.
- Prefer `timeline.json` and OCR/transcript text before opening many frames.
