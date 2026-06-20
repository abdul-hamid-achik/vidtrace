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

func TestProgressReporterPlainMode(t *testing.T) {
	var b bytes.Buffer
	r := newProgressReporter(&b, false, 7)
	r.step(2, "metadata", "capturing video metadata")
	r.startItems(4, "ocr", "running OCR on 6 frames")
	r.item(4, "ocr", 3, 6, "frame_0003.png") // no-op in plain mode
	r.finishItems()

	output := b.String()
	for _, want := range []string{
		"[2/7] metadata",
		"[#####.............]",
		"capturing video metadata",
		"[4/7] ocr",
		"running OCR on 6 frames",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected plain progress to contain %q, got %q", want, output)
		}
	}
	if strings.Contains(output, "frame_0003.png") {
		t.Fatalf("plain mode should not print per-frame items, got %q", output)
	}
	if strings.Contains(output, "\r") {
		t.Fatalf("plain mode should not use carriage returns, got %q", output)
	}
}

func TestProgressReporterInteractiveRedraws(t *testing.T) {
	var b bytes.Buffer
	r := newProgressReporter(&b, true, 7)
	r.startItems(4, "ocr", "running OCR on 6 frames") // header carried by the live line
	r.item(4, "ocr", 3, 6, "frame_0003.png")
	r.finishItems()

	output := b.String()
	if !strings.Contains(output, "\r") {
		t.Fatalf("interactive mode should redraw the item line with a carriage return, got %q", output)
	}
	for _, want := range []string{"[4/7] ocr", "3/6", "frame_0003.png"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected interactive item line to contain %q, got %q", want, output)
		}
	}
	if !strings.HasSuffix(output, "\n") {
		t.Fatalf("finishItems should end the live line with a newline, got %q", output)
	}
}
