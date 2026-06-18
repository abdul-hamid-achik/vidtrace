package cli

import (
	"bytes"
	"strings"
	"testing"
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
