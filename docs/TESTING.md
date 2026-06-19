# Testing

`vidtrace` uses layered tests because the product is both a Go codebase and a CLI that orchestrates external media tools.

## Test Layers

| Layer | Command | Purpose |
|---|---|---|
| Unit tests | `task test` | Fast Go behavior checks |
| Lint | `task lint` | Static checks through golangci-lint |
| Build check | `task build` | Compile the CLI |
| Synthetic smoke | `task smoke` | Run extraction against generated media outside the repo |
| Glyphrun E2E | `task e2e` | Verify specs, real PTY CLI behavior, and artifacts |

## Standard Checks

Run this before handing off code:

```bash
task check
```

Run full local verification, including E2E:

```bash
task all
```

## Glyphrun

Glyphrun specs live in `specs/glyphrun/`.

```bash
task e2e
```

Current specs cover:

- `cli_doctor.yml`: version and doctor output.
- `cli_docs.yml`: built-in docs for humans and agents.
- `cli_compare.yml`: ticket comparison and bundle validation JSON.
- `cli_studio.yml`: interactive Studio navigation in a real PTY.
- `extract_json.yml`: JSON extraction output and generated artifacts.

Artifacts are written to `.glyphrun/`, which is ignored by Git.

## CI

GitHub Actions runs formatting, module drift, unit tests, build, lint, and `goreleaser check`.

CI does not run the media smoke path because Whisper and OCR runtime dependencies are expensive and platform-sensitive. Run this locally before release work:

```bash
task all
```

## Real Video Testing

A local sample video may exist at:

```bash
~/Downloads/bug.mp4
```

Do not commit this video or generated bundles. Run real-video checks outside the repo:

```bash
bin/vidtrace extract ~/Downloads/bug.mp4 --out /tmp/vidtrace-bug-smoke --name bug --json
```

## What To Assert

For agent-facing behavior, prefer JSON and generated files over human text:

- exit code
- valid JSON on stdout for `--json`
- artifact bundle exists
- `metadata.json` exists and has `schema_version`
- `timeline.json` exists and has entries
- `compare --json` emits a stable result shape
- transcript files exist
- OCR files match the frame count
