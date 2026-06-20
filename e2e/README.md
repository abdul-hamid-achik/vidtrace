# End-to-End Specs

These glyphrun specs exercise `vidtrace` in a real PTY. They use synthetic media
and write all temporary files under `.glyphrun/`, which is ignored by Git.

```bash
task e2e
```

## Layout

- `flows/` — one spec per user-facing flow. Each is a self-contained behavior
  contract (intent, target, outcomes).
- `fixtures/` — shell scripts that build deterministic sample bundles, shared by
  flows so bundle setup is not duplicated inline.
- `actions/` — reusable step snippets imported with `imports` and called with
  `use:` (see `glyph docs snippets`).

## Flows

- `cli_doctor.yml`: verifies `doctor` and `version`.
- `cli_docs.yml`: verifies built-in agent and artifact docs.
- `cli_compare.yml`: verifies `compare --json` and `validate --json` against a fixture bundle.
- `cli_evidence_search.yml`: verifies `index`/`search --json`, filters, multi-bundle indexing, and the semantic-without-embedder error.
- `cli_investigate.yml`: verifies the investigation handoff JSON/Markdown and OCR-noise filtering.
- `docs_site.yml`: verifies the VitePress documentation build.
- `cli_studio.yml`: verifies the interactive Studio can open and navigate a bundle.
- `extract_json.yml`: verifies `extract --json` and generated artifact files.

## Fixtures

- `fixtures/sample_bundle.sh <bundle-dir> [source-video]` — builds the ticket
  sample bundle used by the evidence-search and investigate flows.

## Actions

- `actions/wait_clean_exit.yml` — wait for the target to exit cleanly, then
  snapshot the final screen as `result`. Imported by the non-interactive flows.

Run a single flow with `glyph run e2e/flows/<name>.yml --format md`, and use
`glyph context latest --format md` after a failure.
