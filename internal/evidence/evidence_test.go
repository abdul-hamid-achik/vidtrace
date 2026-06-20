package evidence

import (
	"fmt"
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

func TestSearchAppliesMetadataFilters(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")

	bundleA := writeCustomEvidenceBundle(t, "/tmp/ticket-bug.mp4", 40, []evidenceEntry{
		{time: 0, ocr: "Login", transcript: "I open the ticket list"},
		{time: 10, ocr: "Ticket OPG-14010 details", transcript: "I clicked the ticket and it does not work"},
	})
	bundleB := writeCustomEvidenceBundle(t, "/tmp/checkout-bug.mp4", 40, []evidenceEntry{
		{time: 20, ocr: "Checkout", transcript: "I open the checkout page"},
		{time: 30, ocr: "Checkout failed", transcript: "I clicked the ticket and it does not work"},
	})
	for _, b := range []string{bundleA, bundleB} {
		if _, err := IndexBundle(IndexOptions{BundleDir: b, DBPath: dbPath}); err != nil {
			t.Fatalf("IndexBundle(%s) failed: %v", b, err)
		}
	}

	query := "clicked the ticket and it does not work"

	all, err := Search(SearchOptions{DBPath: dbPath, Query: query, Limit: 10})
	if err != nil {
		t.Fatalf("baseline search failed: %v", err)
	}
	if all.Filters != nil {
		t.Fatalf("expected nil Filters for unfiltered search, got %#v", all.Filters)
	}
	if got := distinctBundles(all.Results); got < 2 {
		t.Fatalf("expected results from both bundles, got %d distinct bundle(s): %#v", got, all.Results)
	}

	absA, _ := filepath.Abs(bundleA)
	onlyA, err := Search(SearchOptions{DBPath: dbPath, Query: query, Limit: 10, Bundle: bundleA})
	if err != nil {
		t.Fatalf("bundle-filtered search failed: %v", err)
	}
	if len(onlyA.Results) == 0 {
		t.Fatal("expected bundle-filtered results")
	}
	for _, r := range onlyA.Results {
		if r.Bundle != absA {
			t.Fatalf("bundle filter leaked result from %q, want %q", r.Bundle, absA)
		}
	}
	if onlyA.Filters == nil || onlyA.Filters.Bundle != absA {
		t.Fatalf("expected echoed bundle filter %q, got %#v", absA, onlyA.Filters)
	}

	onlyB, err := Search(SearchOptions{DBPath: dbPath, Query: query, Limit: 10, SourceVideo: "/tmp/checkout-bug.mp4"})
	if err != nil {
		t.Fatalf("source-video search failed: %v", err)
	}
	if len(onlyB.Results) == 0 {
		t.Fatal("expected source-video-filtered results")
	}
	for _, r := range onlyB.Results {
		if r.SourceVideo != "/tmp/checkout-bug.mp4" {
			t.Fatalf("source-video filter leaked %q", r.SourceVideo)
		}
	}

	minTime := 25.0
	timeFiltered, err := Search(SearchOptions{DBPath: dbPath, Query: query, Limit: 10, MinTime: &minTime})
	if err != nil {
		t.Fatalf("time-filtered search failed: %v", err)
	}
	if len(timeFiltered.Results) == 0 {
		t.Fatal("expected time-filtered results")
	}
	for _, r := range timeFiltered.Results {
		if r.TimeSeconds < minTime {
			t.Fatalf("min-time filter leaked result at %.3f", r.TimeSeconds)
		}
	}
	if timeFiltered.Filters == nil || timeFiltered.Filters.MinTime == nil || *timeFiltered.Filters.MinTime != minTime {
		t.Fatalf("expected echoed min-time filter, got %#v", timeFiltered.Filters)
	}
}

func TestSearchRejectsInvertedTimeRange(t *testing.T) {
	bundleDir := writeEvidenceBundle(t)
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	if _, err := IndexBundle(IndexOptions{BundleDir: bundleDir, DBPath: dbPath}); err != nil {
		t.Fatalf("IndexBundle failed: %v", err)
	}

	minTime, maxTime := 10.0, 5.0
	_, err := Search(SearchOptions{DBPath: dbPath, Query: "ticket", MinTime: &minTime, MaxTime: &maxTime})
	if err == nil || !strings.Contains(err.Error(), "greater than max-time") {
		t.Fatalf("Search error = %v, want inverted time-range failure", err)
	}
}

func TestSearchOverFetchesFilteredMatchesBelowLimit(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")

	// Noise bundle ranks ABOVE the target in raw BM25 (it repeats the query terms),
	// and has far more entries than the search limit.
	var noiseEntries []evidenceEntry
	for i := 0; i < 20; i++ {
		noiseEntries = append(noiseEntries, evidenceEntry{
			time:       float64(i),
			ocr:        "checkout error",
			transcript: "checkout error checkout error checkout error",
		})
	}
	noiseBundle := writeCustomEvidenceBundle(t, "/tmp/noise.mp4", 30, noiseEntries)

	// Target bundle has weaker matches that rank below the global top-K.
	var targetEntries []evidenceEntry
	for i := 0; i < 8; i++ {
		targetEntries = append(targetEntries, evidenceEntry{
			time:       float64(i),
			ocr:        "checkout error",
			transcript: "checkout error",
		})
	}
	targetBundle := writeCustomEvidenceBundle(t, "/tmp/target.mp4", 20, targetEntries)

	for _, b := range []string{noiseBundle, targetBundle} {
		if _, err := IndexBundle(IndexOptions{BundleDir: b, DBPath: dbPath}); err != nil {
			t.Fatalf("IndexBundle(%s) failed: %v", b, err)
		}
	}
	absTarget, _ := filepath.Abs(targetBundle)

	// Small limit: the global top-K BM25 hits are all noise, but over-fetch must
	// still surface the filter-matching target entries, then cap at the limit.
	capped, err := Search(SearchOptions{DBPath: dbPath, Query: "checkout error", Limit: 5, Bundle: targetBundle})
	if err != nil {
		t.Fatalf("capped search failed: %v", err)
	}
	if len(capped.Results) != 5 {
		t.Fatalf("expected 5 capped target results despite noise ranking higher, got %d: %#v", len(capped.Results), capped.Results)
	}
	for _, r := range capped.Results {
		if r.Bundle != absTarget {
			t.Fatalf("over-fetch leaked non-target bundle %q", r.Bundle)
		}
	}

	// Large limit: every target match is returned even though noise dominates BM25.
	full, err := Search(SearchOptions{DBPath: dbPath, Query: "checkout error", Limit: 50, Bundle: targetBundle})
	if err != nil {
		t.Fatalf("full search failed: %v", err)
	}
	if len(full.Results) != len(targetEntries) {
		t.Fatalf("expected all %d target results, got %d: %#v", len(targetEntries), len(full.Results), full.Results)
	}
	for _, r := range full.Results {
		if r.Bundle != absTarget {
			t.Fatalf("over-fetch leaked non-target bundle %q", r.Bundle)
		}
	}
}

func TestSearchSourceFilter(t *testing.T) {
	bundleDir := writeEvidenceBundle(t)
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	if _, err := IndexBundle(IndexOptions{BundleDir: bundleDir, DBPath: dbPath}); err != nil {
		t.Fatalf("IndexBundle failed: %v", err)
	}

	match, err := Search(SearchOptions{DBPath: dbPath, Query: "ticket", Source: "timeline"})
	if err != nil {
		t.Fatalf("source-filtered search failed: %v", err)
	}
	if len(match.Results) == 0 {
		t.Fatal("expected results for source=timeline")
	}
	if match.Filters == nil || match.Filters.Source != "timeline" {
		t.Fatalf("expected echoed source filter, got %#v", match.Filters)
	}

	none, err := Search(SearchOptions{DBPath: dbPath, Query: "ticket", Source: "nonexistent"})
	if err != nil {
		t.Fatalf("source-filtered search failed: %v", err)
	}
	if len(none.Results) != 0 {
		t.Fatalf("expected zero results for unknown source, got %#v", none.Results)
	}
}

func TestSearchCombinedFilters(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	bundleA := writeCustomEvidenceBundle(t, "/tmp/ticket-bug.mp4", 40, []evidenceEntry{
		{time: 0, ocr: "Login", transcript: "I clicked the ticket and it does not work"},
		{time: 10, ocr: "Ticket OPG-14010", transcript: "I clicked the ticket and it does not work"},
		{time: 20, ocr: "Logout", transcript: "I clicked the ticket and it does not work"},
	})
	if _, err := IndexBundle(IndexOptions{BundleDir: bundleA, DBPath: dbPath}); err != nil {
		t.Fatalf("IndexBundle failed: %v", err)
	}
	absA, _ := filepath.Abs(bundleA)

	minTime, maxTime := 5.0, 15.0
	rep, err := Search(SearchOptions{
		DBPath:      dbPath,
		Query:       "clicked the ticket and it does not work",
		Limit:       10,
		Bundle:      bundleA,
		SourceVideo: "/tmp/ticket-bug.mp4",
		MinTime:     &minTime,
		MaxTime:     &maxTime,
	})
	if err != nil {
		t.Fatalf("combined-filter search failed: %v", err)
	}
	if len(rep.Results) != 1 {
		t.Fatalf("expected exactly the t=10 entry, got %d: %#v", len(rep.Results), rep.Results)
	}
	r := rep.Results[0]
	if r.Bundle != absA || r.SourceVideo != "/tmp/ticket-bug.mp4" || r.TimeSeconds < minTime || r.TimeSeconds > maxTime {
		t.Fatalf("combined filter returned a result violating a constraint: %#v", r)
	}
	if rep.Filters == nil || rep.Filters.Bundle != absA || rep.Filters.SourceVideo != "/tmp/ticket-bug.mp4" ||
		rep.Filters.MinTime == nil || *rep.Filters.MinTime != minTime ||
		rep.Filters.MaxTime == nil || *rep.Filters.MaxTime != maxTime {
		t.Fatalf("expected all filters echoed, got %#v", rep.Filters)
	}
}

func distinctBundles(results []SearchResult) int {
	seen := map[string]struct{}{}
	for _, r := range results {
		seen[r.Bundle] = struct{}{}
	}
	return len(seen)
}

type evidenceEntry struct {
	time       float64
	ocr        string
	transcript string
}

func writeCustomEvidenceBundle(t *testing.T, sourceVideo string, duration float64, entries []evidenceEntry) string {
	t.Helper()

	dir := t.TempDir()
	mustMkdirEvidence(t, filepath.Join(dir, "frames"))
	mustMkdirEvidence(t, filepath.Join(dir, "ocr"))
	mustMkdirEvidence(t, filepath.Join(dir, "transcript"))

	var timelineEntries []string
	var combinedOCR strings.Builder
	for i, entry := range entries {
		frame := fmt.Sprintf("frames/frame_%04d.png", i+1)
		ocrPath := fmt.Sprintf("ocr/frame_%04d.txt", i+1)
		mustWriteEvidence(t, filepath.Join(dir, frame), fmt.Sprintf("fake frame %d", i+1))
		mustWriteEvidence(t, filepath.Join(dir, ocrPath), entry.ocr)
		combinedOCR.WriteString(entry.ocr + "\n")
		timelineEntries = append(timelineEntries, fmt.Sprintf(`    {
      "time_seconds": %g,
      "frame": %q,
      "ocr": {"path": %q, "text": %q},
      "transcript": [{"start_seconds": %g, "end_seconds": %g, "text": %q}]
    }`, entry.time, frame, ocrPath, entry.ocr, entry.time, entry.time+1, entry.transcript))
	}
	mustWriteEvidence(t, filepath.Join(dir, "ocr", "ocr_all_frames.txt"), combinedOCR.String())
	mustWriteEvidence(t, filepath.Join(dir, "metadata.json"), fmt.Sprintf(`{
  "schema_version": "1",
  "source_video": %q,
  "duration_seconds": %g,
  "extract_fps": 1,
  "ocr_languages": ["eng"],
  "whisper_language": "en",
  "whisper_model": "small"
}`, sourceVideo, duration))
	mustWriteEvidence(t, filepath.Join(dir, "timeline.json"), fmt.Sprintf(`{
  "schema_version": "1",
  "entries": [
%s
  ]
}`, strings.Join(timelineEntries, ",\n")))
	return dir
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
