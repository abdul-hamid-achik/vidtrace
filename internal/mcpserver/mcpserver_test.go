package mcpserver

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/abdul-hamid-achik/vidtrace/internal/embed"
	"github.com/abdul-hamid-achik/vidtrace/internal/evidence"
	"github.com/abdul-hamid-achik/vidtrace/internal/fcheap"
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

func TestSearchToolSemanticViaOllamaHTTP(t *testing.T) {
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

	bundleDir := writeBundle(t)
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite")
	if _, err := evidence.IndexBundle(evidence.IndexOptions{
		BundleDir: bundleDir,
		DBPath:    dbPath,
		Embedder:  embed.NewOllama(server.URL, "nomic-embed-text"),
	}); err != nil {
		t.Fatalf("semantic index failed: %v", err)
	}

	result, report, err := searchTool(context.Background(), nil, SearchInput{
		DBPath:     dbPath,
		Query:      "login",
		Mode:       "semantic",
		Embed:      "ollama",
		EmbedModel: "nomic-embed-text",
		OllamaURL:  server.URL,
	})
	if err != nil {
		t.Fatalf("searchTool error: %v", err)
	}
	if result != nil && result.IsError {
		t.Fatalf("unexpected tool error: %#v", result)
	}
	if !report.OK || report.Mode != "semantic" || len(report.Results) == 0 {
		t.Fatalf("unexpected semantic search report: %#v", report)
	}
}

func TestToolsDoNotMutateBundle(t *testing.T) {
	ctx := context.Background()
	bundleDir := writeBundle(t)
	ticket := filepath.Join(t.TempDir(), "ticket.md")
	mustWrite(t, ticket, "login fails after submit")
	dbPath := filepath.Join(t.TempDir(), "evidence.veclite") // deliberately outside the bundle
	if _, err := evidence.IndexBundle(evidence.IndexOptions{BundleDir: bundleDir, DBPath: dbPath}); err != nil {
		t.Fatalf("index failed: %v", err)
	}

	before := snapshotDir(t, bundleDir)

	if _, _, err := validateTool(ctx, nil, ValidateInput{BundleDir: bundleDir}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := searchTool(ctx, nil, SearchInput{DBPath: dbPath, Query: "login"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := compareTool(ctx, nil, CompareInput{BundleDir: bundleDir, TicketPath: ticket}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := analyzeTool(ctx, nil, AnalyzeInput{BundleDir: bundleDir, TicketPath: ticket}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := investigateTool(ctx, nil, InvestigateInput{BundleDir: bundleDir, Query: "login"}); err != nil {
		t.Fatal(err)
	}

	after := snapshotDir(t, bundleDir)
	if len(before) != len(after) {
		t.Fatalf("bundle file set changed: before=%d after=%d files", len(before), len(after))
	}
	for path, sum := range before {
		if after[path] != sum {
			t.Fatalf("bundle file %q was added, removed, or modified by a tool", path)
		}
	}
}

func TestRoundTripExposesExpectedToolSetAndSchema(t *testing.T) {
	ctx := context.Background()
	server := New("test")
	clientT, serverT := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer func() { _ = ss.Close() }()
	client := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "0"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer func() { _ = cs.Close() }()

	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	var names []string
	var validateSchema string
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
		if tool.Name == "validate" {
			raw, _ := json.Marshal(tool.InputSchema)
			validateSchema = string(raw)
		}
	}
	sort.Strings(names)
	want := append([]string(nil), ToolNames()...)
	sort.Strings(want)
	if fmt.Sprint(names) != fmt.Sprint(want) {
		t.Fatalf("registered tools %v != ToolNames %v", names, want)
	}
	if !strings.Contains(validateSchema, `"required"`) || !strings.Contains(validateSchema, "bundle_dir") {
		t.Fatalf("validate input schema missing required bundle_dir: %s", validateSchema)
	}
}

func TestRoundTripCallsCompareAnalyzeInvestigate(t *testing.T) {
	ctx := context.Background()
	bundleDir := writeBundle(t)
	ticket := filepath.Join(t.TempDir(), "ticket.md")
	mustWrite(t, ticket, "login fails after submit")

	server := New("test")
	clientT, serverT := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer func() { _ = ss.Close() }()
	client := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "0"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer func() { _ = cs.Close() }()

	calls := []struct {
		name string
		args map[string]any
	}{
		{"compare", map[string]any{"bundle_dir": bundleDir, "ticket_path": ticket}},
		{"analyze", map[string]any{"bundle_dir": bundleDir, "ticket_path": ticket}},
		{"investigate", map[string]any{"bundle_dir": bundleDir, "query": "login fails"}},
	}
	for _, c := range calls {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: c.name, Arguments: c.args})
		if err != nil {
			t.Fatalf("CallTool %s: %v", c.name, err)
		}
		if res.IsError {
			t.Fatalf("tool %s returned error: %#v", c.name, res)
		}
		if len(res.Content) == 0 {
			t.Fatalf("tool %s returned no content", c.name)
		}
	}
}

func TestStashToolsErrorWhenFcheapUnavailable(t *testing.T) {
	if fcheap.Available() {
		t.Skip("fcheap is installed; error path not testable without mocking")
	}

	ctx := context.Background()

	// stash_list should return a tool error
	result, _, err := stashListTool(ctx, nil, StashListInput{})
	if err != nil {
		t.Fatalf("stashListTool error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected tool error when fcheap unavailable")
	}

	// stash_info should return a tool error
	result, _, err = stashInfoTool(ctx, nil, StashInfoInput{StashID: "some_id"})
	if err != nil {
		t.Fatalf("stashInfoTool error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected tool error when fcheap unavailable")
	}

	// stash_search should return a tool error
	result, _, err = stashSearchTool(ctx, nil, StashSearchInput{Query: "test"})
	if err != nil {
		t.Fatalf("stashSearchTool error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected tool error when fcheap unavailable")
	}

	// stash_connect should return a tool error
	result, _, err = stashConnectTool(ctx, nil, StashConnectInput{StashID: "some_id", Codebase: "/tmp"})
	if err != nil {
		t.Fatalf("stashConnectTool error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected tool error when fcheap unavailable")
	}
}

func TestStashToolsValidateInput(t *testing.T) {
	ctx := context.Background()

	// stash_info requires stash_id
	result, _, err := stashInfoTool(ctx, nil, StashInfoInput{})
	if err != nil {
		t.Fatalf("stashInfoTool error: %v", err)
	}
	if result == nil || !result.IsError || !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "stash_id is required") {
		t.Fatalf("expected stash_id required error")
	}

	// stash_search requires query
	result, _, err = stashSearchTool(ctx, nil, StashSearchInput{})
	if err != nil {
		t.Fatalf("stashSearchTool error: %v", err)
	}
	if result == nil || !result.IsError || !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "query is required") {
		t.Fatalf("expected query required error")
	}

	// stash_connect requires stash_id
	result, _, err = stashConnectTool(ctx, nil, StashConnectInput{Codebase: "/tmp"})
	if err != nil {
		t.Fatalf("stashConnectTool error: %v", err)
	}
	if result == nil || !result.IsError || !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "stash_id is required") {
		t.Fatalf("expected stash_id required error")
	}

	// stash_connect requires codebase
	result, _, err = stashConnectTool(ctx, nil, StashConnectInput{StashID: "some_id"})
	if err != nil {
		t.Fatalf("stashConnectTool error: %v", err)
	}
	if result == nil || !result.IsError || !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "codebase is required") {
		t.Fatalf("expected codebase required error")
	}
}

