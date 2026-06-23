package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abdul-hamid-achik/vidtrace/internal/evidence"
)

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"help"}, &stdout, &stderr, "test")

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "vidtrace turns bug videos") {
		t.Fatalf("expected help text, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "docs") {
		t.Fatalf("expected docs command in help, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "studio") {
		t.Fatalf("expected studio command in help, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "validate") {
		t.Fatalf("expected validate command in help, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "index") || !strings.Contains(stdout.String(), "search") {
		t.Fatalf("expected evidence search commands in help, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "investigate") {
		t.Fatalf("expected investigate command in help, got %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "vidtrace site") {
		t.Fatalf("did not expect site command in help, got %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "tui") {
		t.Fatalf("did not expect tui command in help, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"version"}, &stdout, &stderr, "test-version")

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if got := strings.TrimSpace(stdout.String()); got != "vidtrace test-version" {
		t.Fatalf("unexpected version output: %q", got)
	}
}

func TestUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"nope"}, &stdout, &stderr, "test")

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown command: nope") {
		t.Fatalf("expected unknown command error, got %q", stderr.String())
	}
}

func TestTUICommandIsRenamedToStudio(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"tui"}, &stdout, &stderr, "test")

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown command: tui") {
		t.Fatalf("expected unknown tui command, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "studio") {
		t.Fatalf("expected help to point to studio, got %q", stderr.String())
	}
}

func TestRunDocsOverview(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"docs"}, &stdout, &stderr, "test")

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "vidtrace product docs") {
		t.Fatalf("expected product docs title, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "vidtrace docs agent") {
		t.Fatalf("expected topic list, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "vidtrace docs studio") {
		t.Fatalf("expected studio docs topic, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "docs/SITE.md") {
		t.Fatalf("expected site docs pointer, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunDocsAgent(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"docs", "agent"}, &stdout, &stderr, "test")

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	output := stdout.String()
	for _, want := range []string{
		"vidtrace agent guide",
		"vidtrace extract VIDEO --json",
		"vidtrace index output_dir",
		"vidtrace search /tmp/vidtrace-evidence.veclite",
		"vidtrace investigate output_dir",
		"metadata.json",
		"timeline.json",
		"match, mismatch, or are inconclusive",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected docs output to contain %q, got %q", want, output)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunDocsStudio(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"docs", "studio"}, &stdout, &stderr, "test")

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	output := stdout.String()
	for _, want := range []string{
		"vidtrace studio docs",
		"up/down or k/j",
		"m                   toggle bundle metadata/details",
		"o                   open the selected frame",
		"c                   copy a concise evidence summary",
		"selected frame path",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected docs output to contain %q, got %q", want, output)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunDocsUnknownTopic(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"docs", "nope"}, &stdout, &stderr, "test")

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unknown docs topic: nope") {
		t.Fatalf("expected unknown topic error, got %q", stderr.String())
	}
}

func TestExtractRequiresVideoPath(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"extract"}, &stdout, &stderr, "test")

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage: vidtrace extract") {
		t.Fatalf("expected usage error, got %q", stderr.String())
	}
}

func TestExtractJSONFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"extract", "-json", "/does/not/exist.mp4"}, &stdout, &stderr, "test")

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stdout.String(), `"ok": false`) {
		t.Fatalf("expected json failure, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr for json failure, got %q", stderr.String())
	}
}

