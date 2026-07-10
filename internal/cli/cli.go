package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/charmbracelet/x/term"

	"github.com/abdul-hamid-achik/vidtrace/internal/doctor"
	"github.com/abdul-hamid-achik/vidtrace/internal/pipeline"
	"github.com/abdul-hamid-achik/vidtrace/internal/studio"
)

func Run(args []string, stdout, stderr io.Writer, version string) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp(stdout)
		return 0
	case "version", "--version":
		_, _ = fmt.Fprintf(stdout, "vidtrace %s\n", version)
		return 0
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	case "extract":
		return runExtract(args[1:], stdout, stderr)
	case "index":
		return runIndex(args[1:], stdout, stderr)
	case "migrate-evidence":
		return runMigrateEvidence(args[1:], stdout, stderr)
	case "search":
		return runSearch(args[1:], stdout, stderr)
	case "investigate":
		return runInvestigate(args[1:], stdout, stderr)
	case "analyze":
		return runAnalyze(args[1:], stdout, stderr)
	case "compare":
		return runCompare(args[1:], stdout, stderr)
	case "clip":
		return runClip(args[1:], stdout, stderr)
	case "stash":
		return runStash(args[1:], stdout, stderr)
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "mcp":
		return runMCP(args[1:], stderr, version)
	case "docs":
		return runDocs(args[1:], stdout, stderr)
	case "studio":
		if len(args[1:]) > 1 {
			_, _ = fmt.Fprintln(stderr, "usage: vidtrace studio [bundle]")
			return 2
		}
		bundleDir := ""
		if len(args[1:]) == 1 {
			bundleDir = args[1]
		}
		if err := studio.Run(bundleDir); err != nil {
			_, _ = fmt.Fprintf(stderr, "studio failed: %v\n", err)
			return 1
		}
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printHelp(stderr)
		return 2
	}
}

func runDoctor(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")

	if err := parseFlagsJSON(fs, args, jsonFlagRequested(args)); err != nil {
		return writeUsageError(stdout, stderr, jsonFlagRequested(args), err.Error())
	}

	result := doctor.Check()
	if *jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			_, _ = fmt.Fprintf(stderr, "doctor json failed: %v\n", err)
			return 1
		}
	} else {
		doctor.PrintHuman(stdout, result)
	}

	if !result.OK {
		return 1
	}
	return 0
}

func runExtract(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("extract", flag.ContinueOnError)
	fs.SetOutput(stderr)

	fps := fs.Float64("fps", 1, "frame extraction rate")
	ocrLanguage := fs.String("ocr-lang", "eng", "Tesseract OCR language list")
	whisperLanguage := fs.String("whisper-lang", "en", "Whisper language")
	whisperModel := fs.String("model", "small", "Whisper model")
	outputDir := fs.String("out", defaultOutputDir(), "parent output directory")
	bundleName := fs.String("name", "", "artifact bundle name prefix")
	concurrency := fs.Int("concurrency", 0, "parallel OCR workers (0 = auto, capped to 8)")
	resume := fs.Bool("resume", false, "skip already-completed stages (frames/OCR/transcript) (SPEC §8.4)")
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")

	jsonWanted := jsonFlagRequested(args)
	normalizedArgs, err := normalizeExtractArgs(args)
	if err != nil {
		return writeUsageError(stdout, stderr, jsonWanted, err.Error())
	}

	if err := parseFlagsJSON(fs, normalizedArgs, jsonWanted); err != nil {
		return writeUsageError(stdout, stderr, jsonWanted, err.Error())
	}
	if fs.NArg() != 1 {
		return writeUsageError(stdout, stderr, *jsonOutput, "usage: vidtrace extract [flags] /path/to/video.mp4")
	}

	resolvedOutputDir, err := expandHome(*outputDir)
	if err != nil {
		return writeExtractFailure(stdout, stderr, *jsonOutput, err)
	}

	progress := stdout
	interactive := false
	if *jsonOutput {
		progress = nil
	} else if f, ok := stdout.(*os.File); ok {
		// Only draw a live progress bar when stdout is a real terminal; piped or
		// captured output stays plain and parseable.
		interactive = term.IsTerminal(f.Fd())
	}

	summary, err := runExtractWithCancellation(fs.Arg(0), pipeline.Options{
		SourceVideo:     fs.Arg(0),
		FPS:             *fps,
		OCRLanguage:     *ocrLanguage,
		WhisperLanguage: *whisperLanguage,
		WhisperModel:    *whisperModel,
		OutputParentDir: resolvedOutputDir,
		BundleName:      *bundleName,
		Progress:        progress,
		Interactive:     interactive,
		Concurrency:     *concurrency,
		Resume:          *resume,
	})
	if err != nil {
		return writeExtractFailure(stdout, stderr, *jsonOutput, err)
	}

	if *jsonOutput {
		if err := writeJSON(stdout, summary); err != nil {
			_, _ = fmt.Fprintf(stderr, "extract json failed: %v\n", err)
			return 1
		}
	} else {
		pipeline.PrintHuman(stdout, summary)
	}
	return 0
}

