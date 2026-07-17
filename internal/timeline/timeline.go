package timeline

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/abdul-hamid-achik/vidtrace/internal/artifacts"
)

type Document struct {
	SchemaVersion string  `json:"schema_version"`
	Entries       []Entry `json:"entries"`
}

type Entry struct {
	TimeSeconds float64   `json:"time_seconds"`
	Frame       string    `json:"frame"`
	OCR         OCR       `json:"ocr"`
	Transcript  []Segment `json:"transcript"`
	// VisualDelta is the mean absolute pixel difference versus the previous
	// frame on a downscaled grayscale grid (0–1). Omitted on the first frame
	// and when image comparison is unavailable. Higher values mark UI changes.
	VisualDelta *float64 `json:"visual_delta,omitempty"`
}

type OCR struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

type Segment struct {
	StartSeconds float64 `json:"start_seconds"`
	EndSeconds   float64 `json:"end_seconds"`
	Text         string  `json:"text"`
}

type frameRef struct {
	path  string
	index int
}

type whisperDocument struct {
	Segments []whisperSegment `json:"segments"`
}

type whisperSegment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

func Build(bundleDir string, framePaths []string, fps float64, transcriptJSONPath string) (Document, error) {
	if fps <= 0 {
		return Document{}, fmt.Errorf("fps must be greater than 0")
	}

	segments, err := readWhisperSegments(transcriptJSONPath)
	if err != nil {
		return Document{}, err
	}

	// Order by the parsed numeric frame index, not by lexical path. The extractor
	// pads to a minimum width (frame_%04d), so frame_10000.png sorts before
	// frame_9999.png lexically; sorting by index keeps frame times ascending,
	// which the interval tiling below relies on, and emits entries in time order.
	refs := make([]frameRef, 0, len(framePaths))
	for _, framePath := range framePaths {
		index, err := frameIndex(framePath)
		if err != nil {
			return Document{}, err
		}
		refs = append(refs, frameRef{path: framePath, index: index})
	}
	sort.SliceStable(refs, func(i, j int) bool { return refs[i].index < refs[j].index })

	entries := make([]Entry, 0, len(refs))
	frameTimes := make([]float64, 0, len(refs))
	for _, ref := range refs {
		timeSeconds := float64(ref.index-1) / fps
		ocrPath := matchingOCRPath(bundleDir, ref.path)
		ocrText := ""
		if data, err := os.ReadFile(ocrPath); err == nil {
			ocrText = strings.TrimSpace(string(data))
		}

		entries = append(entries, Entry{
			TimeSeconds: timeSeconds,
			Frame:       artifacts.RelSlash(bundleDir, ref.path),
			OCR: OCR{
				Path: artifacts.RelSlash(bundleDir, ocrPath),
				Text: ocrText,
			},
		})
		frameTimes = append(frameTimes, timeSeconds)
	}

	assignSegments(entries, frameTimes, segments)
	attachVisualDeltas(bundleDir, entries)

	return Document{
		SchemaVersion: artifacts.SchemaVersion,
		Entries:       entries,
	}, nil
}

// attachVisualDeltas fills Entry.VisualDelta by comparing each frame to the
// previous one. Failures are ignored so timeline generation never fails on
// image decode issues.
func attachVisualDeltas(bundleDir string, entries []Entry) {
	var prev []byte
	const grid = 16
	for i := range entries {
		path := filepath.Join(bundleDir, filepath.FromSlash(entries[i].Frame))
		sample, err := sampleFrameGray(path, grid)
		if err != nil || len(sample) == 0 {
			prev = nil
			continue
		}
		if prev != nil && len(prev) == len(sample) {
			delta := meanAbsDiff(prev, sample)
			entries[i].VisualDelta = &delta
		}
		prev = sample
	}
}

func sampleFrameGray(path string, grid int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 || grid <= 0 {
		return nil, fmt.Errorf("invalid image bounds")
	}

	samples := make([]byte, grid*grid)
	for y := 0; y < grid; y++ {
		for x := 0; x < grid; x++ {
			px := bounds.Min.X + (x*w)/grid
			py := bounds.Min.Y + (y*h)/grid
			r, g, b, _ := img.At(px, py).RGBA()
			// Approximate luminance from 16-bit channels.
			gray := (299*r + 587*g + 114*b) / 1000 / 256
			if gray > 255 {
				gray = 255
			}
			samples[y*grid+x] = byte(gray)
		}
	}
	return samples, nil
}

func meanAbsDiff(a, b []byte) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var sum float64
	for i := range a {
		d := int(a[i]) - int(b[i])
		if d < 0 {
			d = -d
		}
		sum += float64(d)
	}
	return sum / float64(len(a)) / 255.0
}

// assignSegments attaches transcript segments to frames. Each frame owns the
// half-open interval from its own time to the next frame's actual time; the last
// frame owns everything to the end of the recording. A segment is attached to
// every frame whose interval it overlaps, so a sentence spoken across several
// frames appears on each of them. This tiles the timeline with no gaps even when
// the frame rate is fractional or some frames are missing, and the half-open
// bound means a segment touching a boundary is not double-counted.
//
// Any segment that overlaps no interval (for example a zero-length segment
// exactly on a boundary) is attached to the single nearest frame by midpoint, so
// no transcript is ever dropped.
func assignSegments(entries []Entry, frameTimes []float64, segments []Segment) {
	for _, segment := range segments {
		matched := false
		for i := range entries {
			start := frameTimes[i]
			end := math.Inf(1)
			if i+1 < len(frameTimes) {
				end = frameTimes[i+1]
			}
			if segment.EndSeconds > start && segment.StartSeconds < end {
				entries[i].Transcript = append(entries[i].Transcript, segment)
				matched = true
			}
		}
		if !matched {
			if nearest := nearestFrame(frameTimes, segmentMidpoint(segment)); nearest >= 0 {
				entries[nearest].Transcript = append(entries[nearest].Transcript, segment)
			}
		}
	}
}

func segmentMidpoint(s Segment) float64 {
	return (s.StartSeconds + s.EndSeconds) / 2
}

func nearestFrame(frameTimes []float64, t float64) int {
	best := -1
	var bestDist float64
	for i, ft := range frameTimes {
		dist := math.Abs(t - ft)
		if best == -1 || dist < bestDist {
			best, bestDist = i, dist
		}
	}
	return best
}

func readWhisperSegments(path string) ([]Segment, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read whisper json: %w", err)
	}

	var doc whisperDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse whisper json: %w", err)
	}

	segments := make([]Segment, 0, len(doc.Segments))
	for _, segment := range doc.Segments {
		text := strings.TrimSpace(segment.Text)
		if text == "" {
			continue
		}
		segments = append(segments, Segment{
			StartSeconds: segment.Start,
			EndSeconds:   segment.End,
			Text:         text,
		})
	}
	return segments, nil
}

func frameIndex(path string) (int, error) {
	var index int
	base := filepath.Base(path)
	if _, err := fmt.Sscanf(base, "frame_%d.png", &index); err != nil {
		return 0, fmt.Errorf("parse frame index from %s: %w", base, err)
	}
	if index <= 0 {
		return 0, fmt.Errorf("invalid frame index in %s", base)
	}
	return index, nil
}

func matchingOCRPath(bundleDir, framePath string) string {
	base := strings.TrimSuffix(filepath.Base(framePath), filepath.Ext(framePath)) + ".txt"
	return filepath.Join(bundleDir, "ocr", base)
}
