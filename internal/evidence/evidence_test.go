package evidence

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abdul-hamid-achik/veclite"
	"github.com/abdul-hamid-achik/vidtrace/internal/embed"
)

// fakeEmbedder is a deterministic offline embedder for tests. It hashes words
// into a fixed-dimension bag-of-words vector and L2-normalizes, so texts sharing
// words rank closer under cosine similarity. It exercises the semantic pipeline
// without a live provider; real paraphrase quality comes from a real model.
type fakeEmbedder struct {
	model string
	dims  int
}

func newFakeEmbedder(model string) fakeEmbedder {
	return fakeEmbedder{model: model, dims: 32}
}

func newFakeEmbedderDims(model string, dims int) fakeEmbedder {
	return fakeEmbedder{model: model, dims: dims}
}

func (f fakeEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))
	for i, text := range texts {
		vec := make([]float32, f.dims)
		for _, word := range strings.Fields(strings.ToLower(text)) {
			h := fnv.New32a()
			_, _ = h.Write([]byte(word))
			vec[h.Sum32()%uint32(f.dims)] += 1
		}
		var norm float32
		for _, x := range vec {
			norm += x * x
		}
		if norm > 0 {
			scale := float32(math.Sqrt(float64(norm)))
			for j := range vec {
				vec[j] /= scale
			}
		} else {
			vec[0] = 1 // avoid an all-zero vector
		}
		vectors[i] = vec
	}
	return vectors, nil
}

func (f fakeEmbedder) Profile() embed.Profile {
	return embed.Profile{Provider: "fake", Model: f.model, Dimensions: f.dims}
}

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

func TestIndexBundlesIndexesMultipleBundlesIntoOneDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	bundleA := writeCustomEvidenceBundle(t, "/tmp/a.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Login", transcript: "I open the ticket list"},
		{time: 1, ocr: "Ticket OPG-1 details", transcript: "I clicked the ticket"},
	})
	bundleB := writeCustomEvidenceBundle(t, "/tmp/b.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Checkout", transcript: "I open the checkout page"},
	})

	report, err := IndexBundles([]string{bundleA, bundleB}, dbPath, nil)
	if err != nil {
		t.Fatalf("IndexBundles failed: %v", err)
	}
	if !report.OK || report.IndexedEntries != 3 || report.InsertedEntries != 3 || report.UpdatedEntries != 0 {
		t.Fatalf("unexpected aggregate report: %#v", report)
	}
	if len(report.Bundles) != 2 || report.Bundles[0].IndexedEntries != 2 || report.Bundles[1].IndexedEntries != 1 {
		t.Fatalf("unexpected per-bundle results: %#v", report.Bundles)
	}

	res, err := Search(SearchOptions{DBPath: dbPath, Query: "ticket checkout", Limit: 10})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if got := distinctBundles(res.Results); got != 2 {
		t.Fatalf("expected evidence from both bundles, got %d: %#v", got, res.Results)
	}

	// Re-indexing the same bundles updates in place rather than duplicating.
	again, err := IndexBundles([]string{bundleA, bundleB}, dbPath, nil)
	if err != nil {
		t.Fatalf("second IndexBundles failed: %v", err)
	}
	if again.IndexedEntries != 3 || again.InsertedEntries != 0 || again.UpdatedEntries != 3 {
		t.Fatalf("expected idempotent re-index, got: %#v", again)
	}
}

func TestIndexBundlesDeduplicatesRepeatedPaths(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	bundleA := writeCustomEvidenceBundle(t, "/tmp/a.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Login", transcript: "open the ticket"},
		{time: 1, ocr: "Ticket details", transcript: "click the ticket"},
	})

	report, err := IndexBundles([]string{bundleA, bundleA}, dbPath, nil)
	if err != nil {
		t.Fatalf("IndexBundles failed: %v", err)
	}
	if len(report.Bundles) != 1 || report.IndexedEntries != 2 || report.InsertedEntries != 2 {
		t.Fatalf("expected duplicate path to be indexed once, got: %#v", report)
	}
}

