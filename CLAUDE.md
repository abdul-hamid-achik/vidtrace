# CLAUDE.md

This file gives Claude Code and Claude-style agents project-specific operating guidance.

## Start Here

Read `AGENTS.md` first. It contains the canonical coding-agent instructions for this repo.

## Project Summary

`vidtrace` is a Go CLI that turns bug videos into structured evidence bundles for humans and agents. It orchestrates external tools instead of reimplementing media processing:

- `ffmpeg` for frames
- `ffprobe` for metadata
- `tesseract` for OCR
- `whisper` for transcript generation
- `fcheap` (optional) for bundle stashing and vault restoration
- `vecgrep` (optional) for semantic codebase search via `fcheap connect`

## Common Commands

```bash
task check
task smoke
task run -- docs agent
task run -- validate /path/to/bundle --json
task run -- compare /path/to/bundle --ticket ticket.md --json
task run -- analyze /path/to/bundle --ticket ticket.md
task run -- investigate /path/to/bundle --query "ticket click" --codebase /path/to/repo --connect --json
task run -- stash save /path/to/bundle --name "bug-evidence" --json
task run -- studio /path/to/bundle
task site
task e2e
```

Run the extractor for a local real video without committing artifacts:

```bash
bin/vidtrace extract ~/Downloads/bug.mp4 --out /tmp/vidtrace-real --name bug --json
bin/vidtrace validate /tmp/vidtrace-real/bug_artifacts_* --json
bin/vidtrace index /tmp/vidtrace-real/bug_artifacts_* --db /tmp/vidtrace-real/evidence.veclite --json
bin/vidtrace search /tmp/vidtrace-real/evidence.veclite "clicking a task does not take me to the assessment" --json
bin/vidtrace investigate /tmp/vidtrace-real/bug_artifacts_* --query "clicking a task does not take me to the assessment" --codebase /path/to/repo --json
bin/vidtrace investigate /tmp/vidtrace-real/bug_artifacts_* --query "clicking a task does not take me to the assessment" --codebase /path/to/repo --connect --json
bin/vidtrace stash save /tmp/vidtrace-real/bug_artifacts_* --name "bug-evidence" --json
bin/vidtrace studio /tmp/vidtrace-real/bug_artifacts_*
```

## Files Not To Commit

- Videos such as `~/Downloads/bug.mp4`
- Generated artifact bundles
- `.glyphrun/`
- `bin/`
- `docs/.vitepress/dist/`

## Agent Contract

When `--json` is used, stdout must remain parseable JSON. Progress logs and human summaries must not be mixed into JSON output.

Use `vidtrace docs agent` for the fastest in-CLI product guide. For ticket/video work, inspect `metadata.json`, `timeline.json`, OCR text, transcript files, and selected frame images before deciding whether the ticket matches the video. In Studio, use `m` for metadata, `o` to open the selected frame, `r` to reveal it in Finder on macOS, and `c` to copy a concise evidence summary when clipboard tooling is available.

## Notes and Documentation Boundary

The `docs/` folder is the VitePress website source (published to Vercel). Keep it to public product docs only: CLI contracts, schemas, architecture, testing, install, release, usage, and ADRs. Do not drop strategy notes, implementation checkpoints, bug analysis, or planning files into `docs/`.

Project notes, strategy, checkpoints, implementation notes, and bug analysis belong in the Obsidian vault at `~/notes/projects/<project>/`. Use the `obsidian` CLI (`/usr/local/bin/obsidian`) to read and update notes there. Each project has its own folder (`vidtrace`, `veclite`, `vecgrep`, `graphite`, etc.) with an `index.md` that tracks current state — keep it updated when a note is added, and link notes with Obsidian wikilinks.

`BACKLOG.md` and `CHANGELOG.md` stay in the repo; everything longer-lived than a PR lives in the vault. See `AGENTS.md` "Notes and Documentation Boundary" for the full rule.
