package evidence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndexBundleAndSearchKeywordEvidence(t *testing.T) {
	bundleDir := writeEvidenceBundle(t)
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")

	indexReport, err := IndexBundle(IndexOptions{
		BundleDir: bundleDir,
		DBPath:    dbPath,
	})
	if err != nil {
		t.Fatalf("IndexBundle failed: %v", err)
	}
	if !indexReport.OK || indexReport.IndexedEntries != 2 || indexReport.InsertedEntries != 2 {
		t.Fatalf("unexpected index report: %#v", indexReport)
	}

	searchReport, err := Search(SearchOptions{
		DBPath: dbPath,
		Query:  "clicking ticket does not work",
		Limit:  5,
	})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if !searchReport.OK || len(searchReport.Results) == 0 {
		t.Fatalf("unexpected search report: %#v", searchReport)
	}
	result := searchReport.Results[0]
	if result.Frame != "frames/frame_0002.png" {
		t.Fatalf("first result frame = %q, want frame_0002", result.Frame)
	}
	if !strings.Contains(result.Transcript, "ticket") || !result.HasTranscript {
		t.Fatalf("result did not include transcript evidence: %#v", result)
	}
	if result.SourceVideo != "/tmp/ticket-bug.mp4" {
		t.Fatalf("source video = %q", result.SourceVideo)
	}
}

func TestIndexBundleIsIdempotentForSameBundle(t *testing.T) {
	bundleDir := writeEvidenceBundle(t)
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")

	if _, err := IndexBundle(IndexOptions{BundleDir: bundleDir, DBPath: dbPath}); err != nil {
		t.Fatalf("first IndexBundle failed: %v", err)
	}
	report, err := IndexBundle(IndexOptions{BundleDir: bundleDir, DBPath: dbPath})
	if err != nil {
		t.Fatalf("second IndexBundle failed: %v", err)
	}
	if report.IndexedEntries != 2 || report.InsertedEntries != 0 || report.UpdatedEntries != 2 {
		t.Fatalf("unexpected second index report: %#v", report)
	}
}

func TestIndexBundleRejectsInvalidBundle(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")

	_, err := IndexBundle(IndexOptions{
		BundleDir: t.TempDir(),
		DBPath:    dbPath,
	})
	if err == nil || !strings.Contains(err.Error(), "bundle validation failed") {
		t.Fatalf("IndexBundle error = %v, want validation failure", err)
	}
}

func TestEvidenceContentIncludesDeterministicFields(t *testing.T) {
	bundleDir := writeEvidenceBundle(t)
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	if _, err := IndexBundle(IndexOptions{BundleDir: bundleDir, DBPath: dbPath}); err != nil {
		t.Fatalf("IndexBundle failed: %v", err)
	}

	report, err := Search(SearchOptions{DBPath: dbPath, Query: "OPG-14010"})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(report.Results) == 0 {
		t.Fatal("expected OPG-14010 search result")
	}
	result := report.Results[0]
	if result.EvidenceID == "" || result.TimeSeconds != 1 || !result.HasOCR {
		t.Fatalf("unexpected result fields: %#v", result)
	}
	if result.OCR != "Ticket OPG-14010 details" {
		t.Fatalf("OCR = %q", result.OCR)
	}
}

func writeEvidenceBundle(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	mustMkdirEvidence(t, filepath.Join(dir, "frames"))
	mustMkdirEvidence(t, filepath.Join(dir, "ocr"))
	mustMkdirEvidence(t, filepath.Join(dir, "transcript"))
	mustWriteEvidence(t, filepath.Join(dir, "frames", "frame_0001.png"), "fake frame 1")
	mustWriteEvidence(t, filepath.Join(dir, "frames", "frame_0002.png"), "fake frame 2")
	mustWriteEvidence(t, filepath.Join(dir, "ocr", "frame_0001.txt"), "Login")
	mustWriteEvidence(t, filepath.Join(dir, "ocr", "frame_0002.txt"), "Ticket OPG-14010 details")
	mustWriteEvidence(t, filepath.Join(dir, "ocr", "ocr_all_frames.txt"), "Login\nTicket OPG-14010 details\n")
	mustWriteEvidence(t, filepath.Join(dir, "metadata.json"), `{
  "schema_version": "1",
  "source_video": "/tmp/ticket-bug.mp4",
  "duration_seconds": 2,
  "extract_fps": 1,
  "ocr_languages": ["eng"],
  "whisper_language": "en",
  "whisper_model": "small"
}`)
	mustWriteEvidence(t, filepath.Join(dir, "timeline.json"), `{
  "schema_version": "1",
  "entries": [
    {
      "time_seconds": 0,
      "frame": "frames/frame_0001.png",
      "ocr": {"path": "ocr/frame_0001.txt", "text": "Login"},
      "transcript": [{"start_seconds": 0, "end_seconds": 1, "text": "I open the ticket list"}]
    },
    {
      "time_seconds": 1,
      "frame": "frames/frame_0002.png",
      "ocr": {"path": "ocr/frame_0002.txt", "text": "Ticket OPG-14010 details"},
      "transcript": [{"start_seconds": 1, "end_seconds": 2, "text": "I clicked the ticket and it does not work"}]
    }
  ]
}`)
	return dir
}

func mustMkdirEvidence(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteEvidence(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
