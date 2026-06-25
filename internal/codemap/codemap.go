// Package codemap wraps the codemap CLI so vidtrace can resolve file:line
// positions to code symbols, expand callers and blast radius, run semantic
// code search, and pin vidtrace evidence findings as persistent annotations.
// It mirrors how internal/fcheap, internal/ffmpeg, internal/tesseract, and
// internal/whisper wrap external CLI tools.
package codemap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Available reports whether the codemap binary is on PATH.
func Available() bool {
	_, err := exec.LookPath("codemap")
	return err == nil
}

// SymbolAtResult is the JSON returned by `codemap symbol-at <file>:<line> --json`.
type SymbolAtResult struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	Symbol     string `json:"symbol,omitempty"`
	FQN        string `json:"fqn,omitempty"`
	Kind       string `json:"kind,omitempty"`
	StartLine  int    `json:"start_line,omitempty"`
	EndLine    int    `json:"end_line,omitempty"`
	Resolution string `json:"resolution"`
}

// Symbol is a code symbol returned by callers, callees, find, and semantic.
type Symbol struct {
	Symbol    string  `json:"symbol"`
	FQN       string  `json:"fqn,omitempty"`
	Kind      string  `json:"kind,omitempty"`
	File      string  `json:"file,omitempty"`
	StartLine int     `json:"start_line,omitempty"`
	EndLine   int     `json:"end_line,omitempty"`
	Score     float64 `json:"score,omitempty"`
	Signature string  `json:"signature,omitempty"`
	Doc       string  `json:"doc,omitempty"`
	Depth     int     `json:"depth,omitempty"`
}

// CallersResult is the JSON returned by `codemap callers <symbol> --json`.
type CallersResult struct {
	Symbol  string   `json:"symbol"`
	Project string   `json:"project,omitempty"`
	Found   bool     `json:"found"`
	Results []Symbol `json:"results,omitempty"`
}

// CalleesResult is the JSON returned by `codemap callees <symbol> --json`.
type CalleesResult struct {
	Symbol  string   `json:"symbol"`
	Project string   `json:"project,omitempty"`
	Found   bool     `json:"found"`
	Results []Symbol `json:"results,omitempty"`
}

// ImpactResult is the JSON returned by `codemap impact <symbol> --json`.
type ImpactResult struct {
	Symbol        string   `json:"symbol"`
	Project       string   `json:"project,omitempty"`
	Found         bool     `json:"found"`
	Locations     []Symbol `json:"locations,omitempty"`
	DirectCallers []Symbol `json:"direct_callers,omitempty"`
	BlastRadius   []Symbol `json:"blast_radius,omitempty"`
	Tested        bool     `json:"tested"`
	Untested      bool     `json:"untested,omitempty"`
}

// SemanticResult is the JSON returned by `codemap semantic <query> --json`.
type SemanticResult struct {
	Query   string   `json:"query"`
	Project string   `json:"project,omitempty"`
	Mode    string   `json:"mode,omitempty"`
	Hits    []Symbol `json:"hits,omitempty"`
}

// FindResult is the JSON returned by `codemap find <query> --json`.
type FindResult struct {
	Query   string   `json:"query"`
	Project string   `json:"project,omitempty"`
	Mode    string   `json:"mode,omitempty"`
	Hits    []Symbol `json:"hits,omitempty"`
}

// ContextResult is the JSON returned by `codemap context <symbol> --json`.
type ContextResult struct {
	Symbol      string       `json:"symbol"`
	Project     string       `json:"project,omitempty"`
	Found       bool         `json:"found"`
	Definitions []Symbol     `json:"definitions,omitempty"`
	Callers     []Symbol     `json:"callers,omitempty"`
	Callees     []Symbol     `json:"callees,omitempty"`
	Tests       []Symbol     `json:"tests,omitempty"`
	Annotations []Annotation `json:"annotations,omitempty"`
}

// SourceResult is the JSON returned by `codemap source <symbol> --json`.
type SourceResult struct {
	Symbol      string       `json:"symbol"`
	Project     string       `json:"project,omitempty"`
	Matches     []Symbol     `json:"matches,omitempty"`
	Annotations []Annotation `json:"annotations,omitempty"`
}

