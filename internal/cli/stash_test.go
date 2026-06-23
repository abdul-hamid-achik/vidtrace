package cli

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestStashHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"stash", "--help"}, &stdout, &stderr, "test")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	for _, want := range []string{"vidtrace stash", "save", "list", "restore", "info", "search"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected stash help to contain %q, got %q", want, stdout.String())
		}
	}
}

func TestStashNoSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"stash"}, &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "vidtrace stash") {
		t.Fatalf("expected stash help on stderr, got %q", stderr.String())
	}
}

func TestStashUnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"stash", "bogus"}, &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown stash subcommand: bogus") {
		t.Fatalf("expected unknown subcommand error, got %q", stderr.String())
	}
}

func TestStashSaveRequiresPath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"stash", "save"}, &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage: vidtrace stash save") {
		t.Fatalf("expected usage error, got %q", stderr.String())
	}
}

func TestStashRestoreRequiresID(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"stash", "restore"}, &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage: vidtrace stash restore") {
		t.Fatalf("expected usage error, got %q", stderr.String())
	}
}

func TestStashInfoRequiresID(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"stash", "info"}, &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage: vidtrace stash info") {
		t.Fatalf("expected usage error, got %q", stderr.String())
	}
}

func TestStashSearchRequiresQuery(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"stash", "search"}, &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage: vidtrace stash search") {
		t.Fatalf("expected usage error, got %q", stderr.String())
	}
}

func TestStashSaveJSONFailureWhenFcheapMissing(t *testing.T) {
	// This test verifies the JSON failure shape when fcheap is not on PATH.
	// In the dev environment fcheap IS installed, so we test the error path
	// by pointing at a non-existent path (which fcheap will reject).
	var stdout, stderr bytes.Buffer
	code := Run([]string{"stash", "save", "/does/not/exist", "--json"}, &stdout, &stderr, "test")
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", stdout.String(), err)
	}
	if result["ok"] != false {
		t.Fatalf("expected ok=false, got %v", result["ok"])
	}
}

func TestHelpListsStashCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	Run([]string{"help"}, &stdout, &stderr, "test")
	if !strings.Contains(stdout.String(), "stash") {
		t.Fatalf("expected help to list stash command, got:\n%s", stdout.String())
	}
}

func TestNormalizeStashArgsCollectsTags(t *testing.T) {
	var tags []string
	args, err := normalizeStashArgs(
		[]string{"--tag", "bug", "--tag", "OPG-15070", "--json", "/path/to/bundle"},
		map[string]struct{}{"json": {}},
		nil,
		&tags,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 || tags[0] != "bug" || tags[1] != "OPG-15070" {
		t.Fatalf("expected tags [bug, OPG-15070], got %v", tags)
	}
	// The positional should be last.
	if args[len(args)-1] != "/path/to/bundle" {
		t.Fatalf("expected positional last, got %v", args)
	}
}

func TestNormalizeStashArgsInlineTag(t *testing.T) {
	var tags []string
	args, err := normalizeStashArgs(
		[]string{"--tag=bug", "/path"},
		map[string]struct{}{"json": {}},
		nil,
		&tags,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 1 || tags[0] != "bug" {
		t.Fatalf("expected tags [bug], got %v", tags)
	}
	if args[len(args)-1] != "/path" {
		t.Fatalf("expected positional last, got %v", args)
	}
}

func TestNormalizeStashArgsNilTags(t *testing.T) {
	args, err := normalizeStashArgs(
		[]string{"--json", "/path"},
		map[string]struct{}{"json": {}},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args[len(args)-1] != "/path" {
		t.Fatalf("expected positional last, got %v", args)
	}
}

func TestNormalizeStashArgsMissingTagValue(t *testing.T) {
	var tags []string
	_, err := normalizeStashArgs(
		[]string{"--tag"},
		nil,
		nil,
		&tags,
	)
	if err == nil || !strings.Contains(err.Error(), "missing value for flag") {
		t.Fatalf("expected missing value error, got %v", err)
	}
}

func TestTruncateStashText(t *testing.T) {
	short := "hello world"
	result := truncateStashText(short, 80)
	if result != "hello world" {
		t.Fatalf("expected short text unchanged, got %q", result)
	}

	long := strings.Repeat("word ", 30)
	result = truncateStashText(long, 20)
	if !strings.HasSuffix(result, "...") {
		t.Fatalf("expected truncated text with ellipsis, got %q", result)
	}
}

func TestStashSaveWithTagsJSON(t *testing.T) {
	// This is an integration test that actually calls fcheap if available.
	// Skip if fcheap is not installed.
	if !fcheapAvailable() {
		t.Skip("fcheap not installed")
	}

	dir := t.TempDir()
	mustWrite(t, dir+"/test.txt", "test content for stash")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"stash", "save", dir, "--name", "test-vidtrace-unit", "--tag", "unit-test", "--tag", "vidtrace", "--json"}, &stdout, &stderr, "test")
	if code != 0 {
		t.Fatalf("stash save failed: code=%d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected JSON, got %q: %v", stdout.String(), err)
	}
	if result["ok"] != true {
		t.Fatalf("expected ok=true, got %v", result)
	}
	stashID, ok := result["id"].(string)
	if !ok || stashID == "" {
		t.Fatalf("expected non-empty id, got %v", result["id"])
	}

	// Clean up: drop the stash we just created.
	_ = dropStash(stashID)
}

// fcheapAvailable checks whether the fcheap binary is on PATH.
func fcheapAvailable() bool {
	_, err := exec.LookPath("fcheap")
	return err == nil
}

// dropStash removes a stash from the vault (for test cleanup).
func dropStash(stashID string) error {
	return exec.Command("fcheap", "drop", stashID, "--force").Run()
}
