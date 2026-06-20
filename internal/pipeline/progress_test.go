package pipeline

import (
	"bytes"
	"strings"
	"testing"
)

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name  string
		done  int
		total int
		width int
		want  string
	}{
		{name: "empty", done: 0, total: 4, width: 8, want: "[........]"},
		{name: "half", done: 2, total: 4, width: 8, want: "[####....]"},
		{name: "full", done: 4, total: 4, width: 8, want: "[########]"},
		{name: "clamped", done: 5, total: 4, width: 8, want: "[########]"},
		{name: "zero width", done: 1, total: 4, width: 0, want: "[]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := progressBar(tt.done, tt.total, tt.width); got != tt.want {
				t.Fatalf("progressBar() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProgressStepAndItem(t *testing.T) {
	var b bytes.Buffer

	progressStep(&b, 2, 7, "metadata", "capturing video metadata")
	progressItem(&b, 4, 7, "ocr", 3, 6, "frame_0003.png")

	output := b.String()
	for _, want := range []string{
		"[2/7] metadata",
		"[#####.............]",
		"capturing video metadata",
		"[4/7] ocr",
		"[#########.........] 3/6 frame_0003.png",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected progress output to contain %q, got %q", want, output)
		}
	}
}