// Annotation is a pinned note or external data payload attached to a symbol.
type Annotation struct {
	ID        int    `json:"id"`
	Kind      string `json:"kind,omitempty"`
	Target    string `json:"target,omitempty"`
	Source    string `json:"source,omitempty"`
	Note      string `json:"note,omitempty"`
	Data      string `json:"data,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// AnnotateResult is the JSON returned by `codemap annotate <symbol> --json`.
type AnnotateResult struct {
	ID      int    `json:"id"`
	Kind    string `json:"kind,omitempty"`
	Matched bool   `json:"matched"`
	Source  string `json:"source,omitempty"`
	Target  string `json:"target,omitempty"`
}

// NotIndexedResult is the JSON returned when the project has not been indexed.
type NotIndexedResult struct {
	Indexed bool   `json:"indexed"`
	Note    string `json:"note,omitempty"`
	Project string `json:"project,omitempty"`
}

// SymbolAt resolves a file:line position to its enclosing symbol.
func SymbolAt(ctx context.Context, file string, line int) (SymbolAtResult, error) {
	pos := fmt.Sprintf("%s:%d", file, line)
	output, err := run(ctx, "symbol-at", pos, "--json")
	if err != nil {
		return SymbolAtResult{}, err
	}
	return decodeJSON[SymbolAtResult](output, "symbol-at")
}

// Callers lists functions/methods that call the given symbol.
func Callers(ctx context.Context, symbol string, precise bool) (CallersResult, error) {
	args := []string{"callers", symbol, "--json"}
	if precise {
		args = append(args, "--lsp")
	}
	output, err := run(ctx, args...)
	if err != nil {
		return CallersResult{}, err
	}
	return decodeJSON[CallersResult](output, "callers")
}

// Callees lists functions/methods that the given symbol calls.
func Callees(ctx context.Context, symbol string, precise bool) (CalleesResult, error) {
	args := []string{"callees", symbol, "--json"}
	if precise {
		args = append(args, "--lsp")
	}
	output, err := run(ctx, args...)
	if err != nil {
		return CalleesResult{}, err
	}
	return decodeJSON[CalleesResult](output, "callees")
}

// Impact returns the blast radius (transitive callers) and test coverage for a symbol.
func Impact(ctx context.Context, symbol string, depth int) (ImpactResult, error) {
	args := []string{"impact", symbol, "--json"}
	if depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", depth))
	}
	output, err := run(ctx, args...)
	if err != nil {
		return ImpactResult{}, err
	}
	return decodeJSON[ImpactResult](output, "impact")
}

// ImpactAt resolves a file:line position to a symbol and returns its impact.
func ImpactAt(ctx context.Context, file string, line int, depth int) (ImpactResult, error) {
	pos := fmt.Sprintf("%s:%d", file, line)
	args := []string{"impact", "--at", pos, "--json"}
	if depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", depth))
	}
	output, err := run(ctx, args...)
	if err != nil {
		return ImpactResult{}, err
	}
	return decodeJSON[ImpactResult](output, "impact")
}

// Semantic runs semantic search across the code graph by meaning.
func Semantic(ctx context.Context, query string, topK int) (SemanticResult, error) {
	args := []string{"semantic", query, "--json"}
	if topK > 0 {
		args = append(args, "--top", fmt.Sprintf("%d", topK))
	}
	output, err := run(ctx, args...)
	if err != nil {
		return SemanticResult{}, err
	}
	return decodeJSON[SemanticResult](output, "semantic")
}

// Find finds symbols by name (fast, offline — no embeddings needed).
func Find(ctx context.Context, query string, topK int) (FindResult, error) {
	args := []string{"find", query, "--json"}
	if topK > 0 {
		args = append(args, "--top", fmt.Sprintf("%d", topK))
	}
	output, err := run(ctx, args...)
	if err != nil {
		return FindResult{}, err
	}
	return decodeJSON[FindResult](output, "find")
}

// Context returns everything about a symbol in one call: definition, callers,
// callees, tests, and annotations.
func Context(ctx context.Context, symbol string, depth int) (ContextResult, error) {
	args := []string{"context", symbol, "--json"}
	if depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", depth))
	}
	output, err := run(ctx, args...)
	if err != nil {
		return ContextResult{}, err
	}
	return decodeJSON[ContextResult](output, "context")
}

// Source returns a symbol's source code (the implementation behind its signature).
func Source(ctx context.Context, symbol string) (SourceResult, error) {
	output, err := run(ctx, "source", symbol, "--json")
	if err != nil {
		return SourceResult{}, err
	}
	return decodeJSON[SourceResult](output, "source")
}

// AnnotateOptions controls the codemap annotate invocation.
type AnnotateOptions struct {
	Symbol string
	Note   string
	Source string
	Data   string
}

// Annotate pins a note and/or external data to a code symbol. The annotation
// persists across reindex and branch switches. The Source field should be
// "vidtrace" so annotations are traceable back to vidtrace evidence bundles.
func Annotate(ctx context.Context, opts AnnotateOptions) (AnnotateResult, error) {
	args := []string{"annotate", opts.Symbol, "--json"}
	if opts.Note != "" {
		args = append(args, "--note", opts.Note)
	}
	if opts.Source != "" {
		args = append(args, "--source", opts.Source)
	}
	if opts.Data != "" {
		args = append(args, "--data", opts.Data)
	}
	output, err := run(ctx, args...)
	if err != nil {
		return AnnotateResult{}, err
	}
	return decodeJSON[AnnotateResult](output, "annotate")
}

// run executes the codemap CLI and returns stdout. It wraps stderr in the error
// message on failure, matching the pattern in internal/fcheap and internal/ffmpeg.
func run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "codemap", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg != "" {
			return nil, fmt.Errorf("codemap %s: %w: %s", args[0], err, msg)
		}
		return nil, fmt.Errorf("codemap %s: %w", args[0], err)
	}
	return stdout.Bytes(), nil
}

func decodeJSON[T any](output []byte, command string) (T, error) {
	var result T
	if len(output) == 0 {
		return result, fmt.Errorf("codemap %s: empty output", command)
	}
	trimmed := bytes.TrimSpace(output)
	if err := json.Unmarshal(trimmed, &result); err != nil {
		return result, fmt.Errorf("parse codemap %s json: %w", command, err)
	}
	return result, nil
}
