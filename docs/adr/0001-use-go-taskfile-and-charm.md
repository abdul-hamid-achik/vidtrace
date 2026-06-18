# ADR-0001: Use Go, Taskfile, and Charm v2

## Status

Accepted

## Context

The Bash prototype proves that the media pipeline works, but it is hard to validate, extend, and distribute as a normal CLI. The project needs a maintainable command surface, stable tests, and a future terminal UI for artifact inspection.

## Decision Drivers

- Single binary distribution is valuable.
- The tool is local-first and file-system-heavy.
- External media tools should remain external.
- The TUI should use a well-supported Go ecosystem.
- Development commands should be easy to discover.

## Considered Options

1. Stay Bash.
2. Move to Python.
3. Move to Go with Taskfile and Charm v2.
4. Move to Node or Bun.

## Decision

Use Go for the CLI, Taskfile for developer workflows, and Charm v2 libraries for TUI work.

## Consequences

Good:

- Strong fit for a distributable CLI.
- Clear path to command and artifact tests.
- Good process control for `ffmpeg`, `tesseract`, and `whisper`.
- Charm v2 gives a capable terminal UI stack.

Tradeoffs:

- Python would be closer to Whisper and ML experimentation.
- Go still depends on external binaries for media work.
- The first migration requires keeping Bash and Go in sync until parity.