func TestCompareJSON(t *testing.T) {
	bundleDir := writeCLIBundle(t)
	ticketPath := filepath.Join(t.TempDir(), "ticket.md")
	mustWrite(t, ticketPath, "Login failed after submit")
	var stdout, stderr bytes.Buffer

	code := Run([]string{"compare", bundleDir, "--ticket", ticketPath, "--json"}, &stdout, &stderr, "test")

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	var result struct {
		OK           bool     `json:"ok"`
		Status       string   `json:"status"`
		Confidence   string   `json:"confidence"`
		MatchedTerms []string `json:"matched_terms"`
		TermHits     []struct {
			Term   string `json:"term"`
			Source string `json:"source"`
		} `json:"term_hits"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", stdout.String(), err)
	}
	if !result.OK || result.Status != "match" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Confidence == "" || len(result.MatchedTerms) == 0 || len(result.TermHits) == 0 {
		t.Fatalf("expected confidence, matched terms, and term hits: %#v", result)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestCompareJSONFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"compare", "/missing/bundle", "--ticket", "/missing/ticket.md", "--json"}, &stdout, &stderr, "test")

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stdout.String(), `"ok": false`) {
		t.Fatalf("expected json failure, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr for json failure, got %q", stderr.String())
	}
}

func TestAnalyzeMarkdown(t *testing.T) {
	bundleDir := writeCLIBundle(t)
	ticketPath := filepath.Join(t.TempDir(), "ticket.md")
	mustWrite(t, ticketPath, "Login failed after submit")
	var stdout, stderr bytes.Buffer

	code := Run([]string{"analyze", bundleDir, "--ticket", ticketPath}, &stdout, &stderr, "test")

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	for _, want := range []string{"## Summary", "Status: match", "frames/frame_0001.png"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected output to contain %q, got %q", want, stdout.String())
		}
	}
}

func TestValidateJSON(t *testing.T) {
	bundleDir := writeCLIBundle(t)
	mustWrite(t, filepath.Join(bundleDir, "frames", "frame_0001.png"), "fake frame")
	mustWrite(t, filepath.Join(bundleDir, "ocr", "frame_0001.txt"), "Login failed")
	var stdout, stderr bytes.Buffer

	code := Run([]string{"validate", bundleDir, "--json"}, &stdout, &stderr, "test")

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	var report struct {
		OK              bool `json:"ok"`
		TimelineEntries int  `json:"timeline_entries"`
		Checks          []struct {
			Name string `json:"name"`
			OK   bool   `json:"ok"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", stdout.String(), err)
	}
	if !report.OK || report.TimelineEntries != 1 {
		t.Fatalf("unexpected report: %#v", report)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestValidateJSONFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"validate", "/missing/bundle", "--json"}, &stdout, &stderr, "test")

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stdout.String(), `"ok": false`) {
		t.Fatalf("expected json failure report, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr for json failure, got %q", stderr.String())
	}
}

func TestIndexAndSearchJSON(t *testing.T) {
	bundleDir := writeCLIBundle(t)
	mustWrite(t, filepath.Join(bundleDir, "frames", "frame_0001.png"), "fake frame")
	mustWrite(t, filepath.Join(bundleDir, "ocr", "frame_0001.txt"), "Login failed after submit")
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")

	var indexStdout, indexStderr bytes.Buffer
	indexCode := Run([]string{"index", bundleDir, "--db", dbPath, "--json"}, &indexStdout, &indexStderr, "test")
	if indexCode != 0 {
		t.Fatalf("expected index exit code 0, got %d stderr=%q stdout=%q", indexCode, indexStderr.String(), indexStdout.String())
	}
	var indexReport struct {
		OK             bool   `json:"ok"`
		Collection     string `json:"collection"`
		IndexedEntries int    `json:"indexed_entries"`
	}
	if err := json.Unmarshal(indexStdout.Bytes(), &indexReport); err != nil {
		t.Fatalf("expected index JSON, got %q: %v", indexStdout.String(), err)
	}
	if !indexReport.OK || indexReport.Collection == "" || indexReport.IndexedEntries != 1 {
		t.Fatalf("unexpected index report: %#v", indexReport)
	}
	if indexStderr.Len() != 0 {
		t.Fatalf("expected empty index stderr, got %q", indexStderr.String())
	}

	var searchStdout, searchStderr bytes.Buffer
	searchCode := Run([]string{"search", dbPath, "Login failed", "--json"}, &searchStdout, &searchStderr, "test")
	if searchCode != 0 {
		t.Fatalf("expected search exit code 0, got %d stderr=%q stdout=%q", searchCode, searchStderr.String(), searchStdout.String())
	}
	var searchReport struct {
		OK      bool `json:"ok"`
		Results []struct {
			Frame      string `json:"frame"`
			OCR        string `json:"ocr"`
			Transcript string `json:"transcript"`
		} `json:"results"`
	}
	if err := json.Unmarshal(searchStdout.Bytes(), &searchReport); err != nil {
		t.Fatalf("expected search JSON, got %q: %v", searchStdout.String(), err)
	}
	if !searchReport.OK || len(searchReport.Results) != 1 {
		t.Fatalf("unexpected search report: %#v", searchReport)
	}
	if searchReport.Results[0].Frame != "frames/frame_0001.png" || searchReport.Results[0].OCR == "" {
		t.Fatalf("unexpected search result: %#v", searchReport.Results[0])
	}
	if searchStderr.Len() != 0 {
		t.Fatalf("expected empty search stderr, got %q", searchStderr.String())
	}
}

