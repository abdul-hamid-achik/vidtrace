# Testing

`vidtrace` uses layered tests because the product is both a Go codebase and a CLI that orchestrates external media tools.

## Test Layers

| Layer | Command | Purpose |
|---|---|---|
| Unit tests | `task test` | Fast Go behavior checks |
| Lint | `task lint` | Static checks through golangci-lint |
| Build check | `task build` | Compile the CLI |
| Synthetic smoke | `task smoke` | Run extraction against generated media outside the repo |
| Docs build | `task site` | Build the VitePress site for Vercel |
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

End-to-end specs live under `e2e/`: one flow per file in `e2e/flows/`, shared
bundle builders in `e2e/fixtures/`, and reusable step snippets in `e2e/actions/`.
See `e2e/README.md`.

```bash
task e2e
```

Current flows cover:

- `cli_doctor.yml`: version and doctor output.
- `cli_docs.yml`: built-in docs for humans and agents.
- `cli_compare.yml`: ticket comparison and bundle validation JSON.
- `cli_evidence_search.yml`: evidence indexing and search JSON.
- `cli_investigate.yml`: investigation handoff JSON and Markdown output.
- `docs_site.yml`: VitePress documentation build.
- `cli_studio.yml`: interactive Studio navigation, metadata toggle, and action status text in a real PTY.
- `extract_json.yml`: JSON extraction output and generated artifacts.

Artifacts are written to `.glyphrun/`, which is ignored by Git.

Evidence search is covered by Go tests in `internal/evidence` and CLI JSON tests in `internal/cli`. These tests use temporary bundles and temporary `.veclite` databases outside the repo.

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
- `index --json` and `search --json` emit stable evidence-search JSON
- transcript files exist
- OCR files match the frame count