func normalizeExtractArgs(args []string) ([]string, error) {
	boolFlags := map[string]struct{}{
		"json":   {},
		"resume": {},
	}
	valueFlags := map[string]struct{}{
		"fps":          {},
		"ocr-lang":     {},
		"whisper-lang": {},
		"model":        {},
		"out":          {},
		"name":         {},
	}

	var flags []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}

		name := strings.TrimLeft(arg, "-")
		if before, _, ok := strings.Cut(name, "="); ok {
			name = before
		}

		if _, ok := boolFlags[name]; ok {
			flags = append(flags, arg)
			continue
		}

		if _, ok := valueFlags[name]; ok {
			flags = append(flags, arg)
			if !strings.Contains(arg, "=") {
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

func writeExtractFailure(stdout, stderr io.Writer, jsonOutput bool, err error) int {
	if jsonOutput {
		_ = writeJSON(stdout, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
	} else {
		_, _ = fmt.Fprintf(stderr, "extract failed: %v\n", err)
	}
	return 1
}

// runExtractWithCancellation runs the pipeline with a context that cancels on
// SIGINT/SIGTERM so long ffmpeg/whisper runs can be interrupted cleanly. The
// first returned error from the pipeline wins; a cancellation error is reported
// as a normal extract failure so --json output stays stable.
func runExtractWithCancellation(videoPath string, opts pipeline.Options) (pipeline.Summary, error) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return pipeline.Run(ctx, opts)
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

// jsonFlagRequested reports whether the args set the -json/--json flag. It
// scans args directly so it works even when flag parsing fails before the
// -json flag is bound (for example an unknown flag appearing before -json).
// A bare -json or -json=true counts; -json=false does not.
func jsonFlagRequested(args []string) bool {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") || a == "-" {
			continue
		}
		name := strings.TrimLeft(a, "-")
		if before, _, ok := strings.Cut(name, "="); ok {
			if before == "json" {
				return !strings.HasSuffix(a, "=false")
			}
			continue
		}
		if name == "json" {
			return true
		}
	}
	return false
}

// writeUsageError emits a usage/validation error in the format matching the
// active output mode and returns exit code 2. When jsonOutput is set the
// error is written to stdout as {"ok":false,"error":<msg>} (never bare
// text under -json); otherwise it is written to stderr as plain text.
func writeUsageError(stdout, stderr io.Writer, jsonOutput bool, message string) int {
	if jsonOutput {
		_ = writeJSON(stdout, map[string]any{"ok": false, "error": message})
	} else {
		_, _ = fmt.Fprintln(stderr, message)
	}
	return 2
}

// parseFlagsJSON parses fs against args. When jsonOutput is requested it
// suppresses the flag package's own stderr usage output so the caller can
// emit a structured {ok:false,error} via writeUsageError instead. Returns
// the parse error (never prints on its own when jsonOutput is set).
func parseFlagsJSON(fs *flag.FlagSet, args []string, jsonOutput bool) error {
	if jsonOutput {
		fs.SetOutput(io.Discard)
	}
	return fs.Parse(args)
}

func defaultOutputDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, "Downloads")
}

func expandHome(path string) (string, error) {
	if path == "" {
		return ".", nil
	}
	if path == "~" {
		return os.UserHomeDir()
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
	}
	return path, nil
}

func printHelp(w io.Writer) {
	_, _ = fmt.Fprint(w, `vidtrace turns bug videos into agent-readable evidence.

Usage:
  vidtrace <command> [flags]

Commands:
  analyze      Write a Markdown evidence report for a bundle and ticket
  clip         Cut clips, make GIFs, or stitch videos from timestamp ranges
  compare      Compare a ticket with bundle evidence, optionally as JSON
  doctor       Check required local tools: ffmpeg, ffprobe, tesseract, whisper
  docs         Print built-in product and agent usage docs
  extract      Extract frames, OCR, transcript, metadata, and timeline artifacts
  index        Index bundle evidence into a local VecLite database
  investigate  Create a video-evidence to code-search handoff
  mcp          Run the MCP server (read-only evidence tools over stdio)
  migrate-evidence  Convert a pre-v0.17.0 evidence DB to the single-collection layout
  search       Search an evidence database for timestamped video evidence
  stash        Save, list, restore, inspect, or search artifact bundles in the fcheap vault
  studio       Open the artifact inspection studio
  validate     Validate an artifact bundle, optionally as JSON
  version      Print the CLI version
  help         Show this help

Examples:
  vidtrace doctor
  vidtrace doctor -json
  vidtrace docs agent
  vidtrace extract /path/to/bug.mp4
  vidtrace extract /path/to/bug.mp4 -json
  vidtrace index /path/to/bundle --db /tmp/evidence.veclite --json
  vidtrace search /tmp/evidence.veclite "ticket click does not work" --json
  vidtrace investigate /path/to/bundle --query "ticket click does not work" --codebase /path/to/app
  vidtrace clip cut ~/Downloads/bug.mp4 --label "issue1=0:18-3:40" --json
  vidtrace stash save /path/to/bundle --name "bug" --tag bug
  vidtrace stash list --tool vidtrace --json
  vidtrace analyze /path/to/bundle --ticket ticket.md
  vidtrace compare /path/to/bundle --ticket ticket.md --json
  vidtrace validate /path/to/bundle --json
  vidtrace studio
`)
}
