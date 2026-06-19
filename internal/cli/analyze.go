package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/abdul-hamid-achik/vidtrace/internal/analysis"
)

func runAnalyze(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("analyze", flag.ContinueOnError)
	fs.SetOutput(stderr)
	ticketPath := fs.String("ticket", "", "ticket markdown or text file")

	normalizedArgs, err := normalizeBundleArgs(args, map[string]struct{}{}, map[string]struct{}{"ticket": {}})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	if err := fs.Parse(normalizedArgs); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: vidtrace analyze [flags] /path/to/bundle")
		return 2
	}

	result, err := analysis.Compare(analysis.Options{
		BundleDir:  fs.Arg(0),
		TicketPath: *ticketPath,
	})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "analyze failed: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprint(stdout, analysis.Markdown(result))
	return 0
}

func runCompare(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	fs.SetOutput(stderr)
	ticketPath := fs.String("ticket", "", "ticket markdown or text file")
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")

	normalizedArgs, err := normalizeBundleArgs(args, map[string]struct{}{"json": {}}, map[string]struct{}{"ticket": {}})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	if err := fs.Parse(normalizedArgs); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: vidtrace compare [flags] /path/to/bundle")
		return 2
	}

	result, err := analysis.Compare(analysis.Options{
		BundleDir:  fs.Arg(0),
		TicketPath: *ticketPath,
	})
	if err != nil {
		if *jsonOutput {
			_ = writeJSON(stdout, map[string]any{
				"ok":    false,
				"error": err.Error(),
			})
		} else {
			_, _ = fmt.Fprintf(stderr, "compare failed: %v\n", err)
		}
		return 1
	}

	if *jsonOutput {
		if err := writeJSON(stdout, result); err != nil {
			_, _ = fmt.Fprintf(stderr, "compare json failed: %v\n", err)
			return 1
		}
		return 0
	}

	_, _ = fmt.Fprintf(stdout, "vidtrace compare: %s\n", result.Status)
	_, _ = fmt.Fprintf(stdout, "Score: %.3f\n", result.Score)
	_, _ = fmt.Fprintf(stdout, "Summary: %s\n", result.Summary)
	if len(result.Evidence) > 0 {
		_, _ = fmt.Fprintln(stdout, "Evidence:")
		for _, item := range result.Evidence {
			_, _ = fmt.Fprintf(stdout, "  - %.2fs %s: %s\n", item.TimeSeconds, item.Frame, item.Text)
		}
	}
	return 0
}

func normalizeBundleArgs(args []string, boolFlags, valueFlags map[string]struct{}) ([]string, error) {
	var flags []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if len(arg) == 0 || arg[0] != '-' || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}

		name := trimFlagName(arg)
		if _, ok := boolFlags[name]; ok {
			flags = append(flags, arg)
			continue
		}

		if _, ok := valueFlags[name]; ok {
			flags = append(flags, arg)
			if !hasInlineValue(arg) {
				if i+1 >= len(args) {
					return nil, fmt.Errorf("missing value for flag %s", arg)
				}
				i++
				flags = append(flags, args[i])
			}
			continue
		}

		flags = append(flags, arg)
	}
	return append(flags, positionals...), nil
}

func trimFlagName(arg string) string {
	name := strings.TrimLeft(arg, "-")
	if before, _, ok := strings.Cut(name, "="); ok {
		return before
	}
	return name
}

func hasInlineValue(arg string) bool {
	return strings.Contains(arg, "=")
}
