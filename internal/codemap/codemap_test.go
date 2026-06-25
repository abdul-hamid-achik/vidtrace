package codemap

import (
	"testing"
)

func TestDecodeJSONSymbolAt(t *testing.T) {
	raw := `{
  "file": "internal/fcheap/fcheap.go",
  "line": 94,
  "symbol": "Available",
  "fqn": "fcheap.Available",
  "kind": "function",
  "start_line": 94,
  "end_line": 97,
  "resolution": "exact"
}`
	result, err := decodeJSON[SymbolAtResult]([]byte(raw), "symbol-at")
	if err != nil {
		t.Fatalf("decodeJSON error: %v", err)
	}
	if result.Symbol != "Available" || result.FQN != "fcheap.Available" {
		t.Fatalf("unexpected symbol: %+v", result)
	}
	if result.Kind != "function" || result.Resolution != "exact" {
		t.Fatalf("unexpected kind/resolution: %+v", result)
	}
	if result.StartLine != 94 || result.EndLine != 97 {
		t.Fatalf("unexpected line range: %d-%d", result.StartLine, result.EndLine)
	}
}

func TestDecodeJSONSymbolAtNone(t *testing.T) {
	raw := `{
  "file": "internal/fcheap/fcheap.go",
  "line": 1,
  "resolution": "none"
}`
	result, err := decodeJSON[SymbolAtResult]([]byte(raw), "symbol-at")
	if err != nil {
		t.Fatalf("decodeJSON error: %v", err)
	}
	if result.Resolution != "none" || result.Symbol != "" {
		t.Fatalf("expected none resolution with empty symbol: %+v", result)
	}
}

func TestDecodeJSONCallers(t *testing.T) {
	raw := `{
  "symbol": "Available",
  "project": "vidtrace",
  "found": true,
  "results": [
    {
      "symbol": "runStashSave",
      "fqn": "cli.runStashSave",
      "kind": "function",
      "file": "internal/cli/stash.go",
      "start_line": 44,
      "end_line": 98,
      "signature": "func runStashSave(args []string, stdout, stderr io.Writer) int"
    }
  ]
}`
	result, err := decodeJSON[CallersResult]([]byte(raw), "callers")
	if err != nil {
		t.Fatalf("decodeJSON error: %v", err)
	}
	if !result.Found || result.Symbol != "Available" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Results) != 1 || result.Results[0].Symbol != "runStashSave" {
		t.Fatalf("unexpected results: %+v", result.Results)
	}
	if result.Results[0].FQN != "cli.runStashSave" || result.Results[0].File != "internal/cli/stash.go" {
		t.Fatalf("unexpected caller detail: %+v", result.Results[0])
	}
}

func TestDecodeJSONCallersNotFound(t *testing.T) {
	raw := `{
  "symbol": "NonExistent",
  "project": "vidtrace",
  "found": false,
  "results": null
}`
	result, err := decodeJSON[CallersResult]([]byte(raw), "callers")
	if err != nil {
		t.Fatalf("decodeJSON error: %v", err)
	}
	if result.Found || len(result.Results) != 0 {
		t.Fatalf("expected not found with empty results: %+v", result)
	}
}

