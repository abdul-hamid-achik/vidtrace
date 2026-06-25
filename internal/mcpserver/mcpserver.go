// Package mcpserver exposes vidtrace's read-only evidence tools over the Model
// Context Protocol using the official Go MCP SDK, so agent clients can validate
// bundles, search evidence, compare/analyze tickets, and build investigation
// handoffs without shell parsing.
package mcpserver

import (
	"context"
	"errors"
	"io"
	"strings"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abdul-hamid-achik/vidtrace/internal/analysis"
	"github.com/abdul-hamid-achik/vidtrace/internal/bundle"
	"github.com/abdul-hamid-achik/vidtrace/internal/codemap"
	"github.com/abdul-hamid-achik/vidtrace/internal/embed"
	"github.com/abdul-hamid-achik/vidtrace/internal/evidence"
	"github.com/abdul-hamid-achik/vidtrace/internal/fcheap"
	"github.com/abdul-hamid-achik/vidtrace/internal/investigate"
)

// New builds the vidtrace MCP server with read-only evidence tools registered.
// The tools mirror the CLI `--json` contracts and never mutate videos or
// generated artifact bundles.
func New(version string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "vidtrace",
		Title:   "vidtrace evidence tools",
		Version: version,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "validate",
		Description: "Validate a vidtrace artifact bundle: required files, JSON schemas, timeline entries, and referenced frame/OCR paths.",
	}, validateTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search",
		Description: "Search a vidtrace evidence database for timestamped evidence. Modes: keyword (default, BM25), semantic, or hybrid; semantic and hybrid require an Ollama embedder and a matching semantic index.",
	}, searchTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "compare",
		Description: "Compare a ticket against an artifact bundle and return a structured match assessment (status, confidence, term hits, evidence).",
	}, compareTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "analyze",
		Description: "Compare a ticket against an artifact bundle and return a Markdown evidence report.",
	}, analyzeTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "investigate",
		Description: "Turn a bug query into timestamped video evidence plus suggested code searches and vecgrep commands for a codebase. Supports --connect to run fcheap connect for real code matches and --stash to restore a stashed bundle.",
	}, investigateTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "stash_list",
		Description: "List fcheap stashes in the vault, optionally filtered by tool or tag.",
	}, stashListTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "stash_info",
		Description: "Get detailed info about a fcheap stash including file list and metadata.",
	}, stashInfoTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "stash_search",
		Description: "Search across all indexed fcheap stashes for matching content.",
	}, stashSearchTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "stash_connect",
		Description: "Connect a fcheap stash to a codebase using vecgrep to find file:line code matches. The stash text drives the code search.",
	}, stashConnectTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "codemap_symbol_at",
		Description: "Resolve a file:line position to its enclosing symbol (FQN, kind, range). The entry point for joining vidtrace evidence onto the code graph.",
	}, codemapSymbolAtTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "codemap_callers",
		Description: "List functions/methods that call a given symbol.",
	}, codemapCallersTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "codemap_impact",
		Description: "Impact analysis for a symbol: blast radius (transitive callers) and test coverage.",
	}, codemapImpactTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "codemap_semantic",
		Description: "Semantic search across the code graph by meaning.",
	}, codemapSemanticTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "codemap_find",
		Description: "Find symbols by name (fast, offline). Returns matching symbol names and file locations.",
	}, codemapFindTool)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "codemap_context",
		Description: "Everything about a symbol in one call: definition, callers, callees, tests, and annotations.",
	}, codemapContextTool)

	return server
}

// Serve runs the MCP server over stdio until the context is cancelled or the
// client disconnects. A normal client disconnect (stdin EOF / connection close)
// is treated as a clean shutdown rather than an error.
func Serve(ctx context.Context, version string) error {
	if err := New(version).Run(ctx, &mcp.StdioTransport{}); err != nil && !isCleanShutdown(err) {
		return err
	}
	return nil
}

func isCleanShutdown(err error) bool {
	if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) || errors.Is(err, mcp.ErrConnectionClosed) {
		return true
	}
	// The jsonrpc2 layer reports a normal stream close with these messages, whose
	// error types live in an internal package and cannot be matched by type. Match
	// only these specific phrases so genuine transport errors still surface.
	msg := err.Error()
	return strings.Contains(msg, "server is closing") || strings.Contains(msg, "client is closing")
}

