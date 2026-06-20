package timeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuild(t *testing.T) {
	t.Parallel()

	bundleDir := t.TempDir()
	mustMkdir(t, filepath.Join(bundleDir, "frames"))
	mustMkdir(t, filepath.Join(bundleDir, "ocr"))
	mustMkdir(t, filepath.Join(bundleDir, "transcript"))

	frame1 := filepath.Join(bundleDir, "frames", "frame_0001.png")
	frame2 := filepath.Join(bundleDir, "frames", "frame_0002.png")
	mustWrite(t, frame1, "fake frame")
	mustWrite(t, frame2, "fake frame")
	mustWrite(t, filepath.Join(bundleDir, "ocr", "frame_0001.txt"), "Login failed")
	mustWrite(t, filepath.Join(bundleDir, "ocr", "frame_0002.txt"), "Try again")

	transcriptPath := filepath.Join(bundleDir, "transcript", "bug.json")
	mustWrite(t, transcriptPath, `{
  "segments": [
    {"start": 0, "end": 1.2, "text": "I cannot log in"},
    {"start": 3, "end": 4, "text": "unrelated"}
  ]
}`)

	doc, err := Build(bundleDir, []string{frame1, frame2}, 1, transcriptPath)
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	if doc.SchemaVersion != "1" {
		t.Fatalf("SchemaVersion = %q, want 1", doc.SchemaVersion)
	}
	if len(doc.Entries) != 2 {
		t.Fatalf("len(Entries) = %d, want 2", len(doc.Entries))
	}

	first := doc.Entries[0]
	if first.TimeSeconds != 0 {
		t.Fatalf("first TimeSeconds = %v, want 0", first.TimeSeconds)
	}
	if first.Frame != "frames/frame_0001.png" {
		t.Fatalf("first Frame = %q", first.Frame)
	}
	if first.OCR.Text != "Login failed" {
		t.Fatalf("first OCR text = %q", first.OCR.Text)
	}
	if len(first.Transcript) != 1 || first.Transcript[0].Text != "I cannot log in" {
		t.Fatalf("unexpected first transcript: %#v", first.Transcript)
	}

	second := doc.Entries[1]
	if second.TimeSeconds != 1 {
		t.Fatalf("second TimeSeconds = %v, want 1", second.TimeSeconds)
	}
	if second.OCR.Text != "Try again" {
		t.Fatalf("second OCR text = %q", second.OCR.Text)
	}
}

func TestBuildUsesFPSForFrameTime(t *testing.T) {
	t.Parallel()

	bundleDir := t.TempDir()
	mustMkdir(t, filepath.Join(bundleDir, "frames"))
	mustMkdir(t, filepath.Join(bundleDir, "ocr"))
	frame1 := filepath.Join(bundleDir, "frames", "frame_0001.png")
	frame2 := filepath.Join(bundleDir, "frames", "frame_0002.png")
	mustWrite(t, frame1, "fake frame")
	mustWrite(t, frame2, "fake frame")

	doc, err := Build(bundleDir, []string{frame2, frame1}, 2, "")
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	if doc.Entries[0].TimeSeconds != 0 {
		t.Fatalf("first TimeSeconds = %v, want 0", doc.Entries[0].TimeSeconds)
	}
	if doc.Entries[1].TimeSeconds != 0.5 {
		t.Fatalf("second TimeSeconds = %v, want 0.5", doc.Entries[1].TimeSeconds)
	}
	if doc.Entries[0].OCR.Path != "ocr/frame_0001.txt" {
		t.Fatalf("first OCR path = %q", doc.Entries[0].OCR.Path)
	}
	if doc.Entries[0].OCR.Text != "" {
		t.Fatalf("missing OCR should be represented as empty text, got %q", doc.Entries[0].OCR.Text)
	}
}

func TestBuildRejectsInvalidFPS(t *testing.T) {
	t.Parallel()

	_, err := Build(t.TempDir(), nil, 0, "")
	if err == nil {
		t.Fatalf("expected invalid fps error")
	}
}

func TestBuildFractionalFPSTilesByActualFrameTime(t *testing.T) {
	t.Parallel()
	// fps 0.5 -> frame times 0, 2, 4.
	doc := buildDoc(t, 0.5, 3, `[
		{"start": 0.5, "end": 1.0, "text": "a"},
		{"start": 2.5, "end": 3.0, "text": "b"},
		{"start": 5.0, "end": 6.0, "text": "c"}
	]`)
	if got := []float64{doc.Entries[0].TimeSeconds, doc.Entries[1].TimeSeconds, doc.Entries[2].TimeSeconds}; got[0] != 0 || got[1] != 2 || got[2] != 4 {
		t.Fatalf("fractional fps frame times = %v, want [0 2 4]", got)
	}
	assertTexts(t, "frame1", doc.Entries[0].Transcript, "a")
	assertTexts(t, "frame2", doc.Entries[1].Transcript, "b")
	assertTexts(t, "frame3", doc.Entries[2].Transcript, "c") // trailing audio on the last frame
}

