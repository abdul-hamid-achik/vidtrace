package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/abdul-hamid-achik/vidtrace/internal/clip"
	"github.com/abdul-hamid-achik/vidtrace/internal/fcheap"
)

func runClip(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printClipHelp(stderr)
		return 2
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "cut":
		return runClipCut(rest, stdout, stderr)
	case "gif":
		return runClipGIF(rest, stdout, stderr)
	case "stitch":
		return runClipStitch(rest, stdout, stderr)
	case "help", "-h", "--help":
		printClipHelp(stdout)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "unknown clip subcommand: %s\n\n", sub)
		printClipHelp(stderr)
		return 2
	}
}

func runClipCut(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("clip cut", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outputDir := fs.String("out", defaultOutputDir(), "parent output directory")
	name := fs.String("name", "", "prefix for clip filenames and output directory")
	reencode := fs.Bool("reencode", false, "force re-encoding instead of stream copy")
	stash := fs.Bool("stash", false, "stash the clips directory to fcheap after cutting")
	tool := fs.String("tool", "vidtrace", "tool tag for the fcheap stash")
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")

	var ranges []string
	var labels []string
	var tags []string
	normalizedArgs, err := normalizeClipArgs(args, map[string]struct{}{"json": {}, "reencode": {}, "stash": {}}, map[string]struct{}{
		"out":  {},
		"name": {},
		"tool": {},
	}, &ranges, &labels, &tags)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: vidtrace clip cut [flags] /path/to/video.mp4")
		return 2
	}
	if len(ranges) == 0 && len(labels) == 0 {
		_, _ = fmt.Fprintln(stderr, "at least one --range or --label is required")
		return 2
	}

	specs, err := buildClipSpecs(ranges, labels)
	if err != nil {
		return writeClipFailure(stdout, stderr, *jsonOutput, err.Error())
	}

	videoPath, err := expandHome(fs.Arg(0))
	if err != nil {
		return writeClipFailure(stdout, stderr, *jsonOutput, fmt.Sprintf("resolve video path: %v", err))
	}
	if _, err := os.Stat(videoPath); err != nil {
		return writeClipFailure(stdout, stderr, *jsonOutput, fmt.Sprintf("video not found: %s", videoPath))
	}

	resolvedOutputDir, err := expandHome(*outputDir)
	if err != nil {
		return writeClipFailure(stdout, stderr, *jsonOutput, fmt.Sprintf("resolve output dir: %v", err))
	}

	namePrefix := strings.TrimSpace(*name)
	if namePrefix == "" {
		namePrefix = clipPrefixFromVideo(videoPath)
	}

	clipOutputDir, err := clip.OutputDir(resolvedOutputDir, namePrefix)
	if err != nil {
		return writeClipFailure(stdout, stderr, *jsonOutput, err.Error())
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cutReport, err := clip.CutClips(ctx, videoPath, clipOutputDir, specs, *reencode)
	if err != nil {
		return writeClipFailure(stdout, stderr, *jsonOutput, err.Error())
	}

	report := clipCutReport{
		OK:          cutReport.OK,
		SourceVideo: cutReport.SourceVideo,
		OutputDir:   cutReport.OutputDir,
		Clips:       cutReport.Clips,
	}

	stashResult := maybeStash(ctx, *stash, clipOutputDir, namePrefix+"-clips", *tool, tags)
	if stashResult != nil && stashResult.err != nil {
		report.StashError = stashResult.err.Error()
	} else if stashResult != nil {
		report.Stash = &stashInfo{ID: stashResult.id, Name: stashResult.name}
	}

	if *jsonOutput {
		_ = writeJSON(stdout, report)
	} else {
		printCutHuman(stdout, report)
	}
	return 0
}

func runClipGIF(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("clip gif", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outputDir := fs.String("out", defaultOutputDir(), "parent output directory")
	name := fs.String("name", "", "prefix for GIF filenames and output directory")
	gifFPS := fs.Int("fps", 10, "GIF frame rate")
	gifWidth := fs.Int("width", 480, "GIF width in pixels (height auto-scales)")
	stash := fs.Bool("stash", false, "stash the GIFs directory to fcheap after creating")
	tool := fs.String("tool", "vidtrace", "tool tag for the fcheap stash")
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")

	var ranges []string
	var labels []string
	var tags []string
	normalizedArgs, err := normalizeClipArgs(args, map[string]struct{}{"json": {}, "stash": {}}, map[string]struct{}{
		"out":   {},
		"name":  {},
		"fps":   {},
		"width": {},
		"tool":  {},
	}, &ranges, &labels, &tags)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: vidtrace clip gif [flags] /path/to/video.mp4")
		return 2
	}
	if len(ranges) == 0 && len(labels) == 0 {
		_, _ = fmt.Fprintln(stderr, "at least one --range or --label is required")
		return 2
	}

	specs, err := buildClipSpecs(ranges, labels)
	if err != nil {
		return writeClipFailure(stdout, stderr, *jsonOutput, err.Error())
	}

	videoPath, err := expandHome(fs.Arg(0))
	if err != nil {
		return writeClipFailure(stdout, stderr, *jsonOutput, fmt.Sprintf("resolve video path: %v", err))
	}
	if _, err := os.Stat(videoPath); err != nil {
		return writeClipFailure(stdout, stderr, *jsonOutput, fmt.Sprintf("video not found: %s", videoPath))
	}

	resolvedOutputDir, err := expandHome(*outputDir)
	if err != nil {
		return writeClipFailure(stdout, stderr, *jsonOutput, fmt.Sprintf("resolve output dir: %v", err))
	}

	namePrefix := strings.TrimSpace(*name)
	if namePrefix == "" {
		namePrefix = clipPrefixFromVideo(videoPath)
	}

	gifOutputDir, err := clip.OutputDir(resolvedOutputDir, namePrefix+"_gifs")
	if err != nil {
		return writeClipFailure(stdout, stderr, *jsonOutput, err.Error())
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	gifReport, err := clip.MakeGIFs(ctx, videoPath, gifOutputDir, specs, *gifFPS, *gifWidth)
	if err != nil {
		return writeClipFailure(stdout, stderr, *jsonOutput, err.Error())
	}

	report := clipGIFReport{
		OK:          gifReport.OK,
		SourceVideo: gifReport.SourceVideo,
		OutputDir:   gifReport.OutputDir,
		GIFs:        gifReport.GIFs,
	}

	stashResult := maybeStash(ctx, *stash, gifOutputDir, namePrefix+"-gifs", *tool, tags)
	if stashResult != nil && stashResult.err != nil {
		report.StashError = stashResult.err.Error()
	} else if stashResult != nil {
		report.Stash = &stashInfo{ID: stashResult.id, Name: stashResult.name}
	}

	if *jsonOutput {
		_ = writeJSON(stdout, report)
	} else {
		printGIFHuman(stdout, report)
	}
	return 0
}

func runClipStitch(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("clip stitch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	outputDir := fs.String("out", defaultOutputDir(), "parent output directory")
	name := fs.String("name", "stitched", "output filename (without extension)")
	jsonOutput := fs.Bool("json", false, "print machine-readable JSON")

	normalizedArgs, err := normalizeClipArgs(args, map[string]struct{}{"json": {}}, map[string]struct{}{
		"out":  {},
		"name": {},
	}, nil, nil, nil)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if err := fs.Parse(normalizedArgs); err != nil {
		return 2
	}
	if fs.NArg() < 2 {
		_, _ = fmt.Fprintln(stderr, "usage: vidtrace clip stitch [flags] clip1.mp4 clip2.mp4 [clip3.mp4 ...]")
		return 2
	}

	clipPaths := make([]string, 0, fs.NArg())
	for i := 0; i < fs.NArg(); i++ {
		p, err := expandHome(fs.Arg(i))
		if err != nil {
			return writeClipFailure(stdout, stderr, *jsonOutput, fmt.Sprintf("resolve clip path: %v", err))
		}
		if _, err := os.Stat(p); err != nil {
			return writeClipFailure(stdout, stderr, *jsonOutput, fmt.Sprintf("clip not found: %s", p))
		}
		clipPaths = append(clipPaths, p)
	}

	resolvedOutputDir, err := expandHome(*outputDir)
	if err != nil {
		return writeClipFailure(stdout, stderr, *jsonOutput, fmt.Sprintf("resolve output dir: %v", err))
	}

	stitchDir, err := clip.OutputDir(resolvedOutputDir, *name)
	if err != nil {
		return writeClipFailure(stdout, stderr, *jsonOutput, err.Error())
	}
	outputPath := filepath.Join(stitchDir, *name+".mp4")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result, err := clip.StitchClips(ctx, clipPaths, outputPath)
	if err != nil {
		return writeClipFailure(stdout, stderr, *jsonOutput, err.Error())
	}

	if *jsonOutput {
		_ = writeJSON(stdout, map[string]any{
			"ok":               true,
			"inputs":           result.Inputs,
			"output_path":      result.OutputPath,
			"duration_seconds": result.DurationSeconds,
		})
	} else {
		_, _ = fmt.Fprintf(stdout, "Stitched: %s\n", result.OutputPath)
		_, _ = fmt.Fprintf(stdout, "Inputs: %d\n", len(result.Inputs))
		if result.DurationSeconds > 0 {
			_, _ = fmt.Fprintf(stdout, "Duration: %.1fs\n", result.DurationSeconds)
		}
	}
	return 0
}

// buildClipSpecs assembles ClipSpecs from --range and --label flags. Labels
// take precedence; if both are provided, labels provide names and ranges are
// only used if there are more ranges than labels.
func buildClipSpecs(ranges, labels []string) ([]clip.ClipSpec, error) {
	var specs []clip.ClipSpec

	for _, labelStr := range labels {
		label, start, end, err := clip.ParseLabelRange(labelStr)
		if err != nil {
			return nil, err
		}
		specs = append(specs, clip.ClipSpec{Label: label, StartSec: start, EndSec: end})
	}

	// If there are more ranges than labels, the extra ranges get default labels.
	extraRanges := len(ranges)
	if len(labels) > 0 && len(ranges) > len(labels) {
		extraRanges = 0 // labels and ranges overlap; ignore plain ranges when labels exist
	}
	if len(labels) == 0 {
		extraRanges = len(ranges)
	}
	for i := 0; i < extraRanges; i++ {
		start, end, err := clip.ParseRange(ranges[i])
		if err != nil {
			return nil, err
		}
		specs = append(specs, clip.ClipSpec{Label: clip.DefaultLabel(len(specs) + 1), StartSec: start, EndSec: end})
	}

	if len(specs) == 0 {
		return nil, fmt.Errorf("no clip ranges provided; use --range or --label")
	}
	return specs, nil
}

// clipPrefixFromVideo derives a safe name prefix from a video filename.
func clipPrefixFromVideo(videoPath string) string {
	base := filepath.Base(videoPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	name = strings.ReplaceAll(name, " ", "_")
	// Strip non-alphanumeric except dash/underscore/dot.
	clean := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '_'
	}, name)
	if clean == "" {
		return "video"
	}
	return clean
}

// stashInfo is the JSON-serializable stash result included in clip reports.
type stashInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// stashOutcome holds the result of an optional fcheap stash operation.
type stashOutcome struct {
	id   string
	name string
	err  error
}

// maybeStash stashes the output directory to fcheap when doStash is true.
// Returns nil when stashing is not requested.
func maybeStash(ctx context.Context, doStash bool, dir, stashName, tool string, tags []string) *stashOutcome {
	if !doStash {
		return nil
	}
	if !fcheap.Available() {
		return &stashOutcome{err: fmt.Errorf("fcheap is not installed or not on PATH")}
	}
	stashCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	result, err := fcheap.Save(stashCtx, dir, stashName, tool, tags)
	if err != nil {
		return &stashOutcome{err: err}
	}
	return &stashOutcome{id: result.ID, name: result.Name}
}

// Extended report types that include stash info, used by the CLI layer.
// We extend the clip package types via embedding in the JSON output.

func writeClipFailure(stdout, stderr io.Writer, jsonOutput bool, message string) int {
	if jsonOutput {
		_ = writeJSON(stdout, map[string]any{
			"ok":    false,
			"error": message,
		})
	} else {
		_, _ = fmt.Fprintf(stderr, "clip failed: %s\n", message)
	}
	return 1
}

func printCutHuman(w io.Writer, report clipCutReport) {
	_, _ = fmt.Fprintf(w, "vidtrace clip cut: ok\n")
	_, _ = fmt.Fprintf(w, "Output: %s\n", report.OutputDir)
	_, _ = fmt.Fprintf(w, "Source: %s\n", report.SourceVideo)
	_, _ = fmt.Fprintf(w, "Clips: %d\n", len(report.Clips))
	for _, c := range report.Clips {
		_, _ = fmt.Fprintf(w, "  - %s (%.0fs-%.0fs, %.1fs) -> %s\n", c.Label, c.StartSeconds, c.EndSeconds, c.DurationSeconds, c.Path)
	}
	if report.Stash != nil {
		_, _ = fmt.Fprintf(w, "Stash: %s (%s)\n", report.Stash.ID, report.Stash.Name)
	}
	if report.StashError != "" {
		_, _ = fmt.Fprintf(w, "Stash error: %s\n", report.StashError)
	}
}

func printGIFHuman(w io.Writer, report clipGIFReport) {
	_, _ = fmt.Fprintf(w, "vidtrace clip gif: ok\n")
	_, _ = fmt.Fprintf(w, "Output: %s\n", report.OutputDir)
	_, _ = fmt.Fprintf(w, "Source: %s\n", report.SourceVideo)
	_, _ = fmt.Fprintf(w, "GIFs: %d\n", len(report.GIFs))
	for _, g := range report.GIFs {
		_, _ = fmt.Fprintf(w, "  - %s (%.0fs-%.0fs, %.1fs, %dfps %dpx) -> %s\n", g.Label, g.StartSeconds, g.EndSeconds, g.DurationSeconds, g.FPS, g.Width, g.Path)
	}
	if report.Stash != nil {
		_, _ = fmt.Fprintf(w, "Stash: %s (%s)\n", report.Stash.ID, report.Stash.Name)
	}
	if report.StashError != "" {
		_, _ = fmt.Fprintf(w, "Stash error: %s\n", report.StashError)
	}
}

func printClipHelp(w io.Writer) {
	_, _ = fmt.Fprint(w, `vidtrace clip - cut clips, make GIFs, or stitch videos from timestamp ranges

Usage:
  vidtrace clip <subcommand> [flags] <args>

Subcommands:
  cut      Cut one or more clips from a video at timestamp ranges
  gif      Create GIF(s) from timestamp ranges in a video
  stitch   Join multiple clips into one concatenated video
  help     Show this help

Examples:
  vidtrace clip cut ~/Downloads/bug.mp4 --range "0:18-3:40" --range "3:40-4:05" --json
  vidtrace clip cut ~/Downloads/bug.mp4 --label "issue1=0:18-3:40" --label "issue2=3:40-4:05" --stash --json
  vidtrace clip gif ~/Downloads/bug.mp4 --label "issue1=0:18-3:40" --fps 10 --width 480 --json
  vidtrace clip stitch clip1.mp4 clip2.mp4 clip3.mp4 --name summary --json

Timestamp formats:
  SS          seconds (e.g. 45)
  MM:SS       minutes and seconds (e.g. 3:40)
  HH:MM:SS    hours, minutes, seconds (e.g. 1:23:45)

Range format:
  START-END   e.g. "0:18-3:40" or "14:50-16:14"

Label format:
  LABEL=START-END  e.g. "issue1-blank-row=0:18-3:40"
`)
}

// normalizeClipArgs separates flags from positionals and collects repeated
// --range, --label, and --tag flags into their respective slices. It follows
// the normalizeStashArgs pattern.
func normalizeClipArgs(args []string, boolFlags, valueFlags map[string]struct{}, ranges, labels, tags *[]string) ([]string, error) {
	var flags []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if len(arg) == 0 || arg[0] != '-' || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}

		name := trimFlagName(arg)

		// Collect --range values.
		if name == "range" && ranges != nil {
			val, err := extractRepeatableValue(&i, arg, args)
			if err != nil {
				return nil, err
			}
			*ranges = append(*ranges, val)
			continue
		}

		// Collect --label values.
		if name == "label" && labels != nil {
			val, err := extractRepeatableValue(&i, arg, args)
			if err != nil {
				return nil, err
			}
			*labels = append(*labels, val)
			continue
		}

		// Collect --tag values.
		if name == "tag" && tags != nil {
			val, err := extractRepeatableValue(&i, arg, args)
			if err != nil {
				return nil, err
			}
			*tags = append(*tags, val)
			continue
		}

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

func extractRepeatableValue(i *int, arg string, args []string) (string, error) {
	if hasInlineValue(arg) {
		_, after, ok := strings.Cut(arg, "=")
		if ok {
			return after, nil
		}
	}
	if *i+1 >= len(args) {
		return "", fmt.Errorf("missing value for flag %s", arg)
	}
	*i++
	return args[*i], nil
}

// clipCutReport wraps clip.CutReport with stash info for JSON output.
type clipCutReport struct {
	OK          bool             `json:"ok"`
	SourceVideo string           `json:"source_video"`
	OutputDir   string           `json:"output_dir"`
	Clips       []clip.CutResult `json:"clips"`
	Stash       *stashInfo       `json:"stash,omitempty"`
	StashError  string           `json:"stash_error,omitempty"`
}

// clipGIFReport wraps clip.GIFReport with stash info for JSON output.
type clipGIFReport struct {
	OK          bool             `json:"ok"`
	SourceVideo string           `json:"source_video"`
	OutputDir   string           `json:"output_dir"`
	GIFs        []clip.GIFResult `json:"gifs"`
	Stash       *stashInfo       `json:"stash,omitempty"`
	StashError  string           `json:"stash_error,omitempty"`
}
