# Glyphrun Specs

These specs exercise `vidtrace` in a real PTY.

```bash
task glyphcheck
task e2e
```

The specs intentionally use synthetic media and write all temporary files under `.glyphrun/`, which is ignored by Git.

Current specs:

- `cli_doctor.yml`: verifies `doctor` and `version`.
- `extract_json.yml`: verifies `extract --json` and generated artifact files.

Use `glyph context latest --format md` after a failure.
