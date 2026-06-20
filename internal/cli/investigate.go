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
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")

	normalizedArgs, err := normalizeBundleArgs(args, map[string]struct{}{"json": {}}, map[string]struct{}{
		"query":    {},
		"db":       {},
		"codebase": {},
		"limit":    {},
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: vidtrace investigate /path/to/bundle --query TEXT [--codebase /path/to/repo] [--json]")
		return 2
	}
	if strings.TrimSpace(*query) == "" {
		_, _ = fmt.Fprintln(stderr, "missing required --query")
		return 2
	}

	bundleDir, err := expandHome(fs.Arg(0))
	if err != nil {
		return writeInvestigateFailure(stdout, stderr, *jsonOutput, fmt.Errorf("resolve bundle path: %w", err))
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
		BundleDir:   bundleDir,
		Query:       *query,
		DBPath:      resolvedDBPath,
		CodebaseDir: resolvedCodebase,
		Limit:       *limit,
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