func TestIndexBundlesFailsFastWithoutWritingDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	valid := writeCustomEvidenceBundle(t, "/tmp/a.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Login", transcript: "open the ticket"},
	})
	invalid := t.TempDir() // empty directory is not a valid bundle

	_, err := IndexBundles([]string{valid, invalid}, dbPath, nil)
	if err == nil || !strings.Contains(err.Error(), "bundle validation failed") {
		t.Fatalf("IndexBundles error = %v, want validation failure", err)
	}
	if _, statErr := os.Stat(dbPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected no evidence db created on fail-fast, stat err = %v", statErr)
	}
}

func TestIndexBundlesRequiresAtLeastOneBundle(t *testing.T) {
	_, err := IndexBundles(nil, filepath.Join(t.TempDir(), "evidence.veclite"), nil)
	if err == nil || !strings.Contains(err.Error(), "at least one bundle") {
		t.Fatalf("IndexBundles error = %v, want empty-input failure", err)
	}
}

func TestIndexBundlesMixedInsertAndUpdateCounts(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	bundleA := writeCustomEvidenceBundle(t, "/tmp/a.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Login", transcript: "open the ticket"},
		{time: 1, ocr: "Ticket details", transcript: "click the ticket"},
	})
	bundleB := writeCustomEvidenceBundle(t, "/tmp/b.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Checkout", transcript: "open the checkout"},
	})

	// Pre-index A so the combined call sees A as updates and B as inserts.
	if _, err := IndexBundle(IndexOptions{BundleDir: bundleA, DBPath: dbPath}); err != nil {
		t.Fatalf("pre-index failed: %v", err)
	}

	report, err := IndexBundles([]string{bundleA, bundleB}, dbPath, nil)
	if err != nil {
		t.Fatalf("IndexBundles failed: %v", err)
	}
	if report.IndexedEntries != 3 || report.InsertedEntries != 1 || report.UpdatedEntries != 2 {
		t.Fatalf("unexpected mixed aggregate counts: %#v", report)
	}
	if report.Bundles[0].UpdatedEntries != 2 || report.Bundles[0].InsertedEntries != 0 {
		t.Fatalf("expected bundle A counted as updates: %#v", report.Bundles[0])
	}
	if report.Bundles[1].InsertedEntries != 1 || report.Bundles[1].UpdatedEntries != 0 {
		t.Fatalf("expected bundle B counted as inserts: %#v", report.Bundles[1])
	}
	// Aggregate totals must equal the sum of per-bundle results.
	var sumIndexed, sumInserted, sumUpdated int
	for _, b := range report.Bundles {
		sumIndexed += b.IndexedEntries
		sumInserted += b.InsertedEntries
		sumUpdated += b.UpdatedEntries
	}
	if sumIndexed != report.IndexedEntries || sumInserted != report.InsertedEntries || sumUpdated != report.UpdatedEntries {
		t.Fatalf("aggregate totals do not match per-bundle sum: %#v", report)
	}
}

func TestIndexBundlesDeduplicatesSymlinkAlias(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	bundleA := writeCustomEvidenceBundle(t, "/tmp/a.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Login", transcript: "open the ticket"},
	})
	link := filepath.Join(t.TempDir(), "link-to-bundle")
	if err := os.Symlink(bundleA, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	report, err := IndexBundles([]string{bundleA, link}, dbPath, nil)
	if err != nil {
		t.Fatalf("IndexBundles failed: %v", err)
	}
	if len(report.Bundles) != 1 || report.IndexedEntries != 1 {
		t.Fatalf("expected symlink alias to be indexed once, got: %#v", report)
	}
}

