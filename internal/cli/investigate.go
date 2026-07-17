package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/abdul-hamid-achik/vidtrace/internal/embed"
	"github.com/abdul-hamid-achik/vidtrace/internal/investigate"
)

func runInvestigate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("investigate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	query := fs.String("query", "", "bug or evidence query")
	dbPath := fs.String("db", "", "optional evidence database path")
	codebaseDir := fs.String("codebase", "", "optional codebase path for vecgrep command suggestions")
	limit := fs.Int("limit", 5, "maximum evidence results")
	connect := fs.Bool("connect", false, "run fcheap connect to find real code matches in the codebase")
	stashID := fs.String("stash", "", "fcheap stash ID to restore and investigate instead of a local bundle")
	connectMode := fs.String("connect-mode", "", "vecgrep search mode for --connect: semantic, keyword, or hybrid")
	connectLimit := fs.Int("connect-limit", 10, "maximum code matches from --connect")
	codemap := fs.Bool("codemap", false, "run codemap expansion after --connect to resolve symbols, callers, and blast radius")
	codemapDepth := fs.Int("codemap-depth", 3, "max hops for codemap blast radius")
	codemapAnnotate := fs.Bool("codemap-annotate", false, "pin vidtrace evidence findings to resolved codemap symbols")
	videoPath := fs.String("video", "", "extract this video first, then investigate the resulting bundle (one-shot)")
	extractOut := fs.String("extract-out", "", "parent output directory for --video extraction")
	extractName := fs.String("extract-name", "", "bundle name prefix for --video extraction")
	extractFPS := fs.Float64("extract-fps", 1, "frame extraction rate for --video")
	mode := fs.String("mode", "keyword", "evidence search mode: keyword, semantic, or hybrid")
	embedProvider := fs.String("embed", "", "embedding provider for semantic/hybrid mode (e.g. ollama)")
	embedModel := fs.String("embed-model", "", "embedding model name for the provider")
	ollamaURL := fs.String("ollama-url", "", "Ollama base URL (default http://localhost:11434)")
	format := fs.String("format", "markdown", "human output format: markdown or github-issue (ignored with --json)")
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")

	jsonWanted := jsonFlagRequested(args)
	normalizedArgs, err := normalizeBundleArgs(args, map[string]struct{}{
		"json":             {},
		"connect":          {},
		"codemap":          {},
		"codemap-annotate": {},
	}, map[string]struct{}{
		"query":         {},
		"db":            {},
		"codebase":      {},
		"limit":         {},
		"stash":         {},
		"connect-mode":  {},
		"connect-limit": {},
		"codemap-depth": {},
		"video":         {},
		"extract-out":   {},
		"extract-name":  {},
		"extract-fps":   {},
		"mode":          {},
		"embed":         {},
		"embed-model":   {},
		"ollama-url":    {},
		"format":        {},
	})
	if err != nil {
		return writeUsageError(stdout, stderr, jsonWanted, err.Error())
	}
	if err := parseFlagsJSON(fs, normalizedArgs, jsonWanted); err != nil {
		return writeUsageError(stdout, stderr, jsonWanted, err.Error())
	}
	if strings.TrimSpace(*query) == "" {
		return writeUsageError(stdout, stderr, *jsonOutput, "missing required --query")
	}

	if *connect && strings.TrimSpace(*codebaseDir) == "" {
		return writeUsageError(stdout, stderr, *jsonOutput, "--connect requires --codebase")
	}

	if *codemap && !*connect {
		return writeUsageError(stdout, stderr, *jsonOutput, "--codemap requires --connect")
	}

	resolvedVideo := strings.TrimSpace(*videoPath)
	resolvedStashID := strings.TrimSpace(*stashID)
	bundleDir := ""
	if fs.NArg() == 1 {
		if resolvedVideo != "" {
			return writeUsageError(stdout, stderr, *jsonOutput, "--video cannot be combined with a bundle path")
		}
		bundleDir, err = expandHome(fs.Arg(0))
		if err != nil {
			return writeInvestigateFailure(stdout, stderr, *jsonOutput, fmt.Errorf("resolve bundle path: %w", err))
		}
	} else if resolvedStashID == "" && resolvedVideo == "" {
		return writeUsageError(stdout, stderr, *jsonOutput, "usage: vidtrace investigate [/path/to/bundle] --query TEXT [--video VIDEO] [--codebase /path/to/repo] [--connect] [--codemap] [--stash ID] [--mode keyword|semantic|hybrid] [--format markdown|github-issue] [--json]")
	}

	if resolvedVideo != "" {
		resolvedVideo, err = expandHome(resolvedVideo)
		if err != nil {
			return writeInvestigateFailure(stdout, stderr, *jsonOutput, fmt.Errorf("resolve video path: %w", err))
		}
	}

	resolvedDBPath := strings.TrimSpace(*dbPath)
	if resolvedDBPath != "" {
		resolvedDBPath, err = expandHome(resolvedDBPath)
		if err != nil {
			return writeInvestigateFailure(stdout, stderr, *jsonOutput, fmt.Errorf("resolve db path: %w", err))
		}
	}
	resolvedCodebase := strings.TrimSpace(*codebaseDir)
	if resolvedCodebase != "" {
		resolvedCodebase, err = expandHome(resolvedCodebase)
		if err != nil {
			return writeInvestigateFailure(stdout, stderr, *jsonOutput, fmt.Errorf("resolve codebase path: %w", err))
		}
	}
	resolvedExtractOut := strings.TrimSpace(*extractOut)
	if resolvedExtractOut != "" {
		resolvedExtractOut, err = expandHome(resolvedExtractOut)
		if err != nil {
			return writeInvestigateFailure(stdout, stderr, *jsonOutput, fmt.Errorf("resolve extract-out path: %w", err))
		}
	}

	embedder, err := embed.Build(*embedProvider, *embedModel, *ollamaURL)
	if err != nil {
		return writeInvestigateFailure(stdout, stderr, *jsonOutput, err)
	}

	report, err := investigate.Run(investigate.Options{
		BundleDir:       bundleDir,
		Query:           *query,
		DBPath:          resolvedDBPath,
		CodebaseDir:     resolvedCodebase,
		Limit:           *limit,
		Connect:         *connect,
		StashID:         resolvedStashID,
		ConnectMode:     strings.TrimSpace(*connectMode),
		ConnectLimit:    *connectLimit,
		Codemap:         *codemap,
		CodemapDepth:    *codemapDepth,
		CodemapAnnotate: *codemapAnnotate,
		VideoPath:       resolvedVideo,
		ExtractOut:      resolvedExtractOut,
		ExtractName:     strings.TrimSpace(*extractName),
		ExtractFPS:      *extractFPS,
		Mode:            strings.TrimSpace(*mode),
		Embedder:        embedder,
	})
	if err != nil {
		return writeInvestigateFailure(stdout, stderr, *jsonOutput, err)
	}

	if *jsonOutput {
		if err := writeJSON(stdout, report); err != nil {
			_, _ = fmt.Fprintf(stderr, "investigate json failed: %v\n", err)
			return 1
		}
		return 0
	}
	_, _ = fmt.Fprint(stdout, investigate.FormatMarkdown(report, *format))
	return 0
}

func writeInvestigateFailure(stdout, stderr io.Writer, jsonOutput bool, err error) int {
	if jsonOutput {
		_ = writeJSON(stdout, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
	} else {
		_, _ = fmt.Fprintf(stderr, "investigate failed: %v\n", err)
	}
	return 1
}