// ToolNames lists the registered tool names, for documentation and tests.
func ToolNames() []string {
	return []string{"validate", "search", "compare", "analyze", "investigate", "stash_list", "stash_info", "stash_search", "stash_connect", "codemap_symbol_at", "codemap_callers", "codemap_impact", "codemap_semantic", "codemap_find", "codemap_context"}
}

// ValidateInput selects the bundle to validate.
type ValidateInput struct {
	BundleDir string `json:"bundle_dir" jsonschema:"path to the artifact bundle directory"`
}

func validateTool(_ context.Context, _ *mcp.CallToolRequest, in ValidateInput) (*mcp.CallToolResult, bundle.ValidationReport, error) {
	if strings.TrimSpace(in.BundleDir) == "" {
		return toolError[bundle.ValidationReport]("bundle_dir is required")
	}
	return nil, bundle.Validate(in.BundleDir), nil
}

// SearchInput mirrors the `vidtrace search` flags.
type SearchInput struct {
	DBPath      string   `json:"db_path" jsonschema:"path to the evidence .veclite database"`
	Query       string   `json:"query" jsonschema:"natural-language or keyword query"`
	Limit       int      `json:"limit,omitempty" jsonschema:"maximum results (default 10)"`
	Mode        string   `json:"mode,omitempty" jsonschema:"keyword (default), semantic, or hybrid"`
	Bundle      string   `json:"bundle,omitempty" jsonschema:"restrict results to a single bundle directory"`
	SourceVideo string   `json:"source_video,omitempty" jsonschema:"restrict results to a source video path"`
	Source      string   `json:"source,omitempty" jsonschema:"restrict results to an evidence source, e.g. timeline"`
	MinTime     *float64 `json:"min_time,omitempty" jsonschema:"keep results at or after this time in seconds"`
	MaxTime     *float64 `json:"max_time,omitempty" jsonschema:"keep results at or before this time in seconds"`
	Embed       string   `json:"embed,omitempty" jsonschema:"embedding provider for semantic/hybrid search (ollama)"`
	EmbedModel  string   `json:"embed_model,omitempty" jsonschema:"embedding model name for the provider"`
	OllamaURL   string   `json:"ollama_url,omitempty" jsonschema:"Ollama base URL (default http://localhost:11434)"`
}

func searchTool(_ context.Context, _ *mcp.CallToolRequest, in SearchInput) (*mcp.CallToolResult, evidence.SearchReport, error) {
	embedder, err := embed.Build(in.Embed, in.EmbedModel, in.OllamaURL)
	if err != nil {
		return toolError[evidence.SearchReport](err.Error())
	}
	report, err := evidence.Search(evidence.SearchOptions{
		DBPath:      in.DBPath,
		Query:       in.Query,
		Limit:       in.Limit,
		Mode:        in.Mode,
		Embedder:    embedder,
		Bundle:      in.Bundle,
		SourceVideo: in.SourceVideo,
		Source:      in.Source,
		MinTime:     in.MinTime,
		MaxTime:     in.MaxTime,
	})
	if err != nil {
		return toolError[evidence.SearchReport](err.Error())
	}
	return nil, report, nil
}

// CompareInput selects the bundle and ticket to compare.
type CompareInput struct {
	BundleDir  string `json:"bundle_dir" jsonschema:"path to the artifact bundle directory"`
	TicketPath string `json:"ticket_path" jsonschema:"path to the ticket markdown or text file"`
}

func compareTool(_ context.Context, _ *mcp.CallToolRequest, in CompareInput) (*mcp.CallToolResult, analysis.Result, error) {
	result, err := analysis.Compare(analysis.Options{BundleDir: in.BundleDir, TicketPath: in.TicketPath})
	if err != nil {
		return toolError[analysis.Result](err.Error())
	}
	return nil, result, nil
}

// AnalyzeInput selects the bundle and ticket to analyze.
type AnalyzeInput struct {
	BundleDir  string `json:"bundle_dir" jsonschema:"path to the artifact bundle directory"`
	TicketPath string `json:"ticket_path" jsonschema:"path to the ticket markdown or text file"`
}

// AnalyzeOutput carries the Markdown evidence report.
type AnalyzeOutput struct {
	Markdown string `json:"markdown" jsonschema:"the Markdown evidence report"`
}

func analyzeTool(_ context.Context, _ *mcp.CallToolRequest, in AnalyzeInput) (*mcp.CallToolResult, AnalyzeOutput, error) {
	result, err := analysis.Compare(analysis.Options{BundleDir: in.BundleDir, TicketPath: in.TicketPath})
	if err != nil {
		return toolError[AnalyzeOutput](err.Error())
	}
	markdown := analysis.Markdown(result)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: markdown}},
	}, AnalyzeOutput{Markdown: markdown}, nil
}

