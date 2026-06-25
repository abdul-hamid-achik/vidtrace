// Package clip provides timestamp parsing, validation, and orchestration for
// cutting video clips, making GIFs, and stitching clips together. It wraps
// internal/ffmpeg for the actual media operations and internal/fcheap for
// optional stashing.
package clip

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/abdul-hamid-achik/vidtrace/internal/artifacts"
	"github.com/abdul-hamid-achik/vidtrace/internal/ffmpeg"
)

var timestampPattern = regexp.MustCompile(`^\d+(:\d+)*$`)

// ClipSpec defines one clip or GIF to produce from a video.
type ClipSpec struct {
	Label    string  // human-readable name, used in the filename
	StartSec float64 // start time in seconds
	EndSec   float64 // end time in seconds
}

// CutResult is the outcome of cutting one clip.
type CutResult struct {
	Label           string  `json:"label"`
	StartSeconds    float64 `json:"start_seconds"`
	EndSeconds      float64 `json:"end_seconds"`
	DurationSeconds float64 `json:"duration_seconds"`
	Path            string  `json:"path"`
}

// GIFResult is the outcome of making one GIF.
type GIFResult struct {
	Label           string  `json:"label"`
	StartSeconds    float64 `json:"start_seconds"`
	EndSeconds      float64 `json:"end_seconds"`
	DurationSeconds float64 `json:"duration_seconds"`
	FPS             int     `json:"fps"`
	Width           int     `json:"width"`
	Path            string  `json:"path"`
}

// StitchResult is the outcome of stitching clips together.
type StitchResult struct {
	Inputs          []string `json:"inputs"`
	OutputPath      string   `json:"output_path"`
	DurationSeconds float64  `json:"duration_seconds"`
}

// CutReport is the JSON report returned by CutClips.
type CutReport struct {
	OK          bool        `json:"ok"`
	SourceVideo string      `json:"source_video"`
	OutputDir   string      `json:"output_dir"`
	Clips       []CutResult `json:"clips"`
}

// GIFReport is the JSON report returned by MakeGIFs.
type GIFReport struct {
	OK          bool        `json:"ok"`
	SourceVideo string      `json:"source_video"`
	OutputDir   string      `json:"output_dir"`
	GIFs        []GIFResult `json:"gifs"`
}

// ParseTimestamp parses a timestamp string into seconds. Supports "SS",
// "MM:SS", and "HH:MM:SS" formats.
func ParseTimestamp(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty timestamp")
	}
	if !timestampPattern.MatchString(s) {
		return 0, fmt.Errorf("invalid timestamp %q (expected MM:SS, HH:MM:SS, or seconds)", s)
	}

	parts := strings.Split(s, ":")
	var segments []float64
	for _, part := range parts {
		value, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid timestamp segment %q in %q: %w", part, s, err)
		}
		segments = append(segments, value)
	}

	var total float64
	switch len(segments) {
	case 1:
		total = segments[0]
	case 2:
		total = segments[0]*60 + segments[1]
	case 3:
		total = segments[0]*3600 + segments[1]*60 + segments[2]
	default:
		return 0, fmt.Errorf("invalid timestamp %q: too many segments", s)
	}
	if total < 0 {
		return 0, fmt.Errorf("timestamp %q is negative", s)
	}
	return total, nil
}

// ParseRange parses a range string like "0:18-3:40" into start and end seconds.
// The separator is a hyphen. Whitespace around the range is trimmed.
func ParseRange(s string) (start, end float64, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, fmt.Errorf("empty range")
	}
	parts := strings.SplitN(s, "-", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range %q: expected START-END", s)
	}
	start, err = ParseTimestamp(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid range start in %q: %w", s, err)
	}
	end, err = ParseTimestamp(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid range end in %q: %w", s, err)
	}
	if start >= end {
		return 0, 0, fmt.Errorf("invalid range %q: start (%.3fs) must be before end (%.3fs)", s, start, end)
	}
	return start, end, nil
}