func TestDecodeJSONImpact(t *testing.T) {
	raw := `{
  "symbol": "Available",
  "project": "vidtrace",
  "found": true,
  "locations": [
    {
      "symbol": "Available",
      "fqn": "fcheap.Available",
      "kind": "function",
      "file": "internal/fcheap/fcheap.go",
      "start_line": 94,
      "end_line": 97,
      "signature": "func Available() bool"
    }
  ],
  "direct_callers": [
    {
      "symbol": "runStashSave",
      "fqn": "cli.runStashSave",
      "kind": "function",
      "file": "internal/cli/stash.go",
      "start_line": 44
    }
  ],
  "blast_radius": [
    {
      "symbol": "TestSafeBundleName",
      "fqn": "artifacts.TestSafeBundleName",
      "kind": "test",
      "file": "internal/artifacts/artifacts_test.go",
      "start_line": 10,
      "depth": 2
    }
  ],
  "tested": true,
  "untested": false
}`
	result, err := decodeJSON[ImpactResult]([]byte(raw), "impact")
	if err != nil {
		t.Fatalf("decodeJSON error: %v", err)
	}
	if !result.Found || result.Symbol != "Available" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Locations) != 1 || result.Locations[0].FQN != "fcheap.Available" {
		t.Fatalf("unexpected locations: %+v", result.Locations)
	}
	if len(result.DirectCallers) != 1 || result.DirectCallers[0].Symbol != "runStashSave" {
		t.Fatalf("unexpected direct callers: %+v", result.DirectCallers)
	}
	if len(result.BlastRadius) != 1 || result.BlastRadius[0].Depth != 2 {
		t.Fatalf("unexpected blast radius: %+v", result.BlastRadius)
	}
	if !result.Tested {
		t.Fatal("expected tested=true")
	}
}

func TestDecodeJSONSemantic(t *testing.T) {
	raw := `{
  "query": "fcheap availability check",
  "project": "vidtrace",
  "mode": "semantic",
  "hits": [
    {
      "symbol": "runConnect",
      "fqn": "investigate.runConnect",
      "kind": "function",
      "file": "internal/investigate/investigate.go",
      "start_line": 178,
      "end_line": 216,
      "score": 0.030776516,
      "signature": "func runConnect(...)"
    }
  ]
}`
	result, err := decodeJSON[SemanticResult]([]byte(raw), "semantic")
	if err != nil {
		t.Fatalf("decodeJSON error: %v", err)
	}
	if result.Query != "fcheap availability check" || result.Mode != "semantic" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Hits) != 1 || result.Hits[0].Symbol != "runConnect" {
		t.Fatalf("unexpected hits: %+v", result.Hits)
	}
	if result.Hits[0].Score < 0.03 || result.Hits[0].Score > 0.04 {
		t.Fatalf("unexpected score: %f", result.Hits[0].Score)
	}
}

func TestDecodeJSONFind(t *testing.T) {
	raw := `{
  "query": "Available",
  "project": "vidtrace",
  "mode": "name",
  "hits": [
    {
      "symbol": "Available",
      "fqn": "fcheap.Available",
      "kind": "function",
      "file": "internal/fcheap/fcheap.go",
      "start_line": 94,
      "end_line": 97,
      "score": 0,
      "signature": "func Available() bool"
    }
  ]
}`
	result, err := decodeJSON[FindResult]([]byte(raw), "find")
	if err != nil {
		t.Fatalf("decodeJSON error: %v", err)
	}
	if result.Mode != "name" || len(result.Hits) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Hits[0].FQN != "fcheap.Available" {
		t.Fatalf("unexpected hit: %+v", result.Hits[0])
	}
}

