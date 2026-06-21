package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestWriteCombinedOCRGlobSafety verifies the documented AGENTS.md gotcha: the
// combined OCR file (ocr_all_frames.txt) must never be included in its own
// input list. The pipeline scopes its glob to frame_*.txt; this test confirms
// that even if a combined file already exists in the ocr directory, it is not
// read back into the next combined output.
func TestWriteCombinedOCRGlobSafety(t *testing.T) {
	t.Parallel()

	ocrDir := t.TempDir()
	// Simulate a prior run's combined file sitting alongside frame OCR outputs.
	if err := os.WriteFile(filepath.Join(ocrDir, "ocr_all_frames.txt"), []byte("STALE COMBINED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ocrDir, "frame_0001.txt"), []byte("Login failed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ocrDir, "frame_0002.txt"), []byte("Retry\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Replicate the pipeline's glob: only frame_*.txt, never ocr_all_frames.txt.
	matches, err := filepath.Glob(filepath.Join(ocrDir, "frame_*.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("frame_*.txt glob matched %d files, want 2 (combined file must be excluded): %v", len(matches), matches)
	}

	combinedPath := filepath.Join(ocrDir, "ocr_all_frames.txt")
	if err := writeCombinedOCR(combinedPath, "/tmp/bug.mp4", matches, time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("writeCombinedOCR failed: %v", err)
	}

	data, err := os.ReadFile(combinedPath)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if strings.Contains(body, "STALE COMBINED") {
		t.Fatalf("combined OCR file included its own prior output (glob safety violated): %q", body)
	}
	if !strings.Contains(body, "Login failed") || !strings.Contains(body, "Retry") {
		t.Fatalf("combined OCR file missing expected frame text: %q", body)
	}
}

// TestWriteCombinedOCRTimestampIsUTC verifies the combined OCR header uses the
// injected UTC timestamp, matching metadata.json's generated_at format, instead
// of the old local-time time.Now() behavior.
func TestWriteCombinedOCRTimestampIsUTC(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	frameTXT := filepath.Join(dir, "frame_0001.txt")
	if err := os.WriteFile(frameTXT, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	combinedPath := filepath.Join(dir, "ocr_all_frames.txt")
	stamp := time.Date(2026, 6, 21, 12, 30, 0, 0, time.UTC)

	if err := writeCombinedOCR(combinedPath, "/tmp/bug.mp4", []string{frameTXT}, stamp); err != nil {
		t.Fatalf("writeCombinedOCR failed: %v", err)
	}

	data, err := os.ReadFile(combinedPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Generated: 2026-06-21T12:30:00Z") {
		t.Fatalf("expected UTC RFC3339 timestamp in combined OCR, got: %q", string(data))
	}
}

// TestWriteCombinedOCREmptyFramesProduceValidFile verifies that an all-empty OCR
// run (no text detected in any frame) still produces a valid combined file with
// headers and per-frame sections. This guards the docs/ARTIFACT_SCHEMA.md
// contract: empty OCR text means OCR ran but found nothing, not that OCR
// didn't run.
func TestWriteCombinedOCREmptyFramesProduceValidFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for _, name := range []string{"frame_0001.txt", "frame_0002.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	matches, err := filepath.Glob(filepath.Join(dir, "frame_*.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("glob matched %d, want 2", len(matches))
	}

	combinedPath := filepath.Join(dir, "ocr_all_frames.txt")
	stamp := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	if err := writeCombinedOCR(combinedPath, "/tmp/bug.mp4", matches, stamp); err != nil {
		t.Fatalf("writeCombinedOCR failed: %v", err)
	}

	data, err := os.ReadFile(combinedPath)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "Video: /tmp/bug.mp4") {
		t.Fatalf("combined OCR missing Video header: %q", body)
	}
	if !strings.Contains(body, "===== frame_0001.txt =====") || !strings.Contains(body, "===== frame_0002.txt =====") {
		t.Fatalf("combined OCR missing per-frame section headers: %q", body)
	}
}
