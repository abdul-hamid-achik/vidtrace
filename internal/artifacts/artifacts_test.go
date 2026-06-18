package artifacts

import (
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