// InvestigateInput mirrors the `vidtrace investigate` flags. DBPath is
// intentionally omitted so investigation always uses an ephemeral temp database
// and can never point indexing at a persistent, user-writable location.
type InvestigateInput struct {
	BundleDir    string `json:"bundle_dir,omitempty" jsonschema:"path to the artifact bundle directory (optional if stash_id is set)"`
	Query        string `json:"query" jsonschema:"bug or evidence query"`
	CodebaseDir  string `json:"codebase_dir,omitempty" jsonschema:"optional codebase path for vecgrep command suggestions"`
	Limit        int    `json:"limit,omitempty" jsonschema:"maximum evidence results (default 5)"`
	Connect      bool   `json:"connect,omitempty" jsonschema:"run fcheap connect to find real code matches in the codebase"`
	StashID      string `json:"stash_id,omitempty" jsonschema:"fcheap stash ID to restore and investigate instead of a local bundle"`
	ConnectMode  string `json:"connect_mode,omitempty" jsonschema:"vecgrep search mode for connect: semantic, keyword, or hybrid"`
	ConnectLimit int    `json:"connect_limit,omitempty" jsonschema:"maximum code matches from connect (default 10)"`
	// Codemap enables structural code graph expansion after connect surfaces
	// code candidates. Requires Connect and CodebaseDir.
	Codemap bool `json:"codemap,omitempty" jsonschema:"run codemap expansion after connect to resolve symbols, callers, and blast radius"`
	// CodemapDepth controls the blast radius depth (default 3).
	CodemapDepth int `json:"codemap_depth,omitempty" jsonschema:"max hops for codemap blast radius (default 3)"`
	// CodemapAnnotate pins vidtrace evidence findings to resolved symbols.
	CodemapAnnotate bool `json:"codemap_annotate,omitempty" jsonschema:"pin vidtrace evidence findings to resolved codemap symbols"`
}

func investigateTool(_ context.Context, _ *mcp.CallToolRequest, in InvestigateInput) (*mcp.CallToolResult, investigate.Report, error) {
	if strings.TrimSpace(in.Query) == "" {
		return toolError[investigate.Report]("query is required")
	}
	if strings.TrimSpace(in.BundleDir) == "" && strings.TrimSpace(in.StashID) == "" {
		return toolError[investigate.Report]("bundle_dir or stash_id is required")
	}
	if in.Connect && strings.TrimSpace(in.CodebaseDir) == "" {
		return toolError[investigate.Report]("connect requires codebase_dir")
	}
	if in.Codemap && !in.Connect {
		return toolError[investigate.Report]("codemap requires connect")
	}
	report, err := investigate.Run(investigate.Options{
		BundleDir:       in.BundleDir,
		Query:           in.Query,
		CodebaseDir:     in.CodebaseDir,
		Limit:           in.Limit,
		Connect:         in.Connect,
		StashID:         in.StashID,
		ConnectMode:     in.ConnectMode,
		ConnectLimit:    in.ConnectLimit,
		Codemap:         in.Codemap,
		CodemapDepth:    in.CodemapDepth,
		CodemapAnnotate: in.CodemapAnnotate,
	})
	if err != nil {
		return toolError[investigate.Report](err.Error())
	}
	return nil, report, nil
}

// StashListInput filters stashes in the vault.
type StashListInput struct {
	Tool string `json:"tool,omitempty" jsonschema:"filter by tool name (e.g. vidtrace)"`
	Tag  string `json:"tag,omitempty" jsonschema:"filter by tag"`
}

// StashListOutput wraps the list of stashes (MCP output schemas must be objects).
type StashListOutput struct {
	Stashes []fcheap.StashEntry `json:"stashes"`
}

func stashListTool(ctx context.Context, _ *mcp.CallToolRequest, in StashListInput) (*mcp.CallToolResult, StashListOutput, error) {
	if !fcheap.Available() {
		return toolError[StashListOutput]("fcheap is not installed or not on PATH")
	}
	entries, err := fcheap.List(ctx, in.Tool, in.Tag)
	if err != nil {
		return toolError[StashListOutput](err.Error())
	}
	return nil, StashListOutput{Stashes: entries}, nil
}

// StashInfoInput selects a stash to inspect.
type StashInfoInput struct {
	StashID string `json:"stash_id" jsonschema:"the stash ID to inspect"`
}