func TestDecodeJSONAnnotate(t *testing.T) {
	raw := `{
  "id": 42,
  "kind": "node",
  "matched": true,
  "source": "vidtrace",
  "target": "Available"
}`
	result, err := decodeJSON[AnnotateResult]([]byte(raw), "annotate")
	if err != nil {
		t.Fatalf("decodeJSON error: %v", err)
	}
	if result.ID != 42 || !result.Matched {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Source != "vidtrace" || result.Target != "Available" {
		t.Fatalf("unexpected source/target: %+v", result)
	}
}

func TestDecodeJSONSource(t *testing.T) {
	raw := `{
  "symbol": "Available",
  "project": "vidtrace",
  "matches": [
    {
      "symbol": "Available",
      "fqn": "fcheap.Available",
      "kind": "function",
      "file": "internal/fcheap/fcheap.go",
      "start_line": 94,
      "end_line": 97,
      "signature": "func Available() bool",
      "source": "func Available() bool { ... }"
    }
  ],
  "annotations": [
    {
      "id": 8,
      "kind": "node",
      "target": "Available",
      "source": "vidtrace",
      "note": "test annotation",
      "created_at": "2026-06-25T13:57:45Z"
    }
  ]
}`
	result, err := decodeJSON[SourceResult]([]byte(raw), "source")
	if err != nil {
		t.Fatalf("decodeJSON error: %v", err)
	}
	if len(result.Matches) != 1 || result.Matches[0].FQN != "fcheap.Available" {
		t.Fatalf("unexpected matches: %+v", result.Matches)
	}
	if len(result.Annotations) != 1 || result.Annotations[0].ID != 8 {
		t.Fatalf("unexpected annotations: %+v", result.Annotations)
	}
	if result.Annotations[0].Source != "vidtrace" || result.Annotations[0].Note != "test annotation" {
		t.Fatalf("unexpected annotation detail: %+v", result.Annotations[0])
	}
}

func TestDecodeJSONContext(t *testing.T) {
	raw := `{
  "symbol": "Available",
  "project": "vidtrace",
  "found": true,
  "definitions": [
    {
      "symbol": "Available",
      "fqn": "fcheap.Available",
      "kind": "function",
      "file": "internal/fcheap/fcheap.go",
      "start_line": 94,
      "end_line": 97
    }
  ],
  "callers": [
    {
      "symbol": "runStashSave",
      "fqn": "cli.runStashSave",
      "kind": "function",
      "file": "internal/cli/stash.go",
      "start_line": 44
    }
  ],
  "callees": [],
  "tests": [
    {
      "symbol": "TestAvailableReturnsBool",
      "fqn": "fcheap.TestAvailableReturnsBool",
      "kind": "test",
      "file": "internal/fcheap/fcheap_test.go",
      "start_line": 125
    }
  ]
}`
	result, err := decodeJSON[ContextResult]([]byte(raw), "context")
	if err != nil {
		t.Fatalf("decodeJSON error: %v", err)
	}
	if !result.Found || len(result.Definitions) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Callers) != 1 || result.Callers[0].Symbol != "runStashSave" {
		t.Fatalf("unexpected callers: %+v", result.Callers)
	}
	if len(result.Tests) != 1 || result.Tests[0].Kind != "test" {
		t.Fatalf("unexpected tests: %+v", result.Tests)
	}
}

func TestDecodeJSONNotIndexed(t *testing.T) {
	raw := `{
  "indexed": false,
  "note": "project not indexed — run 'codemap index' first",
  "project": "vidtrace"
}`
	result, err := decodeJSON[NotIndexedResult]([]byte(raw), "find")
	if err != nil {
		t.Fatalf("decodeJSON error: %v", err)
	}
	if result.Indexed || result.Project != "vidtrace" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Note == "" {
		t.Fatal("expected non-empty note")
	}
}

func TestDecodeJSONEmptyOutput(t *testing.T) {
	_, err := decodeJSON[SymbolAtResult]([]byte(""), "symbol-at")
	if err == nil {
		t.Fatal("expected error for empty output")
	}
}

func TestDecodeJSONInvalidJSON(t *testing.T) {
	_, err := decodeJSON[SymbolAtResult]([]byte("not json"), "symbol-at")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestAvailableReturnsBool(t *testing.T) {
	// Available() just checks PATH; result depends on environment.
	// We only verify it doesn't panic.
	_ = Available()
}

func TestAnnotateOptionsStructural(t *testing.T) {
	opts := AnnotateOptions{
		Symbol: "handleSubmit",
		Note:   "form submission fails on empty input",
		Source: "vidtrace",
		Data:   `{"evidence_id":"bug#5.0"}`,
	}
	if opts.Symbol != "handleSubmit" || opts.Source != "vidtrace" {
		t.Fatalf("unexpected opts: %+v", opts)
	}
	if opts.Note == "" || opts.Data == "" {
		t.Fatalf("expected non-empty note and data: %+v", opts)
	}
}
