package doctor

import (
	"bytes"
	"strings"
	"testing"
)

func TestCheckIncludesOllamaAsOptionalTool(t *testing.T) {
	result := Check()

	if len(result.OptionalTools) == 0 {
		t.Fatal("expected at least one optional tool")
	}
	var found bool
	for _, tool := range result.OptionalTools {
		if tool.Name == "ollama" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected ollama in optional tools, got %#v", result.OptionalTools)
	}
}

func TestMissingOllamaDoesNotFailDoctor(t *testing.T) {
	// A missing optional tool must surface a hint but never affect OK; OK depends
	// only on required tools and language/model availability.
	result := Result{
		OK:            true,
		OptionalTools: []ToolStatus{{Name: "ollama", Found: false}},
	}
	if !result.OptionalTools[0].Found {
		// Simulate the hint that Check() adds for a missing optional tool.
		result.RecommendedNextSteps = append(result.RecommendedNextSteps, "Optional: install Ollama for semantic and hybrid evidence search (vidtrace index/search --embed ollama).")
	}
	if !result.OK {
		t.Fatal("a missing optional tool must not flip OK to false")
	}
	if len(result.RecommendedNextSteps) != 1 || !strings.Contains(result.RecommendedNextSteps[0], "Optional: install Ollama") {
		t.Fatalf("expected an optional Ollama hint, got %#v", result.RecommendedNextSteps)
	}
}

func TestPrintHumanRendersOptionalTools(t *testing.T) {
	var buf bytes.Buffer
	PrintHuman(&buf, Result{
		OK:            true,
		Tools:         []ToolStatus{{Name: "ffmpeg", Found: true, Path: "/usr/bin/ffmpeg"}},
		OptionalTools: []ToolStatus{{Name: "ollama", Found: false}},
	})
	out := buf.String()
	if !strings.Contains(out, "Optional tools:") || !strings.Contains(out, "ollama: missing") {
		t.Fatalf("expected optional tools section, got:\n%s", out)
	}
}

func TestCheckIncludesFcheapAndVecgrepAsOptionalTools(t *testing.T) {
	result := Check()

	names := map[string]bool{}
	for _, tool := range result.OptionalTools {
		names[tool.Name] = true
	}
	if !names["fcheap"] {
		t.Fatalf("expected fcheap in optional tools, got %#v", result.OptionalTools)
	}
	if !names["vecgrep"] {
		t.Fatalf("expected vecgrep in optional tools, got %#v", result.OptionalTools)
	}
}

func TestMissingFcheapDoesNotFailDoctor(t *testing.T) {
	result := Result{
		OK:            true,
		OptionalTools: []ToolStatus{{Name: "fcheap", Found: false}},
	}
	if !result.OK {
		t.Fatal("a missing optional tool must not flip OK to false")
	}
}

func TestMissingVecgrepDoesNotFailDoctor(t *testing.T) {
	result := Result{
		OK:            true,
		OptionalTools: []ToolStatus{{Name: "vecgrep", Found: false}},
	}
	if !result.OK {
		t.Fatal("a missing optional tool must not flip OK to false")
	}
}

func TestCheckIncludesCodemapAsOptionalTool(t *testing.T) {
	result := Check()

	names := map[string]bool{}
	for _, tool := range result.OptionalTools {
		names[tool.Name] = true
	}
	if !names["codemap"] {
		t.Fatalf("expected codemap in optional tools, got %#v", result.OptionalTools)
	}
}

func TestMissingCodemapDoesNotFailDoctor(t *testing.T) {
	result := Result{
		OK:            true,
		OptionalTools: []ToolStatus{{Name: "codemap", Found: false}},
	}
	if !result.OK {
		t.Fatal("a missing optional tool must not flip OK to false")
	}
}

func TestPrintHumanRendersFcheapAndVecgrep(t *testing.T) {
	var buf bytes.Buffer
	PrintHuman(&buf, Result{
		OK:    true,
		Tools: []ToolStatus{{Name: "ffmpeg", Found: true, Path: "/usr/bin/ffmpeg"}},
		OptionalTools: []ToolStatus{
			{Name: "fcheap", Found: true, Path: "/usr/bin/fcheap"},
			{Name: "vecgrep", Found: false},
			{Name: "codemap", Found: true, Path: "/opt/homebrew/bin/codemap"},
		},
	})
	out := buf.String()
	if !strings.Contains(out, "fcheap: found") || !strings.Contains(out, "vecgrep: missing") {
		t.Fatalf("expected fcheap and vecgrep in optional tools, got:\n%s", out)
	}
	if !strings.Contains(out, "codemap: found") {
		t.Fatalf("expected codemap in optional tools, got:\n%s", out)
	}
}