// ParseLabelRange parses "issue1=0:18-3:40" into a label and range. The label
// is everything before the first "=". If no "=" is present, label is empty.
func ParseLabelRange(s string) (label string, start, end float64, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", 0, 0, fmt.Errorf("empty label range")
	}
	idx := strings.Index(s, "=")
	if idx < 0 {
		return "", 0, 0, fmt.Errorf("invalid label range %q: expected LABEL=START-END", s)
	}
	label = strings.TrimSpace(s[:idx])
	if label == "" {
		return "", 0, 0, fmt.Errorf("invalid label range %q: empty label", s)
	}
	start, end, err = ParseRange(s[idx+1:])
	if err != nil {
		return "", 0, 0, err
	}
	return label, start, end, nil
}

// ValidateSpec checks that a clip spec has valid time bounds.
func ValidateSpec(spec ClipSpec) error {
	if spec.StartSec < 0 {
		return fmt.Errorf("clip %q: start time is negative (%.3f)", spec.Label, spec.StartSec)
	}
	if spec.EndSec <= spec.StartSec {
		return fmt.Errorf("clip %q: end time (%.3f) must be after start (%.3f)", spec.Label, spec.EndSec, spec.StartSec)
	}
	return nil
}

// SafeLabel sanitizes a label for use as a filename component.
func SafeLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}
	// Replace unsafe characters with underscores, matching artifacts.SafeBundleName.
	safe := regexp.MustCompile(`[^A-Za-z0-9._-]+`).ReplaceAllString(label, "_")
	safe = strings.Trim(safe, "._-")
	return safe
}

// DefaultLabel generates a zero-padded default label for a clip index.
func DefaultLabel(index int) string {
	return fmt.Sprintf("clip_%02d", index)
}

// CutClips cuts multiple clips from a video into outputDir. Each clip gets a
// filename derived from its label. When reencode is false, stream copy is used
// for speed; when true, clips are re-encoded. A clips.json manifest is written
// into the output directory.
func CutClips(ctx context.Context, videoPath, outputDir string, specs []ClipSpec, reencode bool) (CutReport, error) {
	if len(specs) == 0 {
		return CutReport{}, fmt.Errorf("at least one clip spec is required")
	}
	for i, spec := range specs {
		if spec.Label == "" {
			specs[i].Label = DefaultLabel(i + 1)
		}
		if err := ValidateSpec(spec); err != nil {
			return CutReport{}, err
		}
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return CutReport{}, fmt.Errorf("create clip output directory: %w", err)
	}

	results := make([]CutResult, 0, len(specs))
	for _, spec := range specs {
		filename := SafeLabel(spec.Label) + ".mp4"
		outputPath := filepath.Join(outputDir, filename)
		if err := ffmpeg.CutClip(ctx, videoPath, spec.StartSec, spec.EndSec, outputPath, reencode); err != nil {
			return CutReport{}, fmt.Errorf("cut clip %q: %w", spec.Label, err)
		}
		results = append(results, CutResult{
			Label:           spec.Label,
			StartSeconds:    spec.StartSec,
			EndSeconds:      spec.EndSec,
			DurationSeconds: spec.EndSec - spec.StartSec,
			Path:            filepath.Base(outputPath),
		})
	}

	report := CutReport{
		OK:          true,
		SourceVideo: videoPath,
		OutputDir:   outputDir,
		Clips:       results,
	}
	_ = writeManifest(filepath.Join(outputDir, "clips.json"), report)
	return report, nil
}

