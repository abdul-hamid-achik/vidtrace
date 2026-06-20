# v0.5 Studio Dogfood and Review Workflow

## Goal

- Validate the full workflow with `~/Downloads/bug.mp4` without committing the video or generated bundles.
- Improve Studio so a reviewer can inspect metadata, navigate evidence, open or reveal frame paths, and copy a concise evidence summary.
- Use VecLite `v0.15.0` BM25 evidence search to help agents move from a bug-video description to timestamped evidence and code-search handoff queries.

## Release Target

- Target version: `v0.5.0`.
- Release type: minor, because Studio gains new user-facing actions.
- No breaking changes to existing `extract`, `validate`, `compare`, or artifact JSON contracts.

## User Stories

- As a human reviewer, I want Studio to show bundle metadata, so I can confirm source video, duration, extraction FPS, OCR language, and Whisper model before trusting evidence.
- As a human reviewer, I want to open or reveal the selected frame path, so I can visually confirm OCR/transcript evidence.
- As a coding agent or reviewer, I want a copied evidence summary format, so I can paste timestamped evidence into a ticket or bug report.

## Implementation Plan

- Add a Studio metadata/details pane or mode.
- Add Studio keybindings:
  - `m`: toggle metadata/details view.
  - `o`: open the selected frame with the OS default opener when possible.
  - `r`: reveal the selected frame in Finder on macOS.
  - `c`: copy a concise evidence summary to clipboard when clipboard tooling is available.
  - Keep existing `up/down`, `k/j`, `q`, `esc`, and `ctrl+c`.
- Keep Studio keyboard-first and covered by Glyphrun.
- Add optional `vidtrace index`, `vidtrace search`, and `vidtrace investigate` commands for local evidence lookup and vecgrep handoff suggestions.
- Keep codebase search in vecgrep; do not index source code inside `vidtrace`.
- Keep all generated artifacts outside the repo during dogfood.
- Update docs for `docs/STUDIO.md`, `README.md`, `docs/USAGE.md`, `docs/CLI_CONTRACT.md`, `BACKLOG.md`, `CHANGELOG.md`, `AGENTS.md`, and `CLAUDE.md`.

## Dogfood Checklist

Run:

```bash
vidtrace extract ~/Downloads/bug.mp4 --out /tmp/vidtrace-real --name bug --json
vidtrace validate /tmp/vidtrace-real/bug_artifacts_* --json
vidtrace index /tmp/vidtrace-real/bug_artifacts_* --db /tmp/vidtrace-real/evidence.veclite --json
vidtrace search /tmp/vidtrace-real/evidence.veclite "clicking a task does not take me to the assessment" --json
vidtrace investigate /tmp/vidtrace-real/bug_artifacts_* --query "clicking a task does not take me to the assessment" --codebase /path/to/repo --json
vidtrace studio /tmp/vidtrace-real/bug_artifacts_*
```

- Record findings in this file under "Dogfood Notes".
- Do not commit `~/Downloads/bug.mp4` or `/tmp/vidtrace-real`.

## Public Interfaces

- No artifact schema changes.
- `vidtrace studio <bundle>` remains the command.
- `vidtrace index <bundle> --db <path>` creates an optional local VecLite database outside the artifact bundle.
- `vidtrace search <db> <query>` searches the optional evidence database.
- `vidtrace investigate <bundle> --query <text> [--codebase <path>]` creates a video-evidence to code-search handoff.
- New Studio-only keybindings are public behavior and must be documented:
  - `m`
  - `o`
  - `r`
  - `c`
- If an action cannot run on the current platform, Studio should show a short non-crashing status message.

## Test Plan

- Unit tests:
  - Metadata/detail formatting.
  - Evidence summary formatting.
  - Frame path resolution from selected timeline entry.
  - Platform command selection for open/reveal, isolated enough to avoid launching apps in tests.
  - Evidence indexing/search report formatting.
  - Investigation handoff and vecgrep command formatting.
  - Human extraction progress formatting.
- Glyphrun:
  - Extend `cli_studio.yml` to verify metadata toggle.
  - Verify navigation still works.
  - Verify unsupported/available action status text without requiring real GUI opening.
  - Add evidence search and investigation handoff specs.
  - Keep all fixtures under `.glyphrun/`.
- Full verification before release:

```bash
task all
goreleaser check
```

## Acceptance Criteria

- `PLAN.md` exists at repo root and is written in English.
- Studio shows metadata/details without leaving the TUI.
- Studio can attempt open/reveal/copy actions from the selected evidence entry.
- Studio displays clear success/failure status for actions.
- Docs and backlog reflect the new Studio workflow.
- `task all` passes.
- `v0.5.0` can be tagged and released after implementation.
- No local videos or generated artifact bundles are committed.

## Assumptions

- `PLAN.md` is a planning artifact, not a replacement for `BACKLOG.md`.
- The immediate next release should prioritize Studio dogfood and review ergonomics.
- BM25 evidence search is in scope for the current development line; semantic/hybrid evidence search remains follow-up roadmap work.
- File-opening behavior may be OS-specific; macOS support is the first target because the current environment is macOS.

## Dogfood Notes

- 2026-06-19: `bin/vidtrace extract ~/Downloads/bug.mp4 --out /tmp/vidtrace-real --name bug --json` completed successfully.
  - Output bundle: `/tmp/vidtrace-real/bug_artifacts_20260619_094713`.
  - Duration: `94.404833` seconds.
  - Frames: `94`.
  - OCR files: `94`.
  - Transcript files: `transcript/bug.json`, `transcript/bug.srt`, `transcript/bug.tsv`, `transcript/bug.txt`, and `transcript/bug.vtt`.
- 2026-06-19: `bin/vidtrace validate /tmp/vidtrace-real/bug_artifacts_20260619_094713 --json` passed with `ok: true`, `94` timeline entries, `0` empty OCR entries, and `9/9` checks passed.
- 2026-06-19: `bin/vidtrace index /tmp/vidtrace-real/bug_artifacts_20260619_094713 --db /tmp/vidtrace-real/evidence-bug-20260619.veclite --json` indexed `94` entries into `evidence_entries_keyword`.
- 2026-06-19: `bin/vidtrace search /tmp/vidtrace-real/evidence-bug-20260619.veclite "clicking a task does not take me to the assessment" --limit 5 --json` returned the relevant transcript cluster at `75s`, `76s`, `81s`, `82s`, and `84s`.
- 2026-06-19: `bin/vidtrace investigate /tmp/vidtrace-real/bug_artifacts_20260619_094713 --query "clicking a task does not take me to the assessment" --codebase /Users/abdulachik/projects/vidtrace --limit 5 --json` returned timestamped video evidence and vecgrep command suggestions.
- 2026-06-19: `bin/vidtrace studio /tmp/vidtrace-real/bug_artifacts_20260619_094713` loaded the real bundle in a PTY, rendered timeline plus selected evidence, toggled metadata with `m`, and exited cleanly with `q`.
- The transcript captured the key spoken bug evidence: the task is created, clicking it does not navigate to the assessment, and clicking again makes the connection task disappear.
- Observed OCR noise in early timeline entries, which is expected for dense browser UI captures and is not a blocker for the Studio review actions.
- Real `o`, `r`, and `c` actions were not invoked against the real bundle to avoid opening GUI apps or changing the clipboard during dogfood. Unit tests cover command selection and summary formatting; Glyphrun covers action status text without launching a GUI.
