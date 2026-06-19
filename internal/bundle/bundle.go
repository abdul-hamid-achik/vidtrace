package bundle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/abdul-hamid-achik/vidtrace/internal/timeline"
)

type Metadata struct {
	SchemaVersion   string   `json:"schema_version"`
	SourceVideo     string   `json:"source_video"`
	GeneratedAt     string   `json:"generated_at"`
	DurationSeconds float64  `json:"duration_seconds,omitempty"`
	Width           int      `json:"width,omitempty"`
	Height          int      `json:"height,omitempty"`
	VideoCodec      string   `json:"video_codec,omitempty"`
	AudioCodec      string   `json:"audio_codec,omitempty"`
	FrameRate       float64  `json:"frame_rate,omitempty"`
	ExtractFPS      float64  `json:"extract_fps"`
	OCRLanguages    []string `json:"ocr_languages"`
	WhisperLanguage string   `json:"whisper_language,omitempty"`
	WhisperModel    string   `json:"whisper_model"`
}

type Bundle struct {
	Dir         string
	Metadata    Metadata
	Timeline    timeline.Document
	CombinedOCR string
}

func Load(dir string) (Bundle, error) {
	resolvedDir, err := filepath.Abs(dir)
	if err != nil {
		return Bundle{}, fmt.Errorf("resolve bundle: %w", err)
	}
	info, err := os.Stat(resolvedDir)
	if err != nil {
		return Bundle{}, fmt.Errorf("bundle not found: %s", resolvedDir)
	}
	if !info.IsDir() {
		return Bundle{}, fmt.Errorf("bundle is not a directory: %s", resolvedDir)
	}

	var metadata Metadata
	if err := readJSON(filepath.Join(resolvedDir, "metadata.json"), &metadata); err != nil {
		return Bundle{}, err
	}

	var timelineDoc timeline.Document
	if err := readJSON(filepath.Join(resolvedDir, "timeline.json"), &timelineDoc); err != nil {
		return Bundle{}, err
	}

	combinedOCR, err := os.ReadFile(filepath.Join(resolvedDir, "ocr", "ocr_all_frames.txt"))
	if err != nil && !os.IsNotExist(err) {
		return Bundle{}, fmt.Errorf("read combined OCR: %w", err)
	}

	return Bundle{
		Dir:         resolvedDir,
		Metadata:    metadata,
		Timeline:    timelineDoc,
		CombinedOCR: string(combinedOCR),
	}, nil
}

func (b Bundle) SearchableText() string {
	var parts []string
	parts = append(parts, b.Metadata.SourceVideo)
	if b.CombinedOCR != "" {
		parts = append(parts, b.CombinedOCR)
	}
	for _, entry := range b.Timeline.Entries {
		parts = append(parts, entry.OCR.Text)
		for _, segment := range entry.Transcript {
			parts = append(parts, segment.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func (b Bundle) TranscriptText() string {
	seen := make(map[string]struct{})
	var parts []string
	for _, entry := range b.Timeline.Entries {
		for _, segment := range entry.Transcript {
			text := strings.TrimSpace(segment.Text)
			if text == "" {
				continue
			}
			key := fmt.Sprintf("%.3f-%.3f-%s", segment.StartSeconds, segment.EndSeconds, text)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", filepath.Base(path), err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	return nil
}
