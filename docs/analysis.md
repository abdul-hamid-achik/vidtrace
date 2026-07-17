---
title: Analysis and Comparison
description: Compare a ticket description against extracted video evidence with confidence scores, term hits, and Markdown analysis reports.
---
# Analysis and Comparison

`vidtrace analyze` and `vidtrace compare` help agents and reviewers connect a ticket description to extracted video evidence.

These commands are heuristic. They search ticket terms across OCR and transcript evidence and cite matching timeline entries. They do not replace human or model review of the generated bundle.

Term matching is deterministic and offline. It normalizes punctuation-separated words, so examples like `log-in`, `log in`, and `login` can match the same evidence.

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

`compare` returns a coverage-aware status:

- `supported`: enough ticket terms appear in OCR/transcript evidence
- `no_observation`: evidence exists but ticket terms do not appear
- `inconclusive`: some terms match, but not enough for confidence
- `contradicted`: the ticket claims a failure, but evidence mostly shows success language
- `unknown`: no usable terms or evidence were available

Pass `--mode hybrid` to also rank the full ticket text against a temporary evidence index (BM25). Classic term hits still appear; hybrid mainly reorders evidence and can lift thin inconclusive cases when the ticket text finds strong timeline hits.
`confidence` explains how much support the heuristic found:

- `high`: a match with broad term coverage and multiple term-level hits
- `medium`: a match with enough evidence for a first pass
- `low`: a mismatch or thin evidence that needs manual inspection

The JSON output is intended for agents:

```json
{
  "ok": true,
  "status": "supported",
  "coverage": "supported",
  "confidence": "medium",
  "score": 0.5,
  "matched_terms": ["login", "submit"],
  "missing_terms": ["network"],
  "term_hits": [
    {
      "term": "login",
      "source": "ocr",
      "time_seconds": 0,
      "frame": "frames/frame_0001.png",
      "ocr_path": "ocr/frame_0001.txt",
      "text": "Login failed after submit"
    }
  ],
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

Known limitations:

- Text matching cannot prove visual state by itself; inspect frames for important conclusions.
- OCR errors can hide real matches or create noisy terms.
- Generic ticket terms can inflate the score. Prefer `term_hits` and `evidence` over score alone.
- The command does not use embeddings or remote models.

## Agent Guidance

1. Run `vidtrace extract VIDEO --json`.
2. Read `output_dir` from stdout.
3. Run `vidtrace validate "$output_dir" --json`.
4. Run `vidtrace compare "$output_dir" --ticket ticket.md --json`.
5. If the result is `mismatch` or `inconclusive`, inspect `timeline.json`, `ocr/ocr_all_frames.txt`, transcript JSON, and selected frames before deciding.
