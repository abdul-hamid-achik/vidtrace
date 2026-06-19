package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/abdul-hamid-achik/vidtrace/internal/bundle"
)

func runValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")

	normalizedArgs, err := normalizeBundleArgs(args, map[string]struct{}{"json": {}}, nil)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	if err := fs.Parse(normalizedArgs); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: vidtrace validate [flags] /path/to/bundle")
		return 2
	}

	report := bundle.Validate(fs.Arg(0))
	if *jsonOutput {
		if err := writeJSON(stdout, report); err != nil {
			_, _ = fmt.Fprintf(stderr, "validate json failed: %v\n", err)
			return 1
		}
	} else {
		printValidationReport(stdout, report)
	}

	if !report.OK {
		return 1
	}
	return 0
}

func printValidationReport(w io.Writer, report bundle.ValidationReport) {
	status := "ok"
	if !report.OK {
		status = "failed"
	}
	_, _ = fmt.Fprintf(w, "vidtrace validate: %s\n", status)
	if report.BundleDir != "" {
		_, _ = fmt.Fprintf(w, "Bundle: %s\n", report.BundleDir)
	}
	_, _ = fmt.Fprintf(w, "Summary: %s\n", report.Summary)
	_, _ = fmt.Fprintf(w, "Timeline entries: %d\n", report.TimelineEntries)
	_, _ = fmt.Fprintf(w, "Empty OCR entries: %d\n", report.EmptyOCREntries)
	_, _ = fmt.Fprintln(w, "Checks:")
	for _, check := range report.Checks {
		marker := "ok"
		if !check.OK {
			marker = "fail"
		}
		if check.Path == "" {
			_, _ = fmt.Fprintf(w, "  - [%s] %s: %s\n", marker, check.Name, check.Message)
		} else {
			_, _ = fmt.Fprintf(w, "  - [%s] %s (%s): %s\n", marker, check.Name, check.Path, check.Message)
		}
	}
}