func TestInvestigateToolValidatesInput(t *testing.T) {
	ctx := context.Background()

	// Missing query
	result, _, err := investigateTool(ctx, nil, InvestigateInput{BundleDir: "/tmp"})
	if err != nil {
		t.Fatalf("investigateTool error: %v", err)
	}
	if result == nil || !result.IsError || !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "query is required") {
		t.Fatalf("expected query required error")
	}

	// Missing both bundle_dir and stash_id
	result, _, err = investigateTool(ctx, nil, InvestigateInput{Query: "test"})
	if err != nil {
		t.Fatalf("investigateTool error: %v", err)
	}
	if result == nil || !result.IsError || !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "bundle_dir or stash_id is required") {
		t.Fatalf("expected bundle_dir or stash_id required error")
	}

	// Connect without codebase_dir should be rejected
	result, _, err = investigateTool(ctx, nil, InvestigateInput{BundleDir: "/tmp", Query: "test", Connect: true})
	if err != nil {
		t.Fatalf("investigateTool error: %v", err)
	}
	if result == nil || !result.IsError || !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "connect requires codebase_dir") {
		t.Fatalf("expected connect requires codebase_dir error")
	}
}

func TestRoundTripIncludesStashTools(t *testing.T) {
	ctx := context.Background()
	server := New("test")
	clientT, serverT := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer func() { _ = ss.Close() }()

	client := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "0"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer func() { _ = cs.Close() }()

	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	names := map[string]bool{}
	for _, tool := range tools.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"stash_list", "stash_info", "stash_search", "stash_connect"} {
		if !names[want] {
			t.Fatalf("tool %q not registered", want)
		}
	}
}

func TestInvestigateToolWithConnectFlag(t *testing.T) {
	bundleDir := writeBundle(t)
	result, report, err := investigateTool(context.Background(), nil, InvestigateInput{
		BundleDir:   bundleDir,
		Query:       "login fails",
		CodebaseDir: t.TempDir(),
		Connect:     true,
	})
	if err != nil {
		t.Fatalf("investigateTool error: %v", err)
	}
	if result != nil && result.IsError {
		t.Fatalf("unexpected tool error: %#v", result)
	}
	if !report.OK {
		t.Fatalf("report should be OK")
	}
	// If fcheap is available, connect may have run. If not, ConnectError should be set.
	// Either way, the report should be valid.
	if !fcheap.Available() && report.ConnectError == "" {
		t.Fatalf("connect_error should be set when fcheap unavailable and --connect is set")
	}
}

func snapshotDir(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		content, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		out[rel] = fmt.Sprintf("%x", sha256.Sum256(content))
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", dir, err)
	}
	return out
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