func TestIndexBundlesPersistsProcessedBundlesOnLaterFailure(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	good := writeCustomEvidenceBundle(t, "/tmp/a.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Login", transcript: "open the ticket"},
	})
	// The second bundle passes validation (the combined OCR path exists) but fails
	// to load because that path is a directory rather than a readable file.
	bad := writeCustomEvidenceBundle(t, "/tmp/b.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Checkout", transcript: "open the checkout"},
	})
	combined := filepath.Join(bad, "ocr", "ocr_all_frames.txt")
	if err := os.Remove(combined); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(combined, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := IndexBundles([]string{good, bad}, dbPath, nil); err == nil {
		t.Fatal("expected a load failure for the second bundle")
	}

	// The first bundle was indexed before the failure, so re-indexing it alone
	// reports updates rather than inserts. This documents that a phase-2 failure
	// leaves already-processed bundles indexed (idempotent to re-run), it does not
	// roll the whole batch back.
	report, err := IndexBundle(IndexOptions{BundleDir: good, DBPath: dbPath})
	if err != nil {
		t.Fatalf("re-index of good bundle failed: %v", err)
	}
	if report.InsertedEntries != 0 || report.UpdatedEntries != 1 {
		t.Fatalf("expected good bundle already persisted before failure, got: %#v", report)
	}
}

func TestIndexAndSearchSemanticAndHybrid(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	bundleDir := writeCustomEvidenceBundle(t, "/tmp/ticket.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Login screen", transcript: "I am on the login page"},
		{time: 1, ocr: "Ticket OPG-14010", transcript: "I clicked the ticket and it does not open"},
	})
	embedder := newFakeEmbedder("fake-model")

	report, err := IndexBundle(IndexOptions{BundleDir: bundleDir, DBPath: dbPath, Embedder: embedder})
	if err != nil {
		t.Fatalf("semantic index failed: %v", err)
	}
	if report.SemanticEntries != 2 || report.Embedding == nil || report.Embedding.Model != "fake-model" || report.Embedding.Dimensions != 32 {
		t.Fatalf("unexpected semantic index report: %#v", report)
	}

	for _, mode := range []string{ModeSemantic, ModeHybrid} {
		res, err := Search(SearchOptions{
			DBPath:   dbPath,
			Query:    "clicked the ticket and it does not open",
			Limit:    5,
			Mode:     mode,
			Embedder: embedder,
		})
		if err != nil {
			t.Fatalf("%s search failed: %v", mode, err)
		}
		if res.Mode != mode || res.Collection != Collection {
			t.Fatalf("%s search wrong mode/collection: %#v", mode, res)
		}
		if len(res.Results) == 0 || res.Results[0].Frame != "frames/frame_0002.png" {
			t.Fatalf("%s search did not rank the matching entry first: %#v", mode, res.Results)
		}
	}

	// Keyword still works on the same database after semantic indexing.
	kw, err := Search(SearchOptions{DBPath: dbPath, Query: "OPG-14010", Mode: ModeKeyword})
	if err != nil || len(kw.Results) == 0 || kw.Mode != ModeKeyword {
		t.Fatalf("keyword search regressed after semantic index: %#v err=%v", kw, err)
	}
}

func TestSemanticSearchFiltersByBundle(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	embedder := newFakeEmbedder("fake-model")
	bundleA := writeCustomEvidenceBundle(t, "/tmp/a.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Checkout", transcript: "the checkout button fails"},
	})
	bundleB := writeCustomEvidenceBundle(t, "/tmp/b.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Checkout", transcript: "the checkout button fails"},
	})
	if _, err := IndexBundles([]string{bundleA, bundleB}, dbPath, embedder); err != nil {
		t.Fatalf("semantic multi-index failed: %v", err)
	}
	absA, _ := filepath.Abs(bundleA)

	res, err := Search(SearchOptions{
		DBPath:   dbPath,
		Query:    "checkout button fails",
		Limit:    10,
		Mode:     ModeSemantic,
		Embedder: embedder,
		Bundle:   bundleA,
	})
	if err != nil {
		t.Fatalf("filtered semantic search failed: %v", err)
	}
	if len(res.Results) == 0 {
		t.Fatal("expected filtered semantic results")
	}
	for _, r := range res.Results {
		if r.Bundle != absA {
			t.Fatalf("bundle filter leaked %q in semantic mode", r.Bundle)
		}
	}
}

