package pipeline

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

// TestProgressReporterConcurrentItemCalls verifies the progress reporter is
// safe for concurrent use: many goroutines calling item() simultaneously must
// not race or panic. Run with -race to catch data races.
func TestProgressReporterConcurrentItemCalls(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	r := newProgressReporter(&b, true, 7)
	r.startItems(4, "ocr", "running OCR on 100 frames (4 workers)")

	const goroutines = 50
	const iterations = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				r.item(4, "ocr", (id*iterations)+i+1, goroutines*iterations, "frame_0001.png")
			}
		}(g)
	}
	wg.Wait()
	r.finishItems()

	output := b.String()
	if !strings.Contains(output, "\r") {
		t.Fatalf("interactive mode should redraw the item line with a carriage return, got %q", output)
	}
	if !strings.HasSuffix(output, "\n") {
		t.Fatalf("finishItems should end the live line with a newline, got %q", output)
	}
}

// TestProgressReporterConcurrentSteps verifies step() and item() can be called
// from different goroutines without racing. This mirrors the real pipeline where
// the Whisper group goroutine calls step(5) while the OCR goroutine calls item().
func TestProgressReporterConcurrentSteps(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	r := newProgressReporter(&b, true, 7)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		r.startItems(4, "ocr", "running OCR")
		for i := 0; i < 20; i++ {
			r.item(4, "ocr", i+1, 20, "frame.png")
		}
		r.finishItems()
	}()

	go func() {
		defer wg.Done()
		r.step(5, "transcript", "transcribing")
	}()

	wg.Wait()
}
