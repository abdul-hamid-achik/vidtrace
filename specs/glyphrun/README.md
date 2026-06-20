# Glyphrun Specs

These specs exercise `vidtrace` in a real PTY.

```bash
task e2e
```

The specs intentionally use synthetic media and write all temporary files under `.glyphrun/`, which is ignored by Git.

Current specs:

- `cli_doctor.yml`: verifies `doctor` and `version`.
- `cli_docs.yml`: verifies built-in agent and artifact docs.
- `cli_compare.yml`: verifies `compare --json` and `validate --json` against a fixture bundle.
- `cli_evidence_search.yml`: verifies `index --json` and `search --json` against a fixture bundle.
- `cli_investigate.yml`: verifies investigation handoff JSON and Markdown output.
- `docs_site.yml`: verifies the VitePress documentation build.
- `cli_studio.yml`: verifies the interactive studio can open and navigate a bundle.
- `extract_json.yml`: verifies `extract --json` and generated artifact files.

Use `glyph context latest --format md` after a failure.
