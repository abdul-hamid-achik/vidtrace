package bundle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	dir := writeFixture(t)

	doc, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if doc.Metadata.SchemaVersion != "1" {
		t.Fatalf("metadata schema = %q", doc.Metadata.SchemaVersion)
	}
	if len(doc.Timeline.Entries) != 2 {
		t.Fatalf("timeline entries = %d, want 2", len(doc.Timeline.Entries))
	}
	if !strings.Contains(doc.SearchableText(), "Login failed") {
		t.Fatalf("expected searchable text to include OCR, got %q", doc.SearchableText())
	}
	if !strings.Contains(doc.TranscriptText(), "I cannot log in") {
		t.Fatalf("expected transcript text, got %q", doc.TranscriptText())
	}
}

func writeFixture(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "ocr"))
	mustMkdir(t, filepath.Join(dir, "frames"))
	mustMkdir(t, filepath.Join(dir, "transcript"))
	mustWrite(t, filepath.Join(dir, "metadata.json"), `{
  "schema_version": "1",
  "source_video": "/tmp/bug.mp4",
  "duration_seconds": 2,
  "extract_fps": 1,
  "ocr_languages": ["eng"],
  "whisper_language": "en",
  "whisper_model": "small"
}`)
	mustWrite(t, filepath.Join(dir, "timeline.json"), `{
  "schema_version": "1",
  "entries": [
    {
      "time_seconds": 0,
      "frame": "frames/frame_0001.png",
      "ocr": {"path": "ocr/frame_0001.txt", "text": "Login failed"},
      "transcript": [{"start_seconds": 0, "end_seconds": 1, "text": "I cannot log in"}]
    },
    {
      "time_seconds": 1,
      "frame": "frames/frame_0002.png",
      "ocr": {"path": "ocr/frame_0002.txt", "text": "Retry button"},
      "transcript": []
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "ocr", "ocr_all_frames.txt"), "Login failed\nRetry button\n")
	return dir
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}