func TestSemanticSearchRequiresEmbedder(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	bundleDir := writeCustomEvidenceBundle(t, "/tmp/a.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Login", transcript: "open the ticket"},
	})
	if _, err := IndexBundle(IndexOptions{BundleDir: bundleDir, DBPath: dbPath, Embedder: newFakeEmbedder("fake-model")}); err != nil {
		t.Fatalf("index failed: %v", err)
	}
	_, err := Search(SearchOptions{DBPath: dbPath, Query: "ticket", Mode: ModeSemantic})
	if err == nil || !strings.Contains(err.Error(), "requires an embedding provider") {
		t.Fatalf("Search error = %v, want missing-embedder failure", err)
	}
}

func TestSemanticSearchWithoutIndexFails(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	bundleDir := writeEvidenceBundle(t)
	// Keyword-only index: no semantic collection is created.
	if _, err := IndexBundle(IndexOptions{BundleDir: bundleDir, DBPath: dbPath}); err != nil {
		t.Fatalf("keyword index failed: %v", err)
	}
	_, err := Search(SearchOptions{DBPath: dbPath, Query: "ticket", Mode: ModeSemantic, Embedder: newFakeEmbedder("fake-model")})
	if err == nil || !strings.Contains(err.Error(), "no semantic index") {
		t.Fatalf("Search error = %v, want missing-index failure", err)
	}
}

func TestKeywordIndexDoesNotCreateSemanticCollection(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	bundleDir := writeEvidenceBundle(t)
	report, err := IndexBundle(IndexOptions{BundleDir: bundleDir, DBPath: dbPath})
	if err != nil {
		t.Fatalf("keyword index failed: %v", err)
	}
	if report.SemanticEntries != 0 || report.Embedding != nil {
		t.Fatalf("keyword-only index should not report semantic data: %#v", report)
	}
}

func TestSemanticSearchRejectsProfileMismatch(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	bundleDir := writeCustomEvidenceBundle(t, "/tmp/a.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Login", transcript: "open the ticket"},
	})
	if _, err := IndexBundle(IndexOptions{BundleDir: bundleDir, DBPath: dbPath, Embedder: newFakeEmbedder("model-a")}); err != nil {
		t.Fatalf("index failed: %v", err)
	}
	_, err := Search(SearchOptions{DBPath: dbPath, Query: "ticket", Mode: ModeSemantic, Embedder: newFakeEmbedder("model-b")})
	if err == nil || !strings.Contains(err.Error(), "profile mismatch") {
		t.Fatalf("Search error = %v, want profile mismatch", err)
	}
}

func TestIndexRejectsMixedEmbeddingProfiles(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	bundleA := writeCustomEvidenceBundle(t, "/tmp/a.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Login", transcript: "open the ticket"},
	})
	bundleB := writeCustomEvidenceBundle(t, "/tmp/b.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Checkout", transcript: "open the checkout"},
	})
	if _, err := IndexBundle(IndexOptions{BundleDir: bundleA, DBPath: dbPath, Embedder: newFakeEmbedder("model-a")}); err != nil {
		t.Fatalf("first index failed: %v", err)
	}
	_, err := IndexBundle(IndexOptions{BundleDir: bundleB, DBPath: dbPath, Embedder: newFakeEmbedder("model-b")})
	if err == nil || !strings.Contains(err.Error(), "profile mismatch") {
		t.Fatalf("expected profile mismatch when mixing embedders, got %v", err)
	}
}

func TestSemanticReindexDoesNotDuplicateVectors(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	embedder := newFakeEmbedder("fake-model")
	bundleDir := writeCustomEvidenceBundle(t, "/tmp/a.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Login", transcript: "open the ticket"},
		{time: 1, ocr: "Ticket details", transcript: "click the ticket"},
	})
	for i := 0; i < 2; i++ {
		if _, err := IndexBundle(IndexOptions{BundleDir: bundleDir, DBPath: dbPath, Embedder: embedder}); err != nil {
			t.Fatalf("semantic index pass %d failed: %v", i, err)
		}
	}

	db, err := veclite.Open(dbPath, veclite.WithReadOnly(true))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()
	coll, err := db.GetCollection(Collection)
	if err != nil {
		t.Fatalf("get text collection: %v", err)
	}
	if coll.Count() != 2 {
		t.Fatalf("expected 2 vectors after re-index (no duplicates), got %d", coll.Count())
	}
}

