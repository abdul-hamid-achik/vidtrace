package artifacts

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSafeBundleName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		video    string
		explicit string
		want     string
	}{
		{
			name:  "uses source basename",
			video: "/tmp/Login Bug.mov",
			want:  "Login_Bug",
		},
		{
			name:     "uses explicit name",
			video:    "/tmp/bug.mp4",
			explicit: "checkout regression",
			want:     "checkout_regression",
		},
		{
			name:     "falls back when sanitized empty",
			video:    "/tmp/bug.mp4",
			explicit: "!!!",
			want:     "video",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := SafeBundleName(tt.video, tt.explicit)
			if got != tt.want {
				t.Fatalf("SafeBundleName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBundlePath(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 18, 12, 34, 56, 0, time.UTC)
	got := BundlePath("/tmp/out", "bug", now)
	want := filepath.Join("/tmp/out", "bug_artifacts_20260618_123456")
	if got != want {
		t.Fatalf("BundlePath() = %q, want %q", got, want)
	}
}

func TestSchemaVersion(t *testing.T) {
	t.Parallel()
	if SchemaVersion != "1" {
		t.Fatalf("SchemaVersion = %q, want 1", SchemaVersion)
	}
}

func TestBundlePathUniqueNoCollision(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	parent := t.TempDir()
	got := BundlePathUnique(parent, "bug", now)
	want := filepath.Join(parent, "bug_artifacts_20260621_120000")
	if got != want {
		t.Fatalf("BundlePathUnique() with no existing dir = %q, want %q", got, want)
	}
}

func TestBundlePathUniqueAppendsSuffixOnCollision(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC)
	parent := t.TempDir()
	// Pre-create the first-choice directory to simulate a same-second re-run.
	first := BundlePath(parent, "bug", now)
	if err := os.MkdirAll(first, 0o755); err != nil {
		t.Fatal(err)
	}
	got := BundlePathUnique(parent, "bug", now)
	want := filepath.Join(parent, "bug_artifacts_20260621_120000_2")
	if got != want {
		t.Fatalf("BundlePathUnique() on collision = %q, want %q", got, want)
	}
	// Pre-create _2 as well and confirm _3 is chosen.
	if err := os.MkdirAll(got, 0o755); err != nil {
		t.Fatal(err)
	}
	got = BundlePathUnique(parent, "bug", now)
	want = filepath.Join(parent, "bug_artifacts_20260621_120000_3")
	if got != want {
		t.Fatalf("BundlePathUnique() on second collision = %q, want %q", got, want)
	}
}
