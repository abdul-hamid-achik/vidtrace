package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/abdul-hamid-achik/vidtrace/internal/evidence"
)

func runIndex(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("index", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "", "evidence database path")
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")

	normalizedArgs, err := normalizeBundleArgs(args, map[string]struct{}{"json": {}}, map[string]struct{}{"db": {}})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: vidtrace index /path/to/bundle --db /path/to/evidence.veclite [--json]")
		return 2
	}
	if strings.TrimSpace(*dbPath) == "" {
		_, _ = fmt.Fprintln(stderr, "missing required --db")
		return 2
	}

	resolvedDBPath, err := expandHome(*dbPath)
	if err != nil {
		return writeEvidenceFailure(stdout, stderr, *jsonOutput, fmt.Errorf("resolve db path: %w", err), "index")
	}
	resolvedBundlePath, err := expandHome(fs.Arg(0))
	if err != nil {
		return writeEvidenceFailure(stdout, stderr, *jsonOutput, fmt.Errorf("resolve bundle path: %w", err), "index")
	}

	report, err := evidence.IndexBundle(evidence.IndexOptions{
		BundleDir: resolvedBundlePath,
		DBPath:    resolvedDBPath,
	})
	if err != nil {
		return writeEvidenceFailure(stdout, stderr, *jsonOutput, err, "index")
	}

	if *jsonOutput {
		if err := writeJSON(stdout, report); err != nil {
			_, _ = fmt.Fprintf(stderr, "index json failed: %v\n", err)
			return 1
		}
		return 0
	}

	_, _ = fmt.Fprintln(stdout, "vidtrace index: ok")
	_, _ = fmt.Fprintf(stdout, "Bundle: %s\n", report.BundleDir)
	_, _ = fmt.Fprintf(stdout, "DB: %s\n", report.DBPath)
	_, _ = fmt.Fprintf(stdout, "Collection: %s\n", report.Collection)
	_, _ = fmt.Fprintf(stdout, "Entries: %d indexed, %d inserted, %d updated\n", report.IndexedEntries, report.InsertedEntries, report.UpdatedEntries)
	return 0
}

func runSearch(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")
	limit := fs.Int("limit", 10, "maximum results")
	bundle := fs.String("bundle", "", "filter results to a single bundle directory")
	sourceVideo := fs.String("source-video", "", "filter results by source video path")
	source := fs.String("source", "", "filter results by evidence source (e.g. timeline)")
	minTime := fs.Float64("min-time", 0, "filter results at or after this time in seconds")
	maxTime := fs.Float64("max-time", 0, "filter results at or before this time in seconds")

	normalizedArgs, err := normalizeBundleArgs(args, map[string]struct{}{"json": {}}, map[string]struct{}{
		"limit":        {},
		"bundle":       {},
		"source-video": {},
		"source":       {},
		"min-time":     {},
		"max-time":     {},
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return 2
	}
	if fs.NArg() < 2 {
		_, _ = fmt.Fprintln(stderr, "usage: vidtrace search /path/to/evidence.veclite QUERY [--limit N] [--bundle DIR] [--source-video PATH] [--source SOURCE] [--min-time SECONDS] [--max-time SECONDS] [--json]")
		return 2
	}

	resolvedDBPath, err := expandHome(fs.Arg(0))
	if err != nil {
		return writeEvidenceFailure(stdout, stderr, *jsonOutput, fmt.Errorf("resolve db path: %w", err), "search")
	}
	searchOpts := evidence.SearchOptions{
		DBPath:      resolvedDBPath,
		Query:       strings.Join(fs.Args()[1:], " "),
		Limit:       *limit,
		SourceVideo: strings.TrimSpace(*sourceVideo),
		Source:      strings.TrimSpace(*source),
	}
	if trimmed := strings.TrimSpace(*bundle); trimmed != "" {
		resolvedBundle, err := expandHome(trimmed)
		if err != nil {
			return writeEvidenceFailure(stdout, stderr, *jsonOutput, fmt.Errorf("resolve bundle filter: %w", err), "search")
		}
		searchOpts.Bundle = resolvedBundle
	}
	setFlags := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { setFlags[f.Name] = true })
	if setFlags["min-time"] {
		v := *minTime
		searchOpts.MinTime = &v
	}
	if setFlags["max-time"] {
		v := *maxTime
		searchOpts.MaxTime = &v
	}
	report, err := evidence.Search(searchOpts)
	if err != nil {
		return writeEvidenceFailure(stdout, stderr, *jsonOutput, err, "search")
	}

	if *jsonOutput {
		if err := writeJSON(stdout, report); err != nil {
			_, _ = fmt.Fprintf(stderr, "search json failed: %v\n", err)
			return 1
		}
		return 0
	}

	_, _ = fmt.Fprintf(stdout, "vidtrace search: %d result(s)\n", len(report.Results))
	for _, result := range report.Results {
		_, _ = fmt.Fprintf(stdout, "  - %.2fs %s score %.3f: %s\n", result.TimeSeconds, result.Frame, result.Score, conciseEvidenceText(result, 160))
	}
	return 0
}

func conciseEvidenceText(result evidence.SearchResult, limit int) string {
	text := result.Transcript
	if strings.TrimSpace(text) == "" {
		text = result.OCR
	}
	if strings.TrimSpace(text) == "" {
		text = result.Frame
	}
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= limit {
		return text
	}
	if limit <= 3 {
		return text[:max(0, limit)]
	}
	return text[:limit-3] + "..."
}

func writeEvidenceFailure(stdout, stderr io.Writer, jsonOutput bool, err error, command string) int {
	if jsonOutput {
		_ = writeJSON(stdout, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
	} else {
		_, _ = fmt.Fprintf(stderr, "%s failed: %v\n", command, err)
	}
	return 1
}