func stashInfoTool(ctx context.Context, _ *mcp.CallToolRequest, in StashInfoInput) (*mcp.CallToolResult, fcheap.StashInfo, error) {
	if strings.TrimSpace(in.StashID) == "" {
		return toolError[fcheap.StashInfo]("stash_id is required")
	}
	if !fcheap.Available() {
		return toolError[fcheap.StashInfo]("fcheap is not installed or not on PATH")
	}
	info, err := fcheap.Info(ctx, in.StashID)
	if err != nil {
		return toolError[fcheap.StashInfo](err.Error())
	}
	return nil, info, nil
}

// StashSearchInput searches across all indexed stashes.
type StashSearchInput struct {
	Query string `json:"query" jsonschema:"search query"`
	Mode  string `json:"mode,omitempty" jsonschema:"search mode: keyword, semantic, or hybrid"`
	Limit int    `json:"limit,omitempty" jsonschema:"maximum results (default 20)"`
}

func stashSearchTool(ctx context.Context, _ *mcp.CallToolRequest, in StashSearchInput) (*mcp.CallToolResult, fcheap.SearchResult, error) {
	if strings.TrimSpace(in.Query) == "" {
		return toolError[fcheap.SearchResult]("query is required")
	}
	if !fcheap.Available() {
		return toolError[fcheap.SearchResult]("fcheap is not installed or not on PATH")
	}
	result, err := fcheap.Search(ctx, in.Query, in.Mode, in.Limit)
	if err != nil {
		return toolError[fcheap.SearchResult](err.Error())
	}
	return nil, result, nil
}

// StashConnectInput connects a stash to a codebase for code search.
type StashConnectInput struct {
	StashID  string `json:"stash_id" jsonschema:"the stash ID whose content drives the code search"`
	Codebase string `json:"codebase" jsonschema:"absolute path to the codebase directory to search"`
	Query    string `json:"query,omitempty" jsonschema:"override the query auto-extracted from the stash"`
	Mode     string `json:"mode,omitempty" jsonschema:"vecgrep search mode: semantic, keyword, or hybrid"`
	Limit    int    `json:"limit,omitempty" jsonschema:"max code matches (default 10)"`
	Index    bool   `json:"index,omitempty" jsonschema:"build the vecgrep index for the codebase first"`
}

func stashConnectTool(ctx context.Context, _ *mcp.CallToolRequest, in StashConnectInput) (*mcp.CallToolResult, fcheap.ConnectResult, error) {
	if strings.TrimSpace(in.StashID) == "" {
		return toolError[fcheap.ConnectResult]("stash_id is required")
	}
	if strings.TrimSpace(in.Codebase) == "" {
		return toolError[fcheap.ConnectResult]("codebase is required")
	}
	if !fcheap.Available() {
		return toolError[fcheap.ConnectResult]("fcheap is not installed or not on PATH")
	}
	result, err := fcheap.Connect(ctx, fcheap.ConnectOptions{
		StashID:     in.StashID,
		CodebaseDir: in.Codebase,
		Query:       in.Query,
		Mode:        in.Mode,
		Limit:       in.Limit,
		Index:       in.Index,
	})
	if err != nil {
		return toolError[fcheap.ConnectResult](err.Error())
	}
	return nil, result, nil
}

// CodemapSymbolAtInput resolves a file:line to its enclosing symbol.
type CodemapSymbolAtInput struct {
	File string `json:"file" jsonschema:"project-relative file path"`
	Line int    `json:"line" jsonschema:"1-based line number"`
}

func codemapSymbolAtTool(ctx context.Context, _ *mcp.CallToolRequest, in CodemapSymbolAtInput) (*mcp.CallToolResult, codemap.SymbolAtResult, error) {
	if strings.TrimSpace(in.File) == "" {
		return toolError[codemap.SymbolAtResult]("file is required")
	}
	if in.Line <= 0 {
		return toolError[codemap.SymbolAtResult]("line is required")
	}
	if !codemap.Available() {
		return toolError[codemap.SymbolAtResult]("codemap is not installed or not on PATH")
	}
	result, err := codemap.SymbolAt(ctx, in.File, in.Line)
	if err != nil {
		return toolError[codemap.SymbolAtResult](err.Error())
	}
	return nil, result, nil
}

// CodemapCallersInput lists callers of a symbol.
type CodemapCallersInput struct {
	Symbol  string `json:"symbol" jsonschema:"the symbol name to look up"`
	Precise bool   `json:"precise,omitempty" jsonschema:"use the language server for exact results (Go via gopls)"`
}

