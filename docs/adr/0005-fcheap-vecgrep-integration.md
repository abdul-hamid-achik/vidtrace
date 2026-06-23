# ADR-0005: Integrate fcheap and vecgrep for Bundle Stashing and Code Search

## Status

Accepted

## Context

vidtrace's `investigate` command produces timestamped video evidence and suggests vecgrep commands as strings, but does not actually run code search or persist bundles for sharing. Two companion tools exist in the same ecosystem:

- **fcheap** (file.cheap): a local stash vault that saves, restores, searches, and connects artifact bundles to codebases. It already detects vidtrace bundles (`bundle_type: "vidtrace"`) and extracts searchable text from them.
- **vecgrep**: a semantic codebase search tool that fcheap's `connect` command wraps to run code search driven by stashed artifact text.

Integrating these tools lets vidtrace move from "suggesting" code searches to "running" them, and lets users stash bundles for archival, sharing, and cross-machine access.

## Decision Drivers

- Keep Go as the orchestration layer; do not add Go library dependencies for fcheap or vecgrep.
- Follow the existing pattern: external CLI tools (ffmpeg, tesseract, whisper) are wrapped by Go packages that shell out via `os/exec`.
- Respect ADR-0004: MCP tools stay read-only; `stash_save` mutates the vault and must be CLI-only.
- Degrade gracefully: if fcheap or vecgrep are not installed, all features report clear errors without crashing.
- Keep `--json` output backward-compatible; new fields use `omitempty`.

## Considered Options

1. Import fcheap/vecgrep as Go libraries.
2. Shell out to the `fcheap` and `vecgrep` CLI binaries, mirroring the ffmpeg/tesseract/whisper pattern.
3. Do not integrate; keep suggesting vecgrep commands as strings only.

## Decision Outcome

Chosen option: **shell out to the fcheap and vecgrep CLI binaries**.

A new `internal/fcheap` package wraps the `fcheap` CLI via `os/exec`, providing `Save()`, `List()`, `Info()`, `Restore()`, `Search()`, `Connect()`, and `Available()`. This mirrors how `internal/ffmpeg`, `internal/tesseract`, and `internal/whisper` work. Vecgrep is never called directly by vidtrace; it is invoked through `fcheap connect`, which handles query extraction, vecgrep index management, and result formatting.

## Implementation Direction

- `internal/fcheap` wraps the fcheap CLI with typed Go functions and JSON parsing.
- `internal/doctor` reports `fcheap` and `vecgrep` as optional tools alongside `ollama`.
- `vidtrace stash save|list|restore|info|search` CLI commands wrap fcheap operations.
- `vidtrace investigate --connect --codebase` runs `fcheap connect` and returns real `file:line` code matches. `--stash <id>` restores a stashed bundle before investigation.
- MCP server exposes read-only stash tools (`stash_list`, `stash_info`, `stash_search`, `stash_connect`) and enhances `investigate` with `connect`, `stash_id`, `connect_mode`, and `connect_limit` fields. `stash_save` is excluded from MCP to respect the read-only constraint.
- All features degrade gracefully when fcheap or vecgrep are not installed.

## Consequences

**Good:**

- vidtrace can stash bundles for sharing, archival, and cross-machine access.
- `investigate --connect` returns real code matches, not just suggested commands.
- `investigate --stash` enables investigating bundles without a local copy.
- MCP clients get read-only stash tools for evidence discovery across the vault.
- No new Go library dependencies; the integration is purely CLI wrapping.

**Bad:**

- Adds two more optional runtime dependencies (`fcheap`, `vecgrep`).
- The `stash save` command is CLI-only, creating a slight asymmetry with the MCP surface.
- The `fcheap connect` flow saves a temporary stash when none is provided, which creates a stash in the vault that the caller may need to clean up.