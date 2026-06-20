package timeline

import (
	"encoding/json"
	"fmt"
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

	return Document{
		SchemaVersion: "1",
		Entries:       entries,
	}, nil
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
