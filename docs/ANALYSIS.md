# Analysis and Comparison

`vidtrace analyze` and `vidtrace compare` help agents and reviewers connect a ticket description to extracted video evidence.

These commands are heuristic. They search ticket terms across OCR and transcript evidence and cite matching timeline entries. They do not replace human or model review of the generated bundle.

## Analyze

```bash
vidtrace analyze /path/to/bug_artifacts_YYYYMMDD_HHMMSS --ticket ticket.md
```

`analyze` writes a Markdown report with:

- summary
- ticket match status
- matched and missing terms
- evidence references with timestamps and paths
- reproduction notes
- gaps

## Compare

```bash
vidtrace compare /path/to/bug_artifacts_YYYYMMDD_HHMMSS --ticket ticket.md
vidtrace compare /path/to/bug_artifacts_YYYYMMDD_HHMMSS --ticket ticket.md --json
```

`compare` returns one of:

- `match`: enough ticket terms appear in OCR/transcript evidence
- `mismatch`: no meaningful ticket terms appear in OCR/transcript evidence
- `inconclusive`: some terms match, but not enough for confidence

The JSON output is intended for agents:

```json
{
  "ok": true,
  "status": "match",
  "score": 0.5,
  "matched_terms": ["login", "submit"],
  "missing_terms": ["network"],
  "evidence": [
    {
      "time_seconds": 0,
      "frame": "frames/frame_0001.png",
      "ocr_path": "ocr/frame_0001.txt",
      "text": "Login failed after submit"
    }
  ]
}
```

## Agent Guidance

1. Run `vidtrace extract VIDEO --json`.
2. Read `output_dir` from stdout.
3. Run `vidtrace compare "$output_dir" --ticket ticket.md --json`.
4. If the result is `mismatch` or `inconclusive`, inspect `timeline.json`, `ocr/ocr_all_frames.txt`, transcript JSON, and selected frames before deciding.
