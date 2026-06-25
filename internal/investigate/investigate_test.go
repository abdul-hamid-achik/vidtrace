package investigate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abdul-hamid-achik/vidtrace/internal/codemap"
	"github.com/abdul-hamid-achik/vidtrace/internal/evidence"
	"github.com/abdul-hamid-achik/vidtrace/internal/fcheap"
)

func TestRunReturnsEvidenceAndVecgrepCommands(t *testing.T) {
	bundleDir := writeInvestigateBundle(t)
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")

	report, err := Run(Options{
		BundleDir:   bundleDir,
		Query:       "clicking a ticket does not work",
		DBPath:      dbPath,
		CodebaseDir: t.TempDir(),
		Limit:       3,
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !report.OK || len(report.Evidence) == 0 {
		t.Fatalf("unexpected report: %#v", report)
	}
	if report.Evidence[0].Frame != "frames/frame_0002.png" {
		t.Fatalf("first evidence frame = %q", report.Evidence[0].Frame)
	}
	if len(report.SuggestedQueries) == 0 || report.SuggestedQueries[0] != "clicking a ticket does not work" {
		t.Fatalf("unexpected suggestions: %#v", report.SuggestedQueries)
	}
	if !containsString(report.SuggestedQueries, "OPG-14010") {
		t.Fatalf("expected OPG-14010 suggestion: %#v", report.SuggestedQueries)
	}
	if len(report.VecgrepCommands) == 0 || !strings.Contains(report.VecgrepCommands[0], "vecgrep search") {
		t.Fatalf("expected vecgrep commands: %#v", report.VecgrepCommands)
	}
}

func TestRunUsesTemporaryDBWhenDBPathOmitted(t *testing.T) {
	report, err := Run(Options{
		BundleDir: writeInvestigateBundle(t),
		Query:     "ticket",
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !report.TemporaryDB || report.DBPath != "" {
		t.Fatalf("expected temporary db without persistent path: %#v", report)
	}
}

func TestMarkdownIncludesEvidenceAndSuggestedSearches(t *testing.T) {
	report, err := Run(Options{
		BundleDir:   writeInvestigateBundle(t),
		Query:       "ticket does not work",
		CodebaseDir: "/tmp/app repo",
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	out := Markdown(report)
	for _, want := range []string{
		"# Investigation Handoff",
		"## Video Evidence",
		"frames/frame_0002.png",
		"## Suggested Code Searches",
		"vecgrep search",
		"'/tmp/app repo'",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected markdown to contain %q, got:\n%s", want, out)
		}
	}
}

func TestVecgrepCommandsQuotesSingleQuotes(t *testing.T) {
	commands := VecgrepCommands("/tmp/abdul's app", []string{"ticket's route"})
	if len(commands) != 1 {
		t.Fatalf("commands = %#v", commands)
	}
	if !strings.Contains(commands[0], "'/tmp/abdul'\"'\"'s app'") {
		t.Fatalf("codebase path was not shell-quoted: %s", commands[0])
	}
	if !strings.Contains(commands[0], "'ticket'\"'\"'s route'") {
		t.Fatalf("query was not shell-quoted: %s", commands[0])
	}
}

func TestRunRejectsMissingQuery(t *testing.T) {
	_, err := Run(Options{BundleDir: writeInvestigateBundle(t)})
	if err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("Run error = %v, want query error", err)
	}
}

func TestSuggestedCodeQueriesFiltersOCRNoise(t *testing.T) {
	results := []evidence.SearchResult{
		{
			OCR:        "https localhost Jun 2026 Monday example.com Checkout failed",
			Transcript: "the assessment page did not open",
		},
	}
	suggestions := SuggestedCodeQueries("bug report", results)
	joined := strings.ToLower(strings.Join(suggestions, " | "))

	for _, noise := range []string{"https", "localhost", "jun", "2026", "monday", "example.com"} {
		if strings.Contains(joined, noise) {
			t.Fatalf("expected %q to be filtered from suggestions, got: %#v", noise, suggestions)
		}
	}
	for _, signal := range []string{"checkout", "failed", "assessment"} {
		if !strings.Contains(joined, signal) {
			t.Fatalf("expected signal word %q to remain in suggestions, got: %#v", signal, suggestions)
		}
	}
}

func TestSuggestedCodeQueriesKeepsCodeLikeTokens(t *testing.T) {
	results := []evidence.SearchResult{
		{OCR: "Ticket OPG-14010 details 2026", Transcript: "the ticket fails"},
	}
	suggestions := SuggestedCodeQueries("ticket", results)

	if !containsString(suggestions, "OPG-14010") {
		t.Fatalf("expected code-like token OPG-14010 to be preserved: %#v", suggestions)
	}
	for _, s := range suggestions {
		if s == "2026" {
			t.Fatalf("four-digit year leaked as a suggestion: %#v", suggestions)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func writeInvestigateBundle(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	mustMkdirInvestigate(t, filepath.Join(dir, "frames"))
	mustMkdirInvestigate(t, filepath.Join(dir, "ocr"))
	mustMkdirInvestigate(t, filepath.Join(dir, "transcript"))
	mustWriteInvestigate(t, filepath.Join(dir, "frames", "frame_0001.png"), "fake frame 1")
	mustWriteInvestigate(t, filepath.Join(dir, "frames", "frame_0002.png"), "fake frame 2")
	mustWriteInvestigate(t, filepath.Join(dir, "ocr", "frame_0001.txt"), "Tickets")
	mustWriteInvestigate(t, filepath.Join(dir, "ocr", "frame_0002.txt"), "Ticket OPG-14010 details")
	mustWriteInvestigate(t, filepath.Join(dir, "ocr", "ocr_all_frames.txt"), "Tickets\nTicket OPG-14010 details\n")
	mustWriteInvestigate(t, filepath.Join(dir, "metadata.json"), `{
  "schema_version": "1",
  "source_video": "/tmp/ticket-bug.mp4",
  "duration_seconds": 2,
  "extract_fps": 1,
  "ocr_languages": ["eng"],
  "whisper_language": "en",
  "whisper_model": "small"
}`)
	mustWriteInvestigate(t, filepath.Join(dir, "timeline.json"), `{
  "schema_version": "1",
  "entries": [
    {
      "time_seconds": 0,
      "frame": "frames/frame_0001.png",
      "ocr": {"path": "ocr/frame_0001.txt", "text": "Tickets"},
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

func TestRunBackwardCompatibleWithoutConnectOrStash(t *testing.T) {
	report, err := Run(Options{
		BundleDir: writeInvestigateBundle(t),
		Query:     "ticket",
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if report.StashID != "" {
		t.Fatalf("stash_id should be empty without --stash: %q", report.StashID)
	}
	if len(report.CodeMatches) != 0 {
		t.Fatalf("code_matches should be empty without --connect: %#v", report.CodeMatches)
	}
	if report.ConnectError != "" {
		t.Fatalf("connect_error should be empty without --connect: %q", report.ConnectError)
	}
}

func TestRunConnectErrorWhenFcheapUnavailable(t *testing.T) {
	// This test verifies that --connect gracefully degrades when fcheap is
	// not available. If fcheap IS installed in the test environment, we skip
	// because the connect will actually try to run.
	if fcheap.Available() {
		t.Skip("fcheap is installed; connect error path not testable without mocking")
	}

	report, err := Run(Options{
		BundleDir:   writeInvestigateBundle(t),
		Query:       "ticket",
		CodebaseDir: t.TempDir(),
		Connect:     true,
	})
	if err != nil {
		t.Fatalf("Run should not fail when connect fails: %v", err)
	}
	if !report.OK {
		t.Fatalf("report should be OK even with connect error")
	}
	if report.ConnectError == "" {
		t.Fatalf("connect_error should be set when fcheap unavailable")
	}
	if len(report.CodeMatches) != 0 {
		t.Fatalf("code_matches should be empty on connect error: %#v", report.CodeMatches)
	}
}

func TestRunStashErrorWhenFcheapUnavailable(t *testing.T) {
	if fcheap.Available() {
		t.Skip("fcheap is installed; stash error path not testable without mocking")
	}

	_, err := Run(Options{
		StashID: "some_stash_id",
		Query:   "ticket",
	})
	if err == nil {
		t.Fatalf("Run should fail when stash is requested but fcheap unavailable")
	}
	if !strings.Contains(err.Error(), "fcheap is not installed") {
		t.Fatalf("error should mention fcheap not installed: %v", err)
	}
}

func TestMarkdownRendersCodeMatchesAndStash(t *testing.T) {
	report := Report{
		OK:      true,
		Query:   "ticket bug",
		Mode:    "keyword",
		Summary: "Found 1 video evidence hit(s).",
		Evidence: []evidence.SearchResult{
			{TimeSeconds: 1.0, Frame: "frames/frame_0002.png", OCR: "Ticket OPG-14010"},
		},
		SuggestedQueries: []string{"ticket bug"},
		StashID:          "test_stash_123",
		CodeMatches: []fcheap.CodeMatch{
			{File: "src/ticket.go", Score: 0.85, Text: "func handleTicketClick()"},
			{File: "src/routes.go", Score: 0.72, Text: "router.GET('/ticket/:id')"},
		},
	}

	out := Markdown(report)
	for _, want := range []string{
		"## Stash",
		"test_stash_123",
		"## Code Matches",
		"src/ticket.go",
		"src/routes.go",
		"func handleTicketClick()",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected markdown to contain %q, got:\n%s", want, out)
		}
	}
}

func TestMarkdownRendersConnectError(t *testing.T) {
	report := Report{
		OK:           true,
		Query:        "ticket bug",
		Mode:         "keyword",
		Summary:      "Found 0 video evidence hit(s).",
		ConnectError: "fcheap is not installed or not on PATH",
	}

	out := Markdown(report)
	if !strings.Contains(out, "## Connect Error") {
		t.Fatalf("expected markdown to contain Connect Error section, got:\n%s", out)
	}
	if !strings.Contains(out, "fcheap is not installed") {
		t.Fatalf("expected markdown to contain the error text")
	}
}

func TestMarkdownNoStashOrCodeMatchesWhenEmpty(t *testing.T) {
	report := Report{
		OK:      true,
		Query:   "ticket",
		Mode:    "keyword",
		Summary: "Found 0 hits.",
	}

	out := Markdown(report)
	if strings.Contains(out, "## Stash") {
		t.Fatalf("markdown should not contain Stash section when StashID is empty")
	}
	if strings.Contains(out, "## Code Matches") {
		t.Fatalf("markdown should not contain Code Matches section when empty")
	}
	if strings.Contains(out, "## Connect Error") {
		t.Fatalf("markdown should not contain Connect Error section when empty")
	}
}

func TestSummaryMentionsCodeMatches(t *testing.T) {
	results := []evidence.SearchResult{
		{TimeSeconds: 1.0, Frame: "frames/frame_0001.png", OCR: "Login failed"},
	}
	matches := []fcheap.CodeMatch{
		{File: "src/auth.go", Score: 0.85, Text: "func login()"},
		{File: "src/routes.go", Score: 0.72, Text: "router.POST('/login')"},
	}

	got := summary(results, []string{"login failed"}, "/tmp/repo", matches, nil)
	if !strings.Contains(got, "2 code match(es) found via fcheap connect") {
		t.Fatalf("expected summary to mention code matches, got: %s", got)
	}
	if strings.Contains(got, "codemap") {
		t.Fatalf("summary should not mention codemap when expansion is nil, got: %s", got)
	}
}

func TestSummaryOmitsCodeMatchesWhenEmpty(t *testing.T) {
	results := []evidence.SearchResult{
		{TimeSeconds: 1.0, Frame: "frames/frame_0001.png", OCR: "Login failed"},
	}

	got := summary(results, []string{"login failed"}, "/tmp/repo", nil, nil)
	if strings.Contains(got, "code match(es)") {
		t.Fatalf("summary should not mention code matches when empty, got: %s", got)
	}
}

func TestSummaryMentionsCodemapExpansion(t *testing.T) {
	results := []evidence.SearchResult{
		{TimeSeconds: 1.0, Frame: "frames/frame_0001.png", OCR: "Login failed"},
	}
	matches := []fcheap.CodeMatch{
		{File: "src/auth.go", Score: 0.85, Text: "func login()"},
	}
	expansion := &CodemapExpansion{
		Symbols: []SymbolExpansion{
			{Symbol: "login", File: "src/auth.go", Line: 10},
			{Symbol: "handleLogin", File: "src/routes.go", Line: 25},
		},
	}

	got := summary(results, []string{"login failed"}, "/tmp/repo", matches, expansion)
	if !strings.Contains(got, "1 code match(es) found via fcheap connect") {
		t.Fatalf("expected summary to mention code matches, got: %s", got)
	}
	if !strings.Contains(got, "2 symbol(s) expanded via codemap") {
		t.Fatalf("expected summary to mention codemap expansion, got: %s", got)
	}
}

func TestMarkdownRendersCodemapExpansion(t *testing.T) {
	report := Report{
		OK:      true,
		Query:   "ticket bug",
		Mode:    "keyword",
		Summary: "Found 1 video evidence hit(s).",
		CodemapExpansion: &CodemapExpansion{
			Symbols: []SymbolExpansion{
				{
					Symbol:     "handleTicketClick",
					File:       "src/ticket.go",
					Line:       42,
					Kind:       "function",
					Resolution: "exact",
					Callers: []codemap.Symbol{
						{Symbol: "router.ServeHTTP", File: "src/router.go", StartLine: 15},
					},
					BlastRadius: []codemap.Symbol{
						{Symbol: "main", File: "main.go", StartLine: 5},
					},
					Tested: true,
				},
			},
		},
	}

	out := Markdown(report)
	for _, want := range []string{
		"## Code Graph Expansion",
		"handleTicketClick",
		"src/ticket.go:42",
		"Callers (1)",
		"router.ServeHTTP",
		"Blast radius (1 symbols)",
		"Test coverage: yes",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected markdown to contain %q, got:\n%s", want, out)
		}
	}
}

func TestMarkdownRendersCodemapError(t *testing.T) {
	report := Report{
		OK:           true,
		Query:        "ticket bug",
		Mode:         "keyword",
		Summary:      "Found 0 video evidence hit(s).",
		CodemapError: "codemap is not installed or not on PATH",
	}

	out := Markdown(report)
	if !strings.Contains(out, "## Codemap Error") {
		t.Fatalf("expected markdown to contain Codemap Error section, got:\n%s", out)
	}
	if !strings.Contains(out, "codemap is not installed") {
		t.Fatalf("expected markdown to contain the codemap error text")
	}
}

func TestMarkdownNoCodemapWhenEmpty(t *testing.T) {
	report := Report{
		OK:      true,
		Query:   "ticket",
		Mode:    "keyword",
		Summary: "Found 0 hits.",
	}

	out := Markdown(report)
	if strings.Contains(out, "## Code Graph Expansion") {
		t.Fatalf("markdown should not contain Code Graph Expansion when nil")
	}
	if strings.Contains(out, "## Codemap Error") {
		t.Fatalf("markdown should not contain Codemap Error when empty")
	}
}

func TestRunCodemapErrorWhenCodemapUnavailable(t *testing.T) {
	// This test verifies that --codemap gracefully degrades when codemap is
	// not available. We need code matches for codemap to run, but codemap
	// expansion only runs when len(codeMatches) > 0, which requires fcheap.
	// So we test the runCodemapExpansion function directly.
	if codemap.Available() {
		t.Skip("codemap is installed; error path not testable without mocking")
	}

	matches := []fcheap.CodeMatch{
		{File: "src/ticket.go:42", Score: 0.85, Text: "func handleTicketClick()"},
	}

	expansion, errStr := runCodemapExpansion(matches, nil, "ticket", Options{})
	if expansion != nil {
		t.Fatalf("expansion should be nil when codemap unavailable")
	}
	if errStr == "" {
		t.Fatalf("expected error string when codemap unavailable")
	}
	if !strings.Contains(errStr, "codemap is not installed") {
		t.Fatalf("error should mention codemap not installed: %s", errStr)
	}
}

func TestSplitFileLine(t *testing.T) {
	tests := []struct {
		input    string
		file     string
		line     int
		expected bool
	}{
		{"src/ticket.go:42", "src/ticket.go", 42, true},
		{"src/ticket.go", "src/ticket.go", 0, false},
		{"src/ticket.go:abc", "src/ticket.go:abc", 0, false},
		{"", "", 0, false},
	}
	for _, tc := range tests {
		file, line, ok := splitFileLine(tc.input)
		if ok != tc.expected {
			t.Errorf("splitFileLine(%q) ok = %v, want %v", tc.input, ok, tc.expected)
		}
		if ok && (file != tc.file || line != tc.line) {
			t.Errorf("splitFileLine(%q) = (%q, %d), want (%q, %d)", tc.input, file, line, tc.file, tc.line)
		}
	}
}

func TestBuildEvidenceRef(t *testing.T) {
	results := []evidence.SearchResult{
		{TimeSeconds: 1.0, Transcript: "I clicked the ticket"},
		{TimeSeconds: 2.0, OCR: "Ticket OPG-14010 details"},
	}
	ref := buildEvidenceRef("ticket", results)
	if !strings.Contains(ref, "1.0s") {
		t.Fatalf("expected evidence ref to contain first timestamp, got: %s", ref)
	}
	if !strings.Contains(ref, "I clicked the ticket") {
		t.Fatalf("expected evidence ref to contain first transcript, got: %s", ref)
	}
}

func TestBuildEvidenceRefEmptyResults(t *testing.T) {
	ref := buildEvidenceRef("query", nil)
	if ref != "no timestamped evidence" {
		t.Fatalf("expected empty results message, got: %s", ref)
	}
}

func mustMkdirInvestigate(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteInvestigate(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
