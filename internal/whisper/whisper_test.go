package whisper

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJSONPath(t *testing.T) {
	t.Parallel()

	got := JSONPath("/tmp/out", "/videos/bug.mp4")
	want := filepath.Join("/tmp/out", "bug.json")
	if got != want {
		t.Fatalf("JSONPath() = %q, want %q", got, want)
	}

	got = JSONPath("/tmp/out", "recording")
	want = filepath.Join("/tmp/out", "recording.json")
	if got != want {
		t.Fatalf("JSONPath() without extension = %q, want %q", got, want)
	}
}

func TestTranscriptFilesListsRegularFilesOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bug.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "bug.json"), []byte(`{"segments":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}

	files, err := TranscriptFiles(dir)
	if err != nil {
		t.Fatalf("TranscriptFiles() failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("TranscriptFiles() = %v, want 2 files", files)
	}
	if filepath.Base(files[0]) != "bug.json" || filepath.Base(files[1]) != "bug.txt" {
		t.Fatalf("TranscriptFiles() order/content = %v", files)
	}
}

func TestTranscriptFilesMissingDir(t *testing.T) {
	t.Parallel()

	_, err := TranscriptFiles(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}