func TestSemanticAndHybridSearchOverFetchFiltered(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	embedder := newFakeEmbedder("fake-model")

	var noise []evidenceEntry
	for i := 0; i < 20; i++ {
		noise = append(noise, evidenceEntry{time: float64(i), ocr: "checkout error", transcript: "checkout error checkout error"})
	}
	noiseBundle := writeCustomEvidenceBundle(t, "/tmp/noise.mp4", 30, noise)
	var target []evidenceEntry
	for i := 0; i < 5; i++ {
		target = append(target, evidenceEntry{time: float64(i), ocr: "checkout error", transcript: "checkout error"})
	}
	targetBundle := writeCustomEvidenceBundle(t, "/tmp/target.mp4", 10, target)
	if _, err := IndexBundles([]string{noiseBundle, targetBundle}, dbPath, embedder); err != nil {
		t.Fatalf("semantic multi-index failed: %v", err)
	}
	absTarget, _ := filepath.Abs(targetBundle)

	for _, mode := range []string{ModeSemantic, ModeHybrid} {
		res, err := Search(SearchOptions{
			DBPath:   dbPath,
			Query:    "checkout error",
			Limit:    50,
			Mode:     mode,
			Embedder: embedder,
			Bundle:   targetBundle,
		})
		if err != nil {
			t.Fatalf("%s filtered search failed: %v", mode, err)
		}
		if len(res.Results) != len(target) {
			t.Fatalf("%s: expected %d target results despite noise, got %d", mode, len(target), len(res.Results))
		}
		for _, r := range res.Results {
			if r.Bundle != absTarget {
				t.Fatalf("%s filter leaked non-target bundle %q", mode, r.Bundle)
			}
		}
	}
}

func TestSemanticSearchRejectsDimensionMismatch(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	bundleDir := writeCustomEvidenceBundle(t, "/tmp/a.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Login", transcript: "open the ticket"},
	})
	if _, err := IndexBundle(IndexOptions{BundleDir: bundleDir, DBPath: dbPath, Embedder: newFakeEmbedderDims("same-model", 32)}); err != nil {
		t.Fatalf("index failed: %v", err)
	}
	// Same provider/model but a different dimension must be rejected by Search.
	_, err := Search(SearchOptions{DBPath: dbPath, Query: "ticket", Mode: ModeSemantic, Embedder: newFakeEmbedderDims("same-model", 16)})
	if err == nil || !strings.Contains(err.Error(), "dimension mismatch") {
		t.Fatalf("Search error = %v, want dimension mismatch", err)
	}
}

func TestIndexRejectsDimensionMismatchAcrossBundles(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	bundleA := writeCustomEvidenceBundle(t, "/tmp/a.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Login", transcript: "open the ticket"},
	})
	bundleB := writeCustomEvidenceBundle(t, "/tmp/b.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Checkout", transcript: "open the checkout"},
	})
	if _, err := IndexBundle(IndexOptions{BundleDir: bundleA, DBPath: dbPath, Embedder: newFakeEmbedderDims("same-model", 32)}); err != nil {
		t.Fatalf("first index failed: %v", err)
	}
	_, err := IndexBundle(IndexOptions{BundleDir: bundleB, DBPath: dbPath, Embedder: newFakeEmbedderDims("same-model", 16)})
	if err == nil || !strings.Contains(err.Error(), "profile mismatch") {
		t.Fatalf("expected profile mismatch for differing dimensions, got %v", err)
	}
}

