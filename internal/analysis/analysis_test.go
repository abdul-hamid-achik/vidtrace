package analysis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareMatch(t *testing.T) {
	t.Parallel()

	bundleDir := writeBundle(t)
	ticketPath := filepath.Join(t.TempDir(), "ticket.md")
	mustWrite(t, ticketPath, "Login failed after submit. User cannot log in.")

	result, err := Compare(Options{BundleDir: bundleDir, TicketPath: ticketPath})
	if err != nil {
		t.Fatalf("Compare() failed: %v", err)
	}

	if result.Status != "match" {
		t.Fatalf("Status = %q, want match; result=%#v", result.Status, result)
	}
	if len(result.Evidence) == 0 {
		t.Fatalf("expected evidence")
	}
	if !contains(result.MatchedTerms, "login") {
		t.Fatalf("expected matched login term, got %v", result.MatchedTerms)
	}
	if result.Confidence == "" {
		t.Fatalf("expected confidence")
	}
	if len(result.TermHits) == 0 {
		t.Fatalf("expected term hits")
	}
	if _, err := json.Marshal(result); err != nil {
		t.Fatalf("result should marshal as JSON: %v", err)
	}
}

func TestCompareNormalizesSeparatedTerms(t *testing.T) {
	t.Parallel()

	bundleDir := writeBundle(t)
	ticketPath := filepath.Join(t.TempDir(), "ticket.md")
	mustWrite(t, ticketPath, "Log-in fails after submit.")

	result, err := Compare(Options{BundleDir: bundleDir, TicketPath: ticketPath})
	if err != nil {
		t.Fatalf("Compare() failed: %v", err)
	}

	if !contains(result.MatchedTerms, "login") {
		t.Fatalf("expected normalized login term, got %v", result.MatchedTerms)
	}
	if !hasTermHit(result.TermHits, "login", "ocr") {
		t.Fatalf("expected OCR term hit for login, got %#v", result.TermHits)
	}
}

func TestCompareMismatch(t *testing.T) {
	t.Parallel()

	bundleDir := writeBundle(t)
	ticketPath := filepath.Join(t.TempDir(), "ticket.md")
	mustWrite(t, ticketPath, "Billing invoice PDF export is blank.")

	result, err := Compare(Options{BundleDir: bundleDir, TicketPath: ticketPath})
	if err != nil {
		t.Fatalf("Compare() failed: %v", err)
	}

	if result.Status != "mismatch" {
		t.Fatalf("Status = %q, want mismatch; result=%#v", result.Status, result)
	}
	if len(result.Evidence) != 0 {
		t.Fatalf("expected no evidence, got %#v", result.Evidence)
	}
	if result.Confidence != "low" {
		t.Fatalf("Confidence = %q, want low", result.Confidence)
	}
}

func TestMarkdown(t *testing.T) {
	t.Parallel()

	result := Result{
		OK:           true,
		Status:       "match",
		Confidence:   "medium",
		Score:        0.5,
		MatchedTerms: []string{"login"},
		TermHits: []TermHit{{
			Term:        "login",
			Source:      "ocr",
			TimeSeconds: 0,
			Frame:       "frames/frame_0001.png",
			Text:        "Login failed",
		}},
		Evidence: []EvidenceRef{{
			TimeSeconds: 0,
			Frame:       "frames/frame_0001.png",
			OCRPath:     "ocr/frame_0001.txt",
			Text:        "Login failed",
		}},
		Gaps: []string{"Verify frame visually."},
	}

	report := Markdown(result)
	for _, want := range []string{"## Summary", "Status: match", "frames/frame_0001.png", "Verify frame visually."} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected report to contain %q, got %q", want, report)
		}
	}
}

func writeBundle(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "ocr"))
	mustMkdir(t, filepath.Join(dir, "frames"))
	mustMkdir(t, filepath.Join(dir, "transcript"))
	mustWrite(t, filepath.Join(dir, "metadata.json"), `{
  "schema_version": "1",
  "source_video": "/tmp/login-bug.mp4",
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
      "ocr": {"path": "ocr/frame_0001.txt", "text": "Login failed after submit"},
      "transcript": [{"start_seconds": 0, "end_seconds": 1, "text": "I cannot log in"}]
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "ocr", "ocr_all_frames.txt"), "Login failed after submit\n")
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

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func hasTermHit(values []TermHit, term, source string) bool {
	for _, value := range values {
		if value.Term == term && value.Source == source {
			return true
		}
	}
	return false
}
