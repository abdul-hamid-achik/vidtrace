package investigate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abdul-hamid-achik/vidtrace/internal/evidence"
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