func TestMultiBundleSemanticCounts(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	embedder := newFakeEmbedder("fake-model")
	bundleA := writeCustomEvidenceBundle(t, "/tmp/a.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Login", transcript: "open the ticket"},
		{time: 1, ocr: "Ticket", transcript: "click the ticket"},
	})
	bundleB := writeCustomEvidenceBundle(t, "/tmp/b.mp4", 10, []evidenceEntry{
		{time: 0, ocr: "Checkout", transcript: "open the checkout"},
		{time: 1, ocr: "Pay", transcript: "pay now"},
		{time: 2, ocr: "Done", transcript: "order done"},
	})

	report, err := IndexBundles([]string{bundleA, bundleB}, dbPath, embedder)
	if err != nil {
		t.Fatalf("semantic multi-index failed: %v", err)
	}
	if report.SemanticEntries != 5 {
		t.Fatalf("expected 5 aggregate semantic entries, got %d", report.SemanticEntries)
	}
	if report.Bundles[0].SemanticEntries != 2 || report.Bundles[1].SemanticEntries != 3 {
		t.Fatalf("unexpected per-bundle semantic counts: %#v", report.Bundles)
	}
	if report.Embedding == nil || report.Embedding.Model != "fake-model" || report.Embedding.Dimensions != 32 {
		t.Fatalf("expected embedding profile echoed, got %#v", report.Embedding)
	}
}

func TestSearchRejectsUnknownMode(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	bundleDir := writeEvidenceBundle(t)
	if _, err := IndexBundle(IndexOptions{BundleDir: bundleDir, DBPath: dbPath}); err != nil {
		t.Fatalf("index failed: %v", err)
	}
	_, err := Search(SearchOptions{DBPath: dbPath, Query: "ticket", Mode: "fuzzy"})
	if err == nil || !strings.Contains(err.Error(), "unknown search mode") {
		t.Fatalf("Search error = %v, want unknown-mode failure", err)
	}
}

func TestMigrateEvidenceIsIdempotentOnModernDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	bundleDir := writeEvidenceBundle(t)
	if _, err := IndexBundle(IndexOptions{BundleDir: bundleDir, DBPath: dbPath}); err != nil {
		t.Fatalf("index failed: %v", err)
	}

	report, err := Migrate(dbPath)
	if err != nil {
		t.Fatalf("migrate on modern DB failed: %v", err)
	}
	if !report.OK || !report.AlreadyMigrated || report.MigratedRecords != 0 {
		t.Fatalf("expected already-migrated report, got %#v", report)
	}

	// A modern DB still has exactly one collection after migrate.
	res, err := Search(SearchOptions{DBPath: dbPath, Query: "ticket"})
	if err != nil {
		t.Fatalf("search after migrate failed: %v", err)
	}
	if len(res.Results) == 0 {
		t.Fatal("expected search results preserved after migrate")
	}
	if res.Collection != Collection {
		t.Fatalf("collection = %q, want %q", res.Collection, Collection)
	}
}

func TestMigrateEvidenceRejectsMissingDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "does-not-exist.veclite")
	_, err := Migrate(dbPath)
	if err == nil || !strings.Contains(err.Error(), "evidence db not found") {
		t.Fatalf("Migrate error = %v, want not-found failure", err)
	}
}

func TestMigrateEvidenceRejectsEmptyDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "empty.veclite")
	if _, err := IndexBundle(IndexOptions{BundleDir: writeEvidenceBundle(t), DBPath: dbPath}); err != nil {
		t.Fatalf("index failed: %v", err)
	}
	// Create an unrelated DB with no evidence collections and confirm migrate
	// refuses to treat it as a legacy evidence database.
	otherPath := filepath.Join(t.TempDir(), "other.veclite")
	db, err := veclite.Open(otherPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := db.CreateCollection("unrelated"); err != nil {
		t.Fatalf("create unrelated: %v", err)
	}
	_ = db.Close()

	_, err = Migrate(otherPath)
	if err == nil || !strings.Contains(err.Error(), "no legacy evidence collections") {
		t.Fatalf("Migrate error = %v, want no-legacy failure", err)
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
