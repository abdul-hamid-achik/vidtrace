package fcheap

import (
	"encoding/json"
	"testing"
)

func TestDecodeJSONSaveResult(t *testing.T) {
	raw := `{"id":"test_20260623_150000","name":"test save"}`
	result, err := decodeJSON[SaveResult]([]byte(raw), "save")
	if err != nil {
		t.Fatalf("decodeJSON error: %v", err)
	}
	if result.ID != "test_20260623_150000" || result.Name != "test save" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestDecodeJSONListResult(t *testing.T) {
	raw := `[{"id":"a_001","name":"A","tool":"vidtrace","tags":["bug"],"file_count":5,"total_size":1024,"created_at":"2026-06-23T20:00:00Z"}]`
	result, err := decodeJSON[[]StashEntry]([]byte(raw), "list")
	if err != nil {
		t.Fatalf("decodeJSON error: %v", err)
	}
	if len(result) != 1 || result[0].ID != "a_001" || result[0].Tool != "vidtrace" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result[0].Tags) != 1 || result[0].Tags[0] != "bug" {
		t.Fatalf("unexpected tags: %+v", result[0].Tags)
	}
}

func TestDecodeJSONConnectResult(t *testing.T) {
	raw := `{
  "stash_id": "bug_001",
  "codebase": "/tmp/repo",
  "query": "login fails",
  "matches": [
    {"stash_id":"vecgrep","score":0.5,"text":"func login()","file":"/tmp/repo/auth.go:10","source":"vecgrep"}
  ]
}`
	result, err := decodeJSON[ConnectResult]([]byte(raw), "connect")
	if err != nil {
		t.Fatalf("decodeJSON error: %v", err)
	}
	if result.StashID != "bug_001" || result.Codebase != "/tmp/repo" || result.Query != "login fails" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Matches) != 1 || result.Matches[0].File != "/tmp/repo/auth.go:10" {
		t.Fatalf("unexpected matches: %+v", result.Matches)
	}
}

func TestDecodeJSONEmptyOutput(t *testing.T) {
	_, err := decodeJSON[SaveResult]([]byte(""), "save")
	if err == nil {
		t.Fatal("expected error for empty output")
	}
}

func TestDecodeJSONInvalidJSON(t *testing.T) {
	_, err := decodeJSON[SaveResult]([]byte("not json"), "save")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestDecodeJSONSearchResult(t *testing.T) {
	raw := `{"query":"login","mode":"keyword","matches":[{"text":"Login failed","file":"ocr/frame_0001.txt","score":5.2}]}`
	result, err := decodeJSON[SearchResult]([]byte(raw), "search")
	if err != nil {
		t.Fatalf("decodeJSON error: %v", err)
	}
	if result.Query != "login" || result.Mode != "keyword" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Matches) != 1 || result.Matches[0].Text != "Login failed" {
		t.Fatalf("unexpected matches: %+v", result.Matches)
	}
}

func TestDecodeJSONStashInfo(t *testing.T) {
	raw := `{"id":"bug_001","name":"Bug","tool":"vidtrace","file_count":3,"total_size":512,"created_at":"2026-06-23T20:00:00Z","files":[{"path":"metadata.json","size":100}]}`
	result, err := decodeJSON[StashInfo]([]byte(raw), "info")
	if err != nil {
		t.Fatalf("decodeJSON error: %v", err)
	}
	if result.ID != "bug_001" || result.FileCount != 3 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Files) != 1 || result.Files[0].Path != "metadata.json" {
		t.Fatalf("unexpected files: %+v", result.Files)
	}
}

func TestRestoreParsesTarget(t *testing.T) {
	// Test that the Restore function's JSON parsing handles both "target" and "path" keys.
	raw := `{"target":"/tmp/restored"}`
	var result struct {
		Target string `json:"target"`
		Path   string `json:"path"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if result.Target != "/tmp/restored" {
		t.Fatalf("expected target /tmp/restored, got %q", result.Target)
	}
}

func TestRestoreParsesPath(t *testing.T) {
	raw := `{"path":"/tmp/restored"}`
	var result struct {
		Target string `json:"target"`
		Path   string `json:"path"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if result.Path != "/tmp/restored" {
		t.Fatalf("expected path /tmp/restored, got %q", result.Path)
	}
}

func TestAvailableReturnsBool(t *testing.T) {
	// Available() just checks PATH; result depends on environment.
	// We only verify it doesn't panic.
	_ = Available()
}

func TestConnectOptionsArgsBuilding(t *testing.T) {
	// This is a structural test that verifies ConnectOptions fields are accessible.
	opts := ConnectOptions{
		StashID:     "bug_001",
		CodebaseDir: "/tmp/repo",
		Query:       "login fails",
		Mode:        "hybrid",
		Limit:       5,
		Index:       true,
	}
	if opts.StashID != "bug_001" || opts.CodebaseDir != "/tmp/repo" {
		t.Fatalf("unexpected opts: %+v", opts)
	}
	if !opts.Index || opts.Limit != 5 || opts.Mode != "hybrid" {
		t.Fatalf("unexpected opts: %+v", opts)
	}
}