func codemapCallersTool(ctx context.Context, _ *mcp.CallToolRequest, in CodemapCallersInput) (*mcp.CallToolResult, codemap.CallersResult, error) {
	if strings.TrimSpace(in.Symbol) == "" {
		return toolError[codemap.CallersResult]("symbol is required")
	}
	if !codemap.Available() {
		return toolError[codemap.CallersResult]("codemap is not installed or not on PATH")
	}
	result, err := codemap.Callers(ctx, in.Symbol, in.Precise)
	if err != nil {
		return toolError[codemap.CallersResult](err.Error())
	}
	return nil, result, nil
}

// CodemapImpactInput analyzes the blast radius of a symbol.
type CodemapImpactInput struct {
	Symbol string `json:"symbol" jsonschema:"the symbol to analyze"`
	Depth  int    `json:"depth,omitempty" jsonschema:"max hops for the blast radius (default 3)"`
}

func codemapImpactTool(ctx context.Context, _ *mcp.CallToolRequest, in CodemapImpactInput) (*mcp.CallToolResult, codemap.ImpactResult, error) {
	if strings.TrimSpace(in.Symbol) == "" {
		return toolError[codemap.ImpactResult]("symbol is required")
	}
	if !codemap.Available() {
		return toolError[codemap.ImpactResult]("codemap is not installed or not on PATH")
	}
	result, err := codemap.Impact(ctx, in.Symbol, in.Depth)
	if err != nil {
		return toolError[codemap.ImpactResult](err.Error())
	}
	return nil, result, nil
}

// CodemapSemanticInput runs semantic search across the code graph.
type CodemapSemanticInput struct {
	Query string `json:"query" jsonschema:"natural-language description of the code to find"`
	TopK  int    `json:"top_k,omitempty" jsonschema:"maximum results (default 10)"`
}

func codemapSemanticTool(ctx context.Context, _ *mcp.CallToolRequest, in CodemapSemanticInput) (*mcp.CallToolResult, codemap.SemanticResult, error) {
	if strings.TrimSpace(in.Query) == "" {
		return toolError[codemap.SemanticResult]("query is required")
	}
	if !codemap.Available() {
		return toolError[codemap.SemanticResult]("codemap is not installed or not on PATH")
	}
	result, err := codemap.Semantic(ctx, in.Query, in.TopK)
	if err != nil {
		return toolError[codemap.SemanticResult](err.Error())
	}
	return nil, result, nil
}

// CodemapFindInput finds symbols by name.
type CodemapFindInput struct {
	Query string `json:"query" jsonschema:"substring to match against symbol names and FQNs"`
	TopK  int    `json:"top_k,omitempty" jsonschema:"maximum results (default 10)"`
}

func codemapFindTool(ctx context.Context, _ *mcp.CallToolRequest, in CodemapFindInput) (*mcp.CallToolResult, codemap.FindResult, error) {
	if strings.TrimSpace(in.Query) == "" {
		return toolError[codemap.FindResult]("query is required")
	}
	if !codemap.Available() {
		return toolError[codemap.FindResult]("codemap is not installed or not on PATH")
	}
	result, err := codemap.Find(ctx, in.Query, in.TopK)
	if err != nil {
		return toolError[codemap.FindResult](err.Error())
	}
	return nil, result, nil
}

// CodemapContextInput gathers everything about a symbol in one call.
type CodemapContextInput struct {
	Symbol string `json:"symbol" jsonschema:"the symbol to gather full context for"`
	Depth  int    `json:"depth,omitempty" jsonschema:"max hops for the blast-radius count (default 3)"`
}

func codemapContextTool(ctx context.Context, _ *mcp.CallToolRequest, in CodemapContextInput) (*mcp.CallToolResult, codemap.ContextResult, error) {
	if strings.TrimSpace(in.Symbol) == "" {
		return toolError[codemap.ContextResult]("symbol is required")
	}
	if !codemap.Available() {
		return toolError[codemap.ContextResult]("codemap is not installed or not on PATH")
	}
	result, err := codemap.Context(ctx, in.Symbol, in.Depth)
	if err != nil {
		return toolError[codemap.ContextResult](err.Error())
	}
	return nil, result, nil
}

// toolError reports a tool-level failure to the client (visible to the model)
// instead of a protocol error, with a zero structured result.
func toolError[T any](message string) (*mcp.CallToolResult, T, error) {
	var zero T
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
	}, zero, nil
}
