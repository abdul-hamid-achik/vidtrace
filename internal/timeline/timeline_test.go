package timeline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuild(t *testing.T) {
	t.Parallel()

	bundleDir := t.TempDir()
	mustMkdir(t, filepath.Join(bundleDir, "frames"))
	mustMkdir(t, filepath.Join(bundleDir, "ocr"))
	mustMkdir(t, filepath.Join(bundleDir, "transcript"))

	frame1 := filepath.Join(bundleDir, "frames", "frame_0001.png")
	frame2 := filepath.Join(bundleDir, "frames", "frame_0002.png")
	mustWrite(t, frame1, "fake frame")
	mustWrite(t, frame2, "fake frame")
	mustWrite(t, filepath.Join(bundleDir, "ocr", "frame_0001.txt"), "Login failed")
	mustWrite(t, filepath.Join(bundleDir, "ocr", "frame_0002.txt"), "Try again")

	transcriptPath := filepath.Join(bundleDir, "transcript", "bug.json")
	mustWrite(t, transcriptPath, `{
  "segments": [
    {"start": 0, "end": 1.2, "text": "I cannot log in"},
    {"start": 3, "end": 4, "text": "unrelated"}
  ]
}`)

	doc, err := Build(bundleDir, []string{frame1, frame2}, 1, transcriptPath)
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	if doc.SchemaVersion != "1" {
		t.Fatalf("SchemaVersion = %q, want 1", doc.SchemaVersion)
	}
	if len(doc.Entries) != 2 {
		t.Fatalf("len(Entries) = %d, want 2", len(doc.Entries))
	}

	first := doc.Entries[0]
	if first.TimeSeconds != 0 {
		t.Fatalf("first TimeSeconds = %v, want 0", first.TimeSeconds)
	}
	if first.Frame != "frames/frame_0001.png" {
		t.Fatalf("first Frame = %q", first.Frame)
	}
	if first.OCR.Text != "Login failed" {
		t.Fatalf("first OCR text = %q", first.OCR.Text)
	}
	if len(first.Transcript) != 1 || first.Transcript[0].Text != "I cannot log in" {
		t.Fatalf("unexpected first transcript: %#v", first.Transcript)
	}

	second := doc.Entries[1]
	if second.TimeSeconds != 1 {
		t.Fatalf("second TimeSeconds = %v, want 1", second.TimeSeconds)
	}
	if second.OCR.Text != "Try again" {
		t.Fatalf("second OCR text = %q", second.OCR.Text)
	}
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
