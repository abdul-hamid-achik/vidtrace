# Roadmap

## Iteration 1: Repository Foundation

- Initialize Git.
- Add Go module.
- Add Taskfile.
- Add `AGENTS.md`.
- Add `vidtrace doctor`.
- Add Charm v2 studio shell.

## Iteration 2: Go Extract Parity

- Implement `vidtrace extract`. Done.
- Match the legacy Bash output layout. Done, plus `metadata.json` and `timeline.json`.
- Add configurable `--fps`, `--ocr-lang`, `--whisper-lang`, `--model`, and `--out`. Done.
- Preserve `scripts/extract.sh` until parity is verified.

## Iteration 3: Structured Artifacts

- Add `metadata.json` from `ffprobe`. Done.
- Add stable run summary JSON. Done.
- Add initial `timeline.json` from OCR and transcript files. Done.
- Improve timeline matching and add focused unit tests. Done: half-open tiling by actual next-frame time, trailing-audio capture, and a nearest-frame fallback, with fractional/boundary/sparse tests.

## Iteration 4: E2E Coverage

- Add `glyphrun` tests. Done for doctor/version, docs, compare/validate, studio, and extract JSON flows.
- Test exit codes, stdout, stderr, and generated files. Started.
- Add small synthetic media fixtures or fixture-generation tasks. Started.

See the repository root `BACKLOG.md` for prioritized work beyond the roadmap.

## Iteration 5: Inspection Studio

- Browse artifact bundles. Done.
- View transcript, OCR, metadata, and frame paths side by side. Done.
- Jump from timeline entries to frames with open/reveal actions. Done for local platform tools.
- Copy concise timestamped evidence summaries. Done.
- Monitor long-running extraction jobs.

## Iteration 6: Distribution

- Add release builds. Done with GoReleaser config.
- Add checksums. Done with GoReleaser config.
- Document install paths. Done.
- Publish a Homebrew cask through `abdul-hamid-achik/homebrew-tap`. Done.

## Iteration 7: Ticket Analysis

- Add built-in agent docs. Done.
- Add reusable agent prompt. Done.
- Add `vidtrace compare` JSON output. Done.
- Add `vidtrace analyze` Markdown output. Done.
- Improve matching with normalized terms, confidence, and term hits. Done.
- Improve matching further with semantic evidence search and vecgrep codebase handoff.

## Iteration 8: Documentation Site Readiness

- Keep README, install, usage, analysis, Studio, release, testing, and artifact docs aligned. In progress.
- Keep `AGENTS.md` and `CLAUDE.md` focused on current agent workflows. In progress.
- Publish the docs site with VitePress and Vercel. Done.

## Iteration 9: Bundle Validation

- Add `vidtrace validate <bundle> --json`. Done.
- Validate required files, schema versions, timeline entries, and referenced frame/OCR paths. Done.
- Cover validation with unit tests and glyphrun. Done.

## Iteration 10: Evidence Search

- Document the VecLite evidence-search architecture. Done.
- Add `vidtrace index <bundle> --db <path>` for VecLite evidence records. Done.
- Add `vidtrace search <db> <query> --json` for timestamped bundle search. Done.
- Start with BM25 keyword search. Done.
- Filter search by bundle, source video, evidence source, and time window for multi-bundle databases. Done.
- Index multiple bundles into one database in a single command. Done.
- Add semantic and hybrid modes behind explicit embedding config. Done with an Ollama embedder behind an `Embedder` interface and a stored embedding-profile guard.
- Keep extraction independent from indexing. Done.
- Use vecgrep as the codebase search companion after video evidence is found. Done for handoff command suggestions.

## Iteration 11: Agent Tooling

- Add an MCP server using the Go MCP SDK. Done with `vidtrace mcp`.
- Expose read-only bundle validation, evidence search, compare, analysis, and investigation tools. Done.
- Keep tool responses aligned with CLI JSON contracts. Done.
