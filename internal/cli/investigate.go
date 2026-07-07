package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

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
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")

	jsonWanted := jsonFlagRequested(args)
	normalizedArgs, err := normalizeBundleArgs(args, map[string]struct{}{"json": {}, "connect": {}}, map[string]struct{}{
		"query":         {},
		"db":            {},
		"codebase":      {},
		"limit":         {},
		"stash":         {},
		"connect-mode":  {},
		"connect-limit": {},
		"codemap-depth": {},
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

	resolvedStashID := strings.TrimSpace(*stashID)
	bundleDir := ""
	if fs.NArg() == 1 {
		bundleDir, err = expandHome(fs.Arg(0))
		if err != nil {
			return writeInvestigateFailure(stdout, stderr, *jsonOutput, fmt.Errorf("resolve bundle path: %w", err))
		}
	} else if resolvedStashID == "" {
		return writeUsageError(stdout, stderr, *jsonOutput, "usage: vidtrace investigate /path/to/bundle --query TEXT [--codebase /path/to/repo] [--connect] [--codemap] [--stash ID] [--json]")
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
	_, _ = fmt.Fprint(stdout, investigate.Markdown(report))
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
