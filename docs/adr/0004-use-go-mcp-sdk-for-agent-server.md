# ADR-0004: Use the Go MCP SDK for the Agent Server

## Status

Accepted

## Context

vidtrace already exposes a stable `--json` CLI for agents. Coding agents increasingly speak the Model Context Protocol (MCP), and calling tools over MCP avoids brittle shell parsing and lets clients discover tool schemas. vidtrace should expose its read-only evidence workflows (validate, search, compare, analyze, investigate) to MCP clients without reimplementing the protocol or duplicating command logic.

## Decision Drivers

- Reuse the existing internal packages and `--json` contracts; do not fork logic for MCP.
- Use the official Go MCP SDK rather than a hand-rolled protocol layer.
- Keep the server read-only: no tool may mutate source videos or generated bundles.
- Keep extraction and the rest of the CLI independent of MCP.

## Considered Options

1. No MCP server; agents keep shelling out to the `--json` CLI.
2. A custom JSON-RPC protocol layer.
3. Use the official Go MCP SDK (`github.com/modelcontextprotocol/go-sdk`).

## Decision Outcome

Chosen option: **use the official Go MCP SDK**.

`vidtrace mcp` runs an MCP stdio server built with the SDK. Tools are thin wrappers over the existing internal packages (`bundle`, `evidence`, `analysis`, `investigate`), so MCP responses stay aligned with the CLI `--json` output.

## Implementation Direction

- `internal/mcpserver` builds the server and registers tools with `mcp.AddTool`, which infers input/output JSON schemas from Go structs.
- Read-only tools: `validate`, `search`, `compare`, `analyze`, `investigate`. `search` accepts the same filters and `keyword|semantic|hybrid` modes as the CLI, including optional Ollama embedding configuration.
- Tool-level failures return an MCP tool error (visible to the model with `isError`), not a protocol error, so the agent can self-correct.
- The CLI command `vidtrace mcp` runs the server over stdio and treats a client disconnect (stdin EOF) as a clean shutdown.
- Tests cover each tool handler and an in-memory client/server round trip for tool discovery and calls.

## Consequences

**Good:**

- Agents call vidtrace tools with discoverable schemas and structured results.
- No duplicated logic: MCP tools wrap the same functions the CLI uses.
- Extraction and other commands stay independent of MCP.

**Bad:**

- Adds the Go MCP SDK as a runtime dependency.
- The server surface must be kept in sync as new read-only workflows are added.