func TestBuildBoundarySegmentIsNotDoubleCounted(t *testing.T) {
	t.Parallel()
	// fps 1 -> frame times 0, 1, 2. A segment starting exactly at t=1 belongs only
	// to frame 2 (frame 1 owns the half-open [0,1)).
	doc := buildDoc(t, 1, 3, `[
		{"start": 0.5, "end": 1.0, "text": "y"},
		{"start": 1.0, "end": 1.5, "text": "x"}
	]`)
	assertTexts(t, "frame1", doc.Entries[0].Transcript, "y")
	assertTexts(t, "frame2", doc.Entries[1].Transcript, "x")
	assertTexts(t, "frame3", doc.Entries[2].Transcript)
}

func TestBuildTrailingAudioAttachesToLastFrame(t *testing.T) {
	t.Parallel()
	// fps 1 -> frame times 0, 1. A segment well after the last frame still attaches.
	doc := buildDoc(t, 1, 2, `[{"start": 5, "end": 6, "text": "late"}]`)
	assertTexts(t, "frame2", doc.Entries[1].Transcript, "late")
}

func TestBuildSpanningSegmentAttachesToEveryOverlappedFrame(t *testing.T) {
	t.Parallel()
	// fps 1 -> frame times 0, 1, 2. A long segment spans all three frames.
	doc := buildDoc(t, 1, 3, `[{"start": 0.5, "end": 2.5, "text": "long"}]`)
	for i := range doc.Entries {
		assertTexts(t, fmt.Sprintf("frame%d", i+1), doc.Entries[i].Transcript, "long")
	}
}

func TestBuildNearestFrameFallbackForUncoveredSegment(t *testing.T) {
	t.Parallel()
	// fps 1 -> frame times 0, 1, 2. A zero-length segment exactly on the t=1
	// boundary overlaps no half-open interval, so it falls back to the nearest
	// frame by midpoint (frame 2 at t=1) and is attached exactly once.
	doc := buildDoc(t, 1, 3, `[{"start": 1, "end": 1, "text": "zero"}]`)
	assertTexts(t, "frame1", doc.Entries[0].Transcript)
	assertTexts(t, "frame2", doc.Entries[1].Transcript, "zero")
	assertTexts(t, "frame3", doc.Entries[2].Transcript)
}

func TestBuildOrdersFramesByNumericIndexAcrossWidthBoundary(t *testing.T) {
	t.Parallel()
	// Past 9999 frames the %04d names are wider; lexical order would mis-sort
	// frame_10000.png before frame_9999.png. Build must order by numeric index so
	// frame times stay ascending and matching stays correct. Pass shuffled.
	doc := buildDocFrames(t, 1, []int{10000, 9998, 10001, 9999}, `[
		{"start": 9999.2, "end": 9999.4, "text": "boundary"}
	]`)
	wantTimes := []float64{9997, 9998, 9999, 10000}
	for i, want := range wantTimes {
		if doc.Entries[i].TimeSeconds != want {
			t.Fatalf("entry %d time = %v, want %v (entries must be ascending by index)", i, doc.Entries[i].TimeSeconds, want)
		}
	}
	// The segment at ~9999.3 belongs to the frame at t=9999 (index 10000), exactly once.
	assertTexts(t, "t=9997", doc.Entries[0].Transcript)
	assertTexts(t, "t=9998", doc.Entries[1].Transcript)
	assertTexts(t, "t=9999", doc.Entries[2].Transcript, "boundary")
	assertTexts(t, "t=10000", doc.Entries[3].Transcript)
}

func TestBuildFramesNotStartingAtIndexOne(t *testing.T) {
	t.Parallel()
	// Frames starting at index 3 -> times [2,3,4]. A segment before the first
	// frame overlaps no interval and falls back to the nearest (first) frame.
	doc := buildDocFrames(t, 1, []int{3, 4, 5}, `[{"start": 0, "end": 1, "text": "early"}]`)
	if doc.Entries[0].TimeSeconds != 2 || doc.Entries[2].TimeSeconds != 4 {
		t.Fatalf("frame times = %v, want first 2 / last 4", []float64{doc.Entries[0].TimeSeconds, doc.Entries[2].TimeSeconds})
	}
	assertTexts(t, "frame@2", doc.Entries[0].Transcript, "early")
}

