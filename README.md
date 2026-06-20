# vidtrace

Turn bug videos into timestamped evidence bundles that humans and coding agents can inspect.

`vidtrace` is a local-first Go CLI. It takes a screen recording of a bug and produces frames, OCR text, transcripts, metadata, and a timeline that connects what was visible with what was said.

## Status

`v0.5.0` is published. The Go CLI can extract evidence bundles, emit stable JSON for automation, validate bundles, compare a ticket against video evidence, open a compact terminal Studio for human review, search bundle evidence with VecLite BM25, and ship through GitHub Releases plus the Homebrew tap.

The next development line should focus on semantic/hybrid evidence search and MCP tooling while keeping extraction independent from optional indexes.

The project is still early. Treat `--json`, `metadata.json`, and `timeline.json` as the main contracts and change them deliberately.

## Who It Is For

- QA and support engineers who receive bug videos.
- Developers who need timestamped evidence instead of vague reproduction notes.
- Coding agents that cannot "watch" a video directly but can inspect files and JSON.

## What It Produces

```text
bug_artifacts_YYYYMMDD_HHMMSS/
├── frames/
│   └── frame_0001.png
├── ocr/
│   ├── frame_0001.txt
│   └── ocr_all_frames.txt
├── transcript/
│   ├── bug.txt
│   ├── bug.srt
│   ├── bug.vtt
│   ├── bug.json
│   └── bug.tsv
├── metadata.json
├── timeline.json
└── README.txt
```

`timeline.json` is the main agent-facing artifact. It maps extracted frames to OCR text and overlapping transcript segments.

## Install

### From Source

```bash
git clone https://github.com/abdul-hamid-achik/vidtrace.git
cd vidtrace
task build
bin/vidtrace doctor
```

### With Homebrew

The cask is published from tagged releases:

```bash
brew tap abdul-hamid-achik/tap
brew install --cask abdul-hamid-achik/tap/vidtrace
vidtrace version
vidtrace doctor
```

If macOS blocks the unsigned binary on first run:

```bash
xattr -dr com.apple.quarantine /opt/homebrew/Caskroom/vidtrace/*/vidtrace
```

See `docs/INSTALL.md` for runtime dependencies and install details.

## Use

Check local dependencies:

```bash
vidtrace doctor
vidtrace doctor -json
```

Print built-in product and agent docs:

```bash
vidtrace docs
vidtrace docs agent
vidtrace docs artifacts
vidtrace docs studio
```

Run a human-readable extraction:

```bash
vidtrace extract /path/to/bug.mp4
```

Human extraction prints step progress bars for bundle creation, metadata, frame extraction, OCR, transcript, and timeline writing.

Run an agent-readable extraction:

```bash
vidtrace extract /path/to/bug.mp4 --json
```

With `--json`, stdout is parseable JSON only. An agent should read `output_dir` from the summary and inspect:

- `metadata.json`
- `timeline.json`
- `ocr/ocr_all_frames.txt`
- `transcript/*.json`
- selected `frames/frame_*.png`

Validate an artifact bundle:

```bash
vidtrace validate /path/to/bug_artifacts_YYYYMMDD_HHMMSS
vidtrace validate /path/to/bug_artifacts_YYYYMMDD_HHMMSS --json
```

Index and search timestamped evidence:

```bash
vidtrace index /path/to/bug_artifacts_YYYYMMDD_HHMMSS --db /tmp/vidtrace-evidence.veclite --json
vidtrace search /tmp/vidtrace-evidence.veclite "clicking a ticket does not work" --json
```

Create an investigation handoff for code search:

```bash
vidtrace investigate /path/to/bug_artifacts_YYYYMMDD_HHMMSS \
  --query "clicking a ticket does not work" \
  --codebase /path/to/app \
  --json
```

Compare a ticket with an artifact bundle:

```bash
vidtrace analyze /path/to/bug_artifacts_YYYYMMDD_HHMMSS --ticket ticket.md
vidtrace compare /path/to/bug_artifacts_YYYYMMDD_HHMMSS --ticket ticket.md --json
```

Open the studio browser:

```bash
vidtrace studio /path/to/bug_artifacts_YYYYMMDD_HHMMSS
```

In Studio, use `up`/`down` or `k`/`j` to move through timeline entries, `m` to toggle metadata, `o` to open the selected frame, `r` to reveal it in Finder on macOS, and `c` to copy a concise evidence summary when clipboard tooling is available.

Studio uses a compact keyboard-first layout. Wide terminals show timeline and evidence details side by side; narrow terminals stack the panes.