func TestSearchFiltersJSON(t *testing.T) {
	bundleDir := writeCLIBundle(t)
	mustWrite(t, filepath.Join(bundleDir, "frames", "frame_0001.png"), "fake frame")
	mustWrite(t, filepath.Join(bundleDir, "ocr", "frame_0001.txt"), "Login failed after submit")
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")

	var idxOut, idxErr bytes.Buffer
	if code := Run([]string{"index", bundleDir, "--db", dbPath, "--json"}, &idxOut, &idxErr, "test"); code != 0 {
		t.Fatalf("index failed: code=%d stderr=%q", code, idxErr.String())
	}

	type searchReport struct {
		OK      bool `json:"ok"`
		Filters *struct {
			SourceVideo string   `json:"source_video"`
			MaxTime     *float64 `json:"max_time"`
		} `json:"filters"`
		Results []struct {
			TimeSeconds float64 `json:"time_seconds"`
		} `json:"results"`
	}

	runSearchJSON := func(t *testing.T, extra ...string) searchReport {
		t.Helper()
		args := append([]string{"search", dbPath, "Login failed"}, extra...)
		args = append(args, "--json")
		var out, errBuf bytes.Buffer
		if code := Run(args, &out, &errBuf, "test"); code != 0 {
			t.Fatalf("search %v failed: code=%d stderr=%q", extra, code, errBuf.String())
		}
		if errBuf.Len() != 0 {
			t.Fatalf("expected empty stderr, got %q", errBuf.String())
		}
		var rep searchReport
		if err := json.Unmarshal(out.Bytes(), &rep); err != nil {
			t.Fatalf("expected search JSON, got %q: %v", out.String(), err)
		}
		return rep
	}

	match := runSearchJSON(t, "--source-video", "/tmp/login-bug.mp4")
	if !match.OK || len(match.Results) != 1 {
		t.Fatalf("expected one result for matching source video, got %#v", match)
	}
	if match.Filters == nil || match.Filters.SourceVideo != "/tmp/login-bug.mp4" {
		t.Fatalf("expected echoed source-video filter, got %#v", match.Filters)
	}

	none := runSearchJSON(t, "--source-video", "/tmp/other.mp4")
	if !none.OK || len(none.Results) != 0 {
		t.Fatalf("expected zero results for non-matching source video, got %#v", none)
	}

	withinTime := runSearchJSON(t, "--max-time", "0")
	if len(withinTime.Results) != 1 || withinTime.Filters == nil || withinTime.Filters.MaxTime == nil {
		t.Fatalf("expected one result within max-time window with echoed filter, got %#v", withinTime)
	}

	afterTime := runSearchJSON(t, "--min-time", "5")
	if len(afterTime.Results) != 0 {
		t.Fatalf("expected zero results after time window, got %#v", afterTime)
	}

	plain := runSearchJSON(t)
	if plain.Filters != nil {
		t.Fatalf("expected nil filters echo for unfiltered search, got %#v", plain.Filters)
	}
}