func TestBuildNonConsecutiveFrameIndicesExtendInterval(t *testing.T) {
	t.Parallel()
	// A missing middle frame: indices 1 and 4 -> times [0,3]. A segment in the gap
	// attaches to the earlier frame, whose interval extends to the next real frame.
	doc := buildDocFrames(t, 1, []int{1, 4}, `[{"start": 1.5, "end": 2.0, "text": "gap"}]`)
	assertTexts(t, "frame@0", doc.Entries[0].Transcript, "gap")
	assertTexts(t, "frame@3", doc.Entries[1].Transcript)
}

func TestBuildNearestFrameFallbackHandlesMultipleUncoveredSegments(t *testing.T) {
	t.Parallel()
	// fps 1 -> times [0,1,2]. Two zero-length segments on boundaries overlap no
	// interval; each must land on its own nearest frame.
	doc := buildDocFrames(t, 1, []int{1, 2, 3}, `[
		{"start": 1, "end": 1, "text": "one"},
		{"start": 2, "end": 2, "text": "two"}
	]`)
	assertTexts(t, "frame@0", doc.Entries[0].Transcript)
	assertTexts(t, "frame@1", doc.Entries[1].Transcript, "one")
	assertTexts(t, "frame@2", doc.Entries[2].Transcript, "two")
}

func TestBuildEmptyTranscriptMarshalsToNull(t *testing.T) {
	t.Parallel()
	doc := buildDocFrames(t, 1, []int{1, 2}, `[{"start": 0, "end": 0.5, "text": "a"}]`)
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"transcript":null`) {
		t.Fatalf("expected an entry with \"transcript\":null, got: %s", raw)
	}
}

func TestNearestFrameBreaksTiesToEarlierFrame(t *testing.T) {
	t.Parallel()
	if got := nearestFrame([]float64{0, 2}, 1); got != 0 {
		t.Fatalf("nearestFrame tie should pick the earlier frame, got index %d", got)
	}
	if got := nearestFrame([]float64{0, 2, 4}, 3); got != 1 {
		t.Fatalf("nearestFrame(3) over [0,2,4] = %d, want 1", got)
	}
	if got := nearestFrame(nil, 1); got != -1 {
		t.Fatalf("nearestFrame over no frames = %d, want -1", got)
	}
}

func buildDocFrames(t *testing.T, fps float64, indices []int, segmentsJSON string) Document {
	t.Helper()
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "frames"))
	mustMkdir(t, filepath.Join(dir, "ocr"))
	var frames []string
	for _, i := range indices {
		p := filepath.Join(dir, "frames", fmt.Sprintf("frame_%04d.png", i))
		mustWrite(t, p, "f")
		frames = append(frames, p)
	}
	transcriptPath := ""
	if segmentsJSON != "" {
		transcriptPath = filepath.Join(dir, "transcript.json")
		mustWrite(t, transcriptPath, fmt.Sprintf(`{"segments": %s}`, segmentsJSON))
	}
	doc, err := Build(dir, frames, fps, transcriptPath)
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}
	if len(doc.Entries) != len(indices) {
		t.Fatalf("len(Entries) = %d, want %d", len(doc.Entries), len(indices))
	}
	return doc
}

func buildDoc(t *testing.T, fps float64, numFrames int, segmentsJSON string) Document {
	t.Helper()
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "frames"))
	mustMkdir(t, filepath.Join(dir, "ocr"))
	var frames []string
	for i := 1; i <= numFrames; i++ {
		p := filepath.Join(dir, "frames", fmt.Sprintf("frame_%04d.png", i))
		mustWrite(t, p, "f")
		frames = append(frames, p)
	}
	transcriptPath := ""
	if segmentsJSON != "" {
		transcriptPath = filepath.Join(dir, "transcript.json")
		mustWrite(t, transcriptPath, fmt.Sprintf(`{"segments": %s}`, segmentsJSON))
	}
	doc, err := Build(dir, frames, fps, transcriptPath)
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}
	if len(doc.Entries) != numFrames {
		t.Fatalf("len(Entries) = %d, want %d", len(doc.Entries), numFrames)
	}
	return doc
}

func assertTexts(t *testing.T, label string, segs []Segment, want ...string) {
	t.Helper()
	got := make([]string, 0, len(segs))
	for _, s := range segs {
		got = append(got, s.Text)
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("%s transcript = %v, want %v", label, got, want)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}
