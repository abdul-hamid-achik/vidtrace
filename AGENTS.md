# AGENTS.md

This file guides coding agents working on `vidtrace`.

## Product Intent

`vidtrace` turns bug screen recordings into structured evidence bundles for humans and agents. The output should make it easy to cite visual and spoken evidence by timestamp.

## Language and Artifacts

- Write code, comments, docs, schemas, task names, and examples in English.
- Keep user-facing CLI text short, factual, and stable enough for tests.
- Use ASCII unless an existing file clearly requires Unicode.

## Tech Stack

- Go `1.26.4`
- Task `3.51.1`
- golangci-lint `2.12.2`
- GoReleaser `2.16.0`
- glyphrun `v0.1.0-e224a88-dev` or newer for E2E specs
- Bun `1.x` for the VitePress documentation site
- VitePress `1.6.4`
- Charm v2 Studio/TUI libraries:
  - `charm.land/bubbletea/v2`
  - `charm.land/bubbles/v2`
  - `charm.land/lipgloss/v2`
- External runtime tools:
  - `ffmpeg`
  - `ffprobe`
  - `tesseract`
  - `whisper`
  - `ollama` (optional, for semantic and hybrid evidence search)

## Architecture Direction

- Keep Go as the orchestration layer.
- Do not reimplement media codecs, OCR, or speech recognition in Go.
- Put command parsing in `internal/cli`.
- Put dependency checks in `internal/doctor`.
- Put bundle loading in `internal/bundle`.
- Put ticket-vs-video comparison in `internal/analysis`.
- Put terminal review UI in `internal/studio`.
- Put evidence indexing/search in `internal/evidence`, and embedding providers behind the `embed.Embedder` interface in `internal/embed`.
- Put the MCP server in `internal/mcpserver`; its tools must wrap existing internal packages and stay read-only (no video/bundle mutation).
- Put future media-tool wrappers in separate internal packages, for example `internal/ffmpeg`, `internal/tesseract`, and `internal/whisper`.
- Keep artifact schemas explicit and versionable.

## Extractor Gotchas

- The OCR combiner glob must stay scoped to `frame_*.txt`; a broader glob can read its own combined output.
- `whisper small` is the preferred default for English bug videos.
- 1 fps is a useful default for UI bug recordings, but fast animation bugs may need `--fps 2`.
- Whisper writes into an output directory, not to a single target file path.

## Agent Workflow

Use the built-in docs when you need to learn the product from the CLI:

```bash
task run -- docs agent
```

For a ticket and video, prefer this loop:

```bash
task run -- extract /path/to/bug.mp4 --json
task run -- index /path/to/bundle --db /tmp/vidtrace-evidence.veclite --json
task run -- search /tmp/vidtrace-evidence.veclite "ticket click does not work" --json
task run -- investigate /path/to/bundle --query "ticket click does not work" --codebase /path/to/repo --json
task run -- compare /path/to/bundle --ticket ticket.md --json
task run -- analyze /path/to/bundle --ticket ticket.md
```

Use `vidtrace studio <bundle>` for human inspection. Studio keys include `m` for metadata, `o` to open the selected frame, `r` to reveal it in Finder on macOS, and `c` to copy a concise evidence summary when clipboard tooling is available. Studio requires an interactive terminal and exits with guidance if it is run non-interactively, so agents should rely on the `--json` commands (or the `vidtrace mcp` server), `metadata.json`, `timeline.json`, OCR text, transcripts, and selected frame files instead of the TUI.

## Iteration Strategy

1. The Go pipeline in `internal/pipeline` is the only extractor. The legacy `scripts/extract.sh` was removed after parity was verified on synthetic and real video (see `CHANGELOG.md`).
2. Add small, testable Go commands.
3. Add unit tests for command behavior and data shaping.
4. Add or update glyphrun E2E specs for real CLI behavior.

## Development Commands

```bash
task check
task all
task run -- doctor
task run -- docs agent
task run -- validate /path/to/bundle --json
task run -- compare /path/to/bundle --ticket ticket.md --json
task run -- studio /path/to/bundle
task site
task smoke
task e2e
```

For real-video Studio dogfood, keep generated artifacts outside the repo:

```bash
task run -- extract ~/Downloads/bug.mp4 --out /tmp/vidtrace-real --name bug --json
task run -- validate /tmp/vidtrace-real/bug_artifacts_* --json
task run -- index /tmp/vidtrace-real/bug_artifacts_* --db /tmp/vidtrace-real/evidence.veclite --json
task run -- search /tmp/vidtrace-real/evidence.veclite "clicking a task does not take me to the assessment" --json
task run -- investigate /tmp/vidtrace-real/bug_artifacts_* --query "clicking a task does not take me to the assessment" --codebase /path/to/repo --json
task run -- studio /tmp/vidtrace-real/bug_artifacts_*
```

## Testing Expectations

- Run `go test ./...` after Go changes.
- Run `task check` before considering a change complete.
- Run `task all` after CLI behavior changes when local media tools are available.
- Run `task site` after documentation site navigation or VitePress config changes.
- Run `task lint` when changing Go code, or rely on `task check`.
- Run `task e2e` after command surface or Studio behavior changes.
- For extractor work, verify generated folders and files, not only stdout.
- Run `vidtrace validate <bundle> --json` before trusting a generated or fixture bundle.
- Prefer stable JSON output for tests over parsing human-readable text.
- Use glyphrun specs in `e2e/flows/` for real terminal behavior, with shared `e2e/fixtures/` bundle scripts and reusable `e2e/actions/` step snippets.

## Git Safety

- This repo may contain user-created media files or generated artifacts.
- Never commit `~/Downloads/bug.mp4` or any copied video fixture.
- Do not delete user videos or artifact folders unless explicitly asked.
- Do not rewrite history or run destructive Git commands without explicit approval.

## Notes and Documentation Boundary

The `docs/` folder is the VitePress documentation website source. It is published to Vercel and consumed by the public. Keep it strictly to public product docs: CLI contracts, schemas, architecture, testing, install, release, usage, and ADRs. Never drop strategy notes, implementation checkpoints, bug analysis, or one-off planning files into `docs/`.

Project notes, strategy, checkpoints, implementation notes, and bug analysis belong in the Obsidian vault at `~/notes/projects/<project>/`. Use the `obsidian` CLI (`/usr/local/bin/obsidian`) to create, read, and update notes there. Each project has its own folder:

- `~/notes/projects/vidtrace/` — this project's notes and index
- `~/notes/projects/veclite/` — VecLite library notes
- `~/notes/projects/vecgrep/` — Vecgrep codebase search tool notes
- `~/notes/projects/graphite/` — Graphite application notes (e.g., ticket-specific bug analysis)

When working across projects (for example a vidtrace feature that touches the VecLite API, or a bug analysis that points at the graphite codebase), put the note in the relevant project's folder and link it from that project's `index.md` using Obsidian wikilinks. Each project folder has an `index.md` that tracks current state, release checkpoints, and missing work — keep it updated when a note is added.

`BACKLOG.md` and `CHANGELOG.md` stay in the repo because they are standard project artifacts; everything else that is longer-lived than a PR should live in the vault.