func TestIndexMultipleBundlesJSON(t *testing.T) {
	mkBundle := func() string {
		d := writeCLIBundle(t)
		mustWrite(t, filepath.Join(d, "frames", "frame_0001.png"), "fake frame")
		mustWrite(t, filepath.Join(d, "ocr", "frame_0001.txt"), "Login failed after submit")
		return d
	}
	bundleA := mkBundle()
	bundleB := mkBundle()
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")

	var out, errBuf bytes.Buffer
	code := Run([]string{"index", bundleA, bundleB, "--db", dbPath, "--json"}, &out, &errBuf, "test")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q stdout=%q", code, errBuf.String(), out.String())
	}
	if errBuf.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", errBuf.String())
	}

	var report struct {
		OK             bool `json:"ok"`
		IndexedEntries int  `json:"indexed_entries"`
		Bundles        []struct {
			BundleDir      string `json:"bundle_dir"`
			IndexedEntries int    `json:"indexed_entries"`
		} `json:"bundles"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("expected multi-index JSON, got %q: %v", out.String(), err)
	}
	if !report.OK || report.IndexedEntries != 2 || len(report.Bundles) != 2 {
		t.Fatalf("unexpected multi-index report: %#v", report)
	}
	for _, b := range report.Bundles {
		if b.IndexedEntries != 1 || b.BundleDir == "" {
			t.Fatalf("unexpected per-bundle entry: %#v", b)
		}
	}
}

func TestIndexSingleBundleJSONKeepsLegacyShape(t *testing.T) {
	bundleDir := writeCLIBundle(t)
	mustWrite(t, filepath.Join(bundleDir, "frames", "frame_0001.png"), "fake frame")
	mustWrite(t, filepath.Join(bundleDir, "ocr", "frame_0001.txt"), "Login failed after submit")
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")

	var out, errBuf bytes.Buffer
	if code := Run([]string{"index", bundleDir, "--db", dbPath, "--json"}, &out, &errBuf, "test"); code != 0 {
		t.Fatalf("index failed: code=%d stderr=%q", code, errBuf.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &raw); err != nil {
		t.Fatalf("expected JSON, got %q: %v", out.String(), err)
	}
	if _, ok := raw["bundle_dir"]; !ok {
		t.Fatalf("single-bundle JSON must keep bundle_dir, got keys: %v", out.String())
	}
	if _, ok := raw["bundles"]; ok {
		t.Fatalf("single-bundle JSON must not include a bundles array: %v", out.String())
	}
}

func TestIndexHumanOutput(t *testing.T) {
	mkBundle := func() string {
		d := writeCLIBundle(t)
		mustWrite(t, filepath.Join(d, "frames", "frame_0001.png"), "fake frame")
		mustWrite(t, filepath.Join(d, "ocr", "frame_0001.txt"), "Login failed after submit")
		return d
	}

	var singleOut, singleErr bytes.Buffer
	if code := Run([]string{"index", mkBundle(), "--db", filepath.Join(t.TempDir(), "s.veclite")}, &singleOut, &singleErr, "test"); code != 0 {
		t.Fatalf("single index failed: code=%d stderr=%q", code, singleErr.String())
	}
	for _, want := range []string{"vidtrace index: ok", "Bundle:", "Entries:"} {
		if !strings.Contains(singleOut.String(), want) {
			t.Fatalf("single human output missing %q, got:\n%s", want, singleOut.String())
		}
	}

	var multiOut, multiErr bytes.Buffer
	if code := Run([]string{"index", mkBundle(), mkBundle(), "--db", filepath.Join(t.TempDir(), "m.veclite")}, &multiOut, &multiErr, "test"); code != 0 {
		t.Fatalf("multi index failed: code=%d stderr=%q", code, multiErr.String())
	}
	for _, want := range []string{"vidtrace index: ok", "Bundles: 2", "Total:"} {
		if !strings.Contains(multiOut.String(), want) {
			t.Fatalf("multi human output missing %q, got:\n%s", want, multiOut.String())
		}
	}
}

func TestSearchBundleAndSourceFiltersJSON(t *testing.T) {
	bundleDir := writeCLIBundle(t)
	mustWrite(t, filepath.Join(bundleDir, "frames", "frame_0001.png"), "fake frame")
	mustWrite(t, filepath.Join(bundleDir, "ocr", "frame_0001.txt"), "Login failed after submit")
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")

	var idxOut, idxErr bytes.Buffer
	if code := Run([]string{"index", bundleDir, "--db", dbPath, "--json"}, &idxOut, &idxErr, "test"); code != 0 {
		t.Fatalf("index failed: code=%d stderr=%q", code, idxErr.String())
	}
	absBundle, err := filepath.Abs(bundleDir)
	if err != nil {
		t.Fatal(err)
	}

	type report struct {
		OK      bool `json:"ok"`
		Filters *struct {
			Bundle  string   `json:"bundle"`
			Source  string   `json:"source"`
			MinTime *float64 `json:"min_time"`
		} `json:"filters"`
		Results []struct {
			TimeSeconds float64 `json:"time_seconds"`
		} `json:"results"`
	}
	run := func(t *testing.T, extra ...string) report {
		t.Helper()
		args := append([]string{"search", dbPath, "Login failed"}, extra...)
		args = append(args, "--json")
		var out, errBuf bytes.Buffer
		if code := Run(args, &out, &errBuf, "test"); code != 0 {
			t.Fatalf("search %v failed: code=%d stderr=%q", extra, code, errBuf.String())
		}
		var rep report
		if err := json.Unmarshal(out.Bytes(), &rep); err != nil {
			t.Fatalf("expected search JSON, got %q: %v", out.String(), err)
		}
		return rep
	}

	// --bundle round-trips through expandHome + filepath.Abs and matches the indexed absolute path.
	matched := run(t, "--bundle", bundleDir)
	if len(matched.Results) != 1 || matched.Filters == nil || matched.Filters.Bundle != absBundle {
		t.Fatalf("expected one result with echoed bundle %q, got %#v", absBundle, matched)
	}

	unmatched := run(t, "--bundle", filepath.Join(t.TempDir(), "other-bundle"))
	if len(unmatched.Results) != 0 {
		t.Fatalf("expected zero results for non-matching bundle, got %#v", unmatched.Results)
	}

	// --source is otherwise entirely untested.
	sourceMatch := run(t, "--source", "timeline")
	if len(sourceMatch.Results) != 1 || sourceMatch.Filters == nil || sourceMatch.Filters.Source != "timeline" {
		t.Fatalf("expected one result with echoed source filter, got %#v", sourceMatch)
	}
	sourceNone := run(t, "--source", "bogus")
	if len(sourceNone.Results) != 0 {
		t.Fatalf("expected zero results for unknown source, got %#v", sourceNone.Results)
	}

	// --min-time 0 must be treated as explicitly set and echoed (no -1 sentinel).
	minZero := run(t, "--min-time", "0")
	if minZero.Filters == nil || minZero.Filters.MinTime == nil || *minZero.Filters.MinTime != 0 {
		t.Fatalf("expected echoed min_time=0, got %#v", minZero.Filters)
	}
}

func TestMCPRejectsExtraArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"mcp", "unexpected"}, &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("expected exit 2 for extra args, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage: vidtrace mcp") {
		t.Fatalf("expected usage message, got %q", stderr.String())
	}
}

func TestHelpListsMCPCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	Run([]string{"help"}, &stdout, &stderr, "test")
	if !strings.Contains(stdout.String(), "mcp ") {
		t.Fatalf("expected help to list the mcp command, got:\n%s", stdout.String())
	}
}

func TestSemanticIndexAndSearchViaOllamaHTTP(t *testing.T) {
	// A stand-in Ollama server returns deterministic fixed-dimension vectors so
	// the CLI -> embed -> evidence wiring is exercised without a live model.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		vecs := make([][]float32, len(req.Input))
		for i := range req.Input {
			vecs[i] = []float32{1, 0.5, 0.25, 0.125}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"embeddings": vecs})
	}))
	defer server.Close()

	bundleDir := writeCLIBundle(t)
	mustWrite(t, filepath.Join(bundleDir, "frames", "frame_0001.png"), "fake frame")
	mustWrite(t, filepath.Join(bundleDir, "ocr", "frame_0001.txt"), "Login failed after submit")
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")

	var idxOut, idxErr bytes.Buffer
	idxCode := Run([]string{"index", bundleDir, "--db", dbPath, "--embed", "ollama", "--embed-model", "nomic-embed-text", "--ollama-url", server.URL, "--json"}, &idxOut, &idxErr, "test")
	if idxCode != 0 {
		t.Fatalf("semantic index failed: code=%d stderr=%q", idxCode, idxErr.String())
	}
	var idxReport struct {
		OK              bool `json:"ok"`
		SemanticEntries int  `json:"semantic_entries"`
		Embedding       *struct {
			Provider   string `json:"provider"`
			Dimensions int    `json:"dimensions"`
		} `json:"embedding"`
	}
	if err := json.Unmarshal(idxOut.Bytes(), &idxReport); err != nil {
		t.Fatalf("expected index JSON, got %q: %v", idxOut.String(), err)
	}
	if !idxReport.OK || idxReport.SemanticEntries != 1 || idxReport.Embedding == nil || idxReport.Embedding.Provider != "ollama" || idxReport.Embedding.Dimensions != 4 {
		t.Fatalf("unexpected semantic index report: %#v", idxReport)
	}

	var searchOut, searchErr bytes.Buffer
	searchCode := Run([]string{"search", dbPath, "login problem", "--mode", "hybrid", "--embed", "ollama", "--embed-model", "nomic-embed-text", "--ollama-url", server.URL, "--json"}, &searchOut, &searchErr, "test")
	if searchCode != 0 {
		t.Fatalf("semantic search failed: code=%d stderr=%q", searchCode, searchErr.String())
	}
	var searchReport struct {
		OK      bool   `json:"ok"`
		Mode    string `json:"mode"`
		Results []struct {
			Frame string `json:"frame"`
		} `json:"results"`
	}
	if err := json.Unmarshal(searchOut.Bytes(), &searchReport); err != nil {
		t.Fatalf("expected search JSON, got %q: %v", searchOut.String(), err)
	}
	if !searchReport.OK || searchReport.Mode != "hybrid" || len(searchReport.Results) == 0 {
		t.Fatalf("unexpected hybrid search report: %#v", searchReport)
	}
}

func TestSemanticSearchWithoutEmbedderJSONFailure(t *testing.T) {
	bundleDir := writeCLIBundle(t)
	mustWrite(t, filepath.Join(bundleDir, "frames", "frame_0001.png"), "fake frame")
	mustWrite(t, filepath.Join(bundleDir, "ocr", "frame_0001.txt"), "Login failed after submit")
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	if code := Run([]string{"index", bundleDir, "--db", dbPath, "--json"}, &bytes.Buffer{}, &bytes.Buffer{}, "test"); code != 0 {
		t.Fatalf("keyword index failed: %d", code)
	}

	var out, errBuf bytes.Buffer
	code := Run([]string{"search", dbPath, "login", "--mode", "semantic", "--json"}, &out, &errBuf, "test")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(out.String(), `"ok": false`) || !strings.Contains(out.String(), "requires an embedding provider") {
		t.Fatalf("expected embedding-provider error JSON, got %q", out.String())
	}
}

func TestIndexRejectsUnknownEmbedProviderJSON(t *testing.T) {
	bundleDir := writeCLIBundle(t)
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	var out, errBuf bytes.Buffer
	code := Run([]string{"index", bundleDir, "--db", dbPath, "--embed", "magic", "--json"}, &out, &errBuf, "test")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(out.String(), `"ok": false`) || !strings.Contains(out.String(), "unknown embedding provider") {
		t.Fatalf("expected unknown-provider error JSON, got %q", out.String())
	}
}

func TestSearchInvertedTimeRangeJSONFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"search", "/tmp/whatever.veclite", "ticket", "--min-time", "10", "--max-time", "5", "--json"}, &stdout, &stderr, "test")
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stdout.String(), `"ok": false`) || !strings.Contains(stdout.String(), "greater than max-time") {
		t.Fatalf("expected inverted time-range json failure, got %q", stdout.String())
	}
}

func TestSearchJSONFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"search", "/missing/evidence.veclite", "ticket", "--json"}, &stdout, &stderr, "test")

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stdout.String(), `"ok": false`) {
		t.Fatalf("expected json failure, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr for json failure, got %q", stderr.String())
	}
}

func TestMigrateEvidenceJSONAlreadyMigrated(t *testing.T) {
	bundleDir := writeCLIBundle(t)
	mustWrite(t, filepath.Join(bundleDir, "frames", "frame_0001.png"), "fake frame")
	mustWrite(t, filepath.Join(bundleDir, "ocr", "frame_0001.txt"), "Login failed after submit")
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")

	var idxOut, idxErr bytes.Buffer
	if code := Run([]string{"index", bundleDir, "--db", dbPath, "--json"}, &idxOut, &idxErr, "test"); code != 0 {
		t.Fatalf("index failed: code=%d stderr=%q", code, idxErr.String())
	}

	var out, errBuf bytes.Buffer
	code := Run([]string{"migrate-evidence", dbPath, "--json"}, &out, &errBuf, "test")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, errBuf.String(), out.String())
	}
	var report struct {
		OK              bool   `json:"ok"`
		Collection      string `json:"collection"`
		AlreadyMigrated bool   `json:"already_migrated"`
		MigratedRecords int    `json:"migrated_records"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("expected migrate JSON, got %q: %v", out.String(), err)
	}
	if !report.OK || !report.AlreadyMigrated || report.MigratedRecords != 0 || report.Collection != "evidence_entries" {
		t.Fatalf("unexpected migrate report: %#v", report)
	}
}

func TestMigrateEvidenceMissingDBJSONFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"migrate-evidence", "/missing/evidence.veclite", "--json"}, &stdout, &stderr, "test")
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stdout.String(), `"ok": false`) || !strings.Contains(stdout.String(), "evidence db not found") {
		t.Fatalf("expected json failure with not-found, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr for json failure, got %q", stderr.String())
	}
}

func TestConciseEvidenceTextTruncatesHumanSearchText(t *testing.T) {
	result := evidence.SearchResult{
		Frame: "frames/frame_0001.png",
		OCR:   "Visible title\n" + strings.Repeat("dense OCR ", 20),
	}

	output := conciseEvidenceText(result, 40)
	if len(output) > 40 || !strings.HasSuffix(output, "...") {
		t.Fatalf("expected truncated evidence text, got %q", output)
	}
	if strings.Contains(output, "\n") {
		t.Fatalf("expected single-line evidence text, got %q", output)
	}
}

func TestInvestigateJSON(t *testing.T) {
	bundleDir := writeCLIBundle(t)
	mustWrite(t, filepath.Join(bundleDir, "frames", "frame_0001.png"), "fake frame")
	mustWrite(t, filepath.Join(bundleDir, "ocr", "frame_0001.txt"), "Login failed after submit")
	var stdout, stderr bytes.Buffer

	code := Run([]string{"investigate", bundleDir, "--query", "login failed", "--codebase", "/tmp/app", "--json"}, &stdout, &stderr, "test")

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	var report struct {
		OK              bool     `json:"ok"`
		Evidence        []any    `json:"evidence"`
		Suggested       []string `json:"suggested_queries"`
		VecgrepCommands []string `json:"vecgrep_commands"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("expected investigate JSON, got %q: %v", stdout.String(), err)
	}
	if !report.OK || len(report.Evidence) == 0 || len(report.Suggested) == 0 || len(report.VecgrepCommands) == 0 {
		t.Fatalf("unexpected investigate report: %#v", report)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestInvestigateMarkdown(t *testing.T) {
	bundleDir := writeCLIBundle(t)
	mustWrite(t, filepath.Join(bundleDir, "frames", "frame_0001.png"), "fake frame")
	mustWrite(t, filepath.Join(bundleDir, "ocr", "frame_0001.txt"), "Login failed after submit")
	var stdout, stderr bytes.Buffer

	code := Run([]string{"investigate", bundleDir, "--query", "login failed"}, &stdout, &stderr, "test")

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", code, stderr.String())
	}
	for _, want := range []string{"# Investigation Handoff", "## Video Evidence", "frames/frame_0001.png", "## Suggested Code Searches"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected markdown to contain %q, got %q", want, stdout.String())
		}
	}
}

func TestInvestigateRequiresQuery(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"investigate", "/tmp/bundle"}, &stdout, &stderr, "test")

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "missing required --query") {
		t.Fatalf("expected query error, got %q", stderr.String())
	}
}

func TestInvestigateConnectRequiresCodebase(t *testing.T) {
	var stdout, stderr bytes.Buffer

	bundleDir := writeCLIBundle(t)
	code := Run([]string{"investigate", bundleDir, "--query", "test", "--connect"}, &stdout, &stderr, "test")

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "--connect requires --codebase") {
		t.Fatalf("expected connect-requires-codebase error, got %q", stderr.String())
	}
}

func TestNormalizeExtractArgsAllowsFlagsAfterPath(t *testing.T) {
	args, err := normalizeExtractArgs([]string{"/tmp/bug.mp4", "--fps", "2", "--json", "--out=/tmp/out"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"--fps", "2", "--json", "--out=/tmp/out", "/tmp/bug.mp4"}
	if strings.Join(args, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("unexpected args: got %v want %v", args, want)
	}
}

func writeCLIBundle(t *testing.T) string {
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}
