package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abdul-hamid-achik/vidtrace/internal/evidence"
)

func TestValidateToolReportsOK(t *testing.T) {
	bundleDir := writeBundle(t)
	result, report, err := validateTool(context.Background(), nil, ValidateInput{BundleDir: bundleDir})
	if err != nil {
		t.Fatalf("validateTool error: %v", err)
	}
	if result != nil && result.IsError {
		t.Fatalf("unexpected tool error: %#v", result)
	}
	if !report.OK || report.TimelineEntries != 1 {
		t.Fatalf("unexpected validation report: %#v", report)
	}
}

func TestValidateToolRequiresBundle(t *testing.T) {
	result, _, err := validateTool(context.Background(), nil, ValidateInput{})
	if err != nil {
		t.Fatalf("validateTool error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected tool error for missing bundle_dir, got %#v", result)
	}
}

func TestSearchToolKeyword(t *testing.T) {
	bundleDir := writeBundle(t)
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	if _, err := evidence.IndexBundle(evidence.IndexOptions{BundleDir: bundleDir, DBPath: dbPath}); err != nil {
		t.Fatalf("index failed: %v", err)
	}

	result, report, err := searchTool(context.Background(), nil, SearchInput{DBPath: dbPath, Query: "login fails"})
	if err != nil {
		t.Fatalf("searchTool error: %v", err)
	}
	if result != nil && result.IsError {
		t.Fatalf("unexpected tool error: %#v", result)
	}
	if !report.OK || report.Mode != "keyword" || len(report.Results) == 0 {
		t.Fatalf("unexpected search report: %#v", report)
	}
}

func TestSearchToolErrorsOnMissingDB(t *testing.T) {
	result, _, err := searchTool(context.Background(), nil, SearchInput{DBPath: "/no/such.veclite", Query: "x"})
	if err != nil {
		t.Fatalf("searchTool error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected tool error for missing db, got %#v", result)
	}
}

func TestSearchToolSemanticWithoutEmbedderErrors(t *testing.T) {
	bundleDir := writeBundle(t)
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	if _, err := evidence.IndexBundle(evidence.IndexOptions{BundleDir: bundleDir, DBPath: dbPath}); err != nil {
		t.Fatalf("index failed: %v", err)
	}
	result, _, err := searchTool(context.Background(), nil, SearchInput{DBPath: dbPath, Query: "x", Mode: "semantic"})
	if err != nil {
		t.Fatalf("searchTool error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected tool error for semantic without embedder, got %#v", result)
	}
}

func TestCompareAndAnalyzeTools(t *testing.T) {
	bundleDir := writeBundle(t)
	ticket := filepath.Join(t.TempDir(), "ticket.md")
	mustWrite(t, ticket, "Login fails after submit, retry button shown")

	_, cmp, err := compareTool(context.Background(), nil, CompareInput{BundleDir: bundleDir, TicketPath: ticket})
	if err != nil {
		t.Fatalf("compareTool error: %v", err)
	}
	if !cmp.OK || cmp.Status == "" {
		t.Fatalf("unexpected compare result: %#v", cmp)
	}

	result, out, err := analyzeTool(context.Background(), nil, AnalyzeInput{BundleDir: bundleDir, TicketPath: ticket})
	if err != nil {
		t.Fatalf("analyzeTool error: %v", err)
	}
	if out.Markdown == "" || result == nil || len(result.Content) == 0 {
		t.Fatalf("expected markdown report content, got out=%#v result=%#v", out, result)
	}
}

func TestInvestigateTool(t *testing.T) {
	bundleDir := writeBundle(t)
	result, report, err := investigateTool(context.Background(), nil, InvestigateInput{BundleDir: bundleDir, Query: "login fails"})
	if err != nil {
		t.Fatalf("investigateTool error: %v", err)
	}
	if result != nil && result.IsError {
		t.Fatalf("unexpected tool error: %#v", result)
	}
	if !report.OK || len(report.SuggestedQueries) == 0 {
		t.Fatalf("unexpected investigate report: %#v", report)
	}
}

func TestBuildEmbedder(t *testing.T) {
	if e, err := buildEmbedder("", "", ""); err != nil || e != nil {
		t.Fatalf("empty provider should yield nil embedder, got %v %v", e, err)
	}
	if _, err := buildEmbedder("ollama", "", ""); err == nil {
		t.Fatal("ollama without model should error")
	}
	if e, err := buildEmbedder("ollama", "nomic-embed-text", ""); err != nil || e == nil {
		t.Fatalf("ollama with model should build, got %v %v", e, err)
	}
	if _, err := buildEmbedder("magic", "m", ""); err == nil {
		t.Fatal("unknown provider should error")
	}
}

func TestServerRoundTripListsAndCallsTools(t *testing.T) {
	ctx := context.Background()
	bundleDir := writeBundle(t)

	server := New("test")
	clientT, serverT := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer func() { _ = ss.Close() }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer func() { _ = cs.Close() }()

	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	got := map[string]bool{}
	for _, tool := range tools.Tools {
		got[tool.Name] = true
	}
	for _, want := range ToolNames() {
		if !got[want] {
			t.Fatalf("tool %q not registered; got %v", want, got)
		}
	}

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "validate",
		Arguments: map[string]any{"bundle_dir": bundleDir},
	})
	if err != nil {
		t.Fatalf("CallTool validate: %v", err)
	}
	if res.IsError || len(res.Content) == 0 {
		t.Fatalf("unexpected validate result: %#v", res)
	}
	text, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected text content, got %T", res.Content[0])
	}
	var decoded struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal([]byte(text.Text), &decoded); err != nil || !decoded.OK {
		t.Fatalf("expected ok=true structured content, got %q (err %v)", text.Text, err)
	}

	// A tool-level error must surface as IsError, not a protocol error.
	bad, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search",
		Arguments: map[string]any{"db_path": "/no/such.veclite", "query": "x"},
	})
	if err != nil {
		t.Fatalf("CallTool search: %v", err)
	}
	if !bad.IsError {
		t.Fatalf("expected IsError for missing db, got %#v", bad)
	}
}

func writeBundle(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "frames"))
	mustMkdir(t, filepath.Join(dir, "ocr"))
	mustMkdir(t, filepath.Join(dir, "transcript"))
	mustWrite(t, filepath.Join(dir, "frames", "frame_0001.png"), "fake frame")
	mustWrite(t, filepath.Join(dir, "ocr", "frame_0001.txt"), "Login failed after submit")
	mustWrite(t, filepath.Join(dir, "ocr", "ocr_all_frames.txt"), "Login failed after submit\n")
	mustWrite(t, filepath.Join(dir, "metadata.json"), `{
  "schema_version": "1",
  "source_video": "/tmp/login-bug.mp4",
  "duration_seconds": 1,
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
      "transcript": [{"start_seconds": 0, "end_seconds": 1, "text": "the login fails when I submit"}]
    }
  ]
}`)
	return dir
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