```bash
vidtrace extract /path/to/bug.mp4 \
  --fps 1 \
  --ocr-lang eng \
  --whisper-lang en \
  --model small \
  --out ~/Downloads \
  --name bug
```

See `docs/USAGE.md` and `docs/CLI_CONTRACT.md` for the full command contract.

## Develop

Tool versions are pinned in `.tool-versions`.

```bash
task build
task test
task lint
task check
task agent VIDEO=/path/to/bug.mp4
task run -- validate /path/to/bundle --json
task run -- index /path/to/bundle --db /tmp/vidtrace-evidence.veclite --json
task run -- search /tmp/vidtrace-evidence.veclite "ticket click" --json
task run -- investigate /path/to/bundle --query "ticket click" --codebase /path/to/app --json
task run -- compare /path/to/bundle --ticket ticket.md --json
task run -- studio /path/to/bundle
task site
```

Useful local tasks:

| Task | Purpose |
|---|---|
| `task run -- doctor` | Run any CLI command |
| `task extract VIDEO=/path/to/bug.mp4` | Human extraction wrapper |
| `task agent VIDEO=/path/to/bug.mp4` | JSON extraction wrapper |
| `task smoke` | Synthetic end-to-end extraction outside the repo |
| `task site` | Build the VitePress docs site into `docs/.vitepress/dist` |
| `task e2e` | Verify and run glyphrun CLI specs |
| `task all` | Full local verification |

Run the synthetic smoke extraction outside the repo:

```bash
task smoke
```

Generate the local documentation site:

```bash
task site
```

The docs site is a VitePress app configured for Vercel. Vercel runs `bun install --frozen-lockfile`, `bun run docs:build`, and serves `docs/.vitepress/dist`.

## Real Video Fixture

A local sample video may exist at:

```bash
~/Downloads/bug.mp4
```

Do not commit that video. Use `/tmp` for generated artifacts:

```bash
bin/vidtrace extract ~/Downloads/bug.mp4 --out /tmp/vidtrace-bug-smoke --name bug --json
```

For `v0.5.0` Studio dogfood, keep the generated bundle outside the repo:

```bash
bin/vidtrace extract ~/Downloads/bug.mp4 --out /tmp/vidtrace-real --name bug --json
bin/vidtrace validate /tmp/vidtrace-real/bug_artifacts_* --json
bin/vidtrace index /tmp/vidtrace-real/bug_artifacts_* --db /tmp/vidtrace-real/evidence.veclite --json
bin/vidtrace search /tmp/vidtrace-real/evidence.veclite "clicking a task does not take me to the assessment" --json
bin/vidtrace investigate /tmp/vidtrace-real/bug_artifacts_* --query "clicking a task does not take me to the assessment" --codebase /path/to/repo --json
bin/vidtrace studio /tmp/vidtrace-real/bug_artifacts_*
```

## Release

GitHub Actions runs CI on pushes and pull requests. A tag like `vX.Y.Z` runs GoReleaser, creates release archives and checksums, and updates `abdul-hamid-achik/homebrew-tap` when `HOMEBREW_TAP_TOKEN` is configured.

See `docs/RELEASE.md` for the full release process.

## Improve

Start with:

- `BACKLOG.md` for prioritized product and engineering work.
- `CHANGELOG.md` for release history.
- `prompts/analyze-bundle.md` for reusable agent analysis instructions.
- `docs/ANALYSIS.md` for ticket-vs-video comparison.
- `docs/STUDIO.md` for the terminal Studio workflow.
- `docs/index.md` for site-ready documentation navigation.
- `docs/ARCHITECTURE.md` for component boundaries.
- `docs/CLI_CONTRACT.md` for command behavior.
- `docs/ARTIFACT_SCHEMA.md` for bundle schemas.
- `docs/TESTING.md` for verification strategy.

Current high-value improvements:

- Add semantic/hybrid evidence search and MCP workflows tracked in `BACKLOG.md`.
- Use `docs/adr/0003-use-veclite-for-optional-evidence-search.md` as the architecture record for optional VecLite indexing.
- Dogfood the `v0.5.0` Studio review workflow with real videos.
- Improve `timeline.json` matching beyond the current frame-window overlap rules.
- Evaluate signing/notarization for macOS distribution.

## Project Conventions

- Persistent repo content is written in English.
- Generated media and artifact bundles are not committed.
- `--json` output is an automation contract; keep it stable.
- `vidtrace docs agent` is the fastest way for an agent to learn the expected workflow.
- External tools remain external. Go orchestrates them.

See `AGENTS.md` and `CLAUDE.md` for agent-specific guidance.