// MakeGIFs creates GIFs from timestamp ranges in a video. Each GIF gets a
// filename derived from its label. fps controls the GIF frame rate; width
// controls the output pixel width. A clips.json manifest is written into the
// output directory.
func MakeGIFs(ctx context.Context, videoPath, outputDir string, specs []ClipSpec, fps, width int) (GIFReport, error) {
	if len(specs) == 0 {
		return GIFReport{}, fmt.Errorf("at least one GIF spec is required")
	}
	if fps <= 0 {
		fps = 10
	}
	if width <= 0 {
		width = 480
	}
	for i, spec := range specs {
		if spec.Label == "" {
			specs[i].Label = DefaultLabel(i + 1)
		}
		if err := ValidateSpec(spec); err != nil {
			return GIFReport{}, err
		}
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return GIFReport{}, fmt.Errorf("create GIF output directory: %w", err)
	}

	results := make([]GIFResult, 0, len(specs))
	for _, spec := range specs {
		filename := SafeLabel(spec.Label) + ".gif"
		outputPath := filepath.Join(outputDir, filename)
		if err := ffmpeg.MakeGIF(ctx, videoPath, spec.StartSec, spec.EndSec, outputPath, fps, width); err != nil {
			return GIFReport{}, fmt.Errorf("make GIF %q: %w", spec.Label, err)
		}
		results = append(results, GIFResult{
			Label:           spec.Label,
			StartSeconds:    spec.StartSec,
			EndSeconds:      spec.EndSec,
			DurationSeconds: spec.EndSec - spec.StartSec,
			FPS:             fps,
			Width:           width,
			Path:            filepath.Base(outputPath),
		})
	}

	report := GIFReport{
		OK:          true,
		SourceVideo: videoPath,
		OutputDir:   outputDir,
		GIFs:        results,
	}
	_ = writeManifest(filepath.Join(outputDir, "clips.json"), report)
	return report, nil
}

// StitchClips concatenates multiple clip files into one output video. It
// creates a temporary concat list file, runs ffmpeg concat, and returns the
// output path and probed duration.
func StitchClips(ctx context.Context, clipPaths []string, outputPath string) (StitchResult, error) {
	if len(clipPaths) == 0 {
		return StitchResult{}, fmt.Errorf("at least one clip is required")
	}
	for _, p := range clipPaths {
		if _, err := os.Stat(p); err != nil {
			return StitchResult{}, fmt.Errorf("clip not found: %s", p)
		}
	}

	// Ensure the output directory exists before writing the concat list.
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return StitchResult{}, fmt.Errorf("create stitch output directory: %w", err)
	}

	// Create a concat list file next to the output.
	listPath := filepath.Join(filepath.Dir(outputPath), ".concat_list.txt")
	var b strings.Builder
	for _, p := range clipPaths {
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = p
		}
		fmt.Fprintf(&b, "file '%s'\n", abs)
	}
	if err := os.WriteFile(listPath, []byte(b.String()), 0o644); err != nil {
		return StitchResult{}, fmt.Errorf("write concat list: %w", err)
	}
	defer func() { _ = os.Remove(listPath) }()

	if err := ffmpeg.ConcatClips(ctx, listPath, outputPath); err != nil {
		return StitchResult{}, fmt.Errorf("stitch clips: %w", err)
	}

	meta, err := ffmpeg.Probe(ctx, outputPath)
	if err != nil {
		// Non-fatal: return the result without duration.
		return StitchResult{
			Inputs:     clipPaths,
			OutputPath: outputPath,
		}, nil
	}
	return StitchResult{
		Inputs:          clipPaths,
		OutputPath:      outputPath,
		DurationSeconds: meta.DurationSeconds,
	}, nil
}

// OutputDir creates a timestamped, collision-free output directory for clips or
// GIFs under parentDir, using name as a prefix. It mirrors the artifact bundle
// pattern from internal/artifacts.
func OutputDir(parentDir, name string) (string, error) {
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return "", fmt.Errorf("create parent directory: %w", err)
	}
	return artifacts.BundlePathUnique(parentDir, name+"_clips", time.Now()), nil
}

// writeManifest writes a JSON manifest file describing the clips or GIFs.
func writeManifest(path string, report any) error {
	return artifacts.WriteJSON(path, report)
}
