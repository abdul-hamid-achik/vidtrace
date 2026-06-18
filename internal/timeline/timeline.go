package timeline

import (
	"encoding/json"
	"fmt"
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

type whisperDocument struct {
	Segments []whisperSegment `json:"segments"`
}

type whisperSegment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

func Build(bundleDir string, framePaths []string, fps float64, transcriptJSONPath string) (Document, error) {
	segments, err := readWhisperSegments(transcriptJSONPath)
	if err != nil {
		return Document{}, err
	}

	sort.Strings(framePaths)
	entries := make([]Entry, 0, len(framePaths))
	for _, framePath := range framePaths {
		index, err := frameIndex(framePath)
		if err != nil {
			return Document{}, err
		}

		timeSeconds := float64(index-1) / fps
		ocrPath := matchingOCRPath(bundleDir, framePath)
		ocrText := ""
		if data, err := os.ReadFile(ocrPath); err == nil {
			ocrText = strings.TrimSpace(string(data))
		}

		entries = append(entries, Entry{
			TimeSeconds: timeSeconds,
			Frame:       artifacts.RelSlash(bundleDir, framePath),
			OCR: OCR{
				Path: artifacts.RelSlash(bundleDir, ocrPath),
				Text: ocrText,
			},
			Transcript: overlappingSegments(segments, timeSeconds, timeSeconds+(1/fps)),
		})
	}

	return Document{
		SchemaVersion: "1",
		Entries:       entries,
	}, nil
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

func overlappingSegments(segments []Segment, start, end float64) []Segment {
	var matches []Segment
	for _, segment := range segments {
		if segment.EndSeconds >= start && segment.StartSeconds <= end {
			matches = append(matches, segment)
		}
	}
	return matches
}
