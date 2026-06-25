package ffmpeg

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// ffmpegAvailable checks whether ffmpeg is on PATH.
func ffmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

// makeTestVideo generates a short synthetic video (2s, 640x360) at outPath.
func makeTestVideo(t *testing.T, outPath string) {
	t.Helper()
	if err := exec.CommandContext(context.Background(), "ffmpeg",
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc2=size=640x360:rate=1:duration=2",
		"-f", "lavfi", "-i", "sine=frequency=1000:duration=2",
		"-c:v", "libx264", "-pix_fmt", "yuv420p",
		"-c:a", "aac",
		outPath,
	).Run(); err != nil {
		t.Fatalf("ffmpeg test video generation failed: %v", err)
	}
}

func TestCutClip(t *testing.T) {
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not installed")
	}

	dir := t.TempDir()
	videoPath := filepath.Join(dir, "input.mp4")
	makeTestVideo(t, videoPath)

	outputPath := filepath.Join(dir, "clip.mp4")
	if err := CutClip(context.Background(), videoPath, 0, 1, outputPath, false); err != nil {
		t.Fatalf("CutClip failed: %v", err)
	}

	// Verify the clip file exists and is non-empty.
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("clip file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("clip file is empty")
	}
}

func TestCutClipReencode(t *testing.T) {
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not installed")
	}

	dir := t.TempDir()
	videoPath := filepath.Join(dir, "input.mp4")
	makeTestVideo(t, videoPath)

	outputPath := filepath.Join(dir, "clip_reencoded.mp4")
	if err := CutClip(context.Background(), videoPath, 0, 1, outputPath, true); err != nil {
		t.Fatalf("CutClip reencode failed: %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("reencoded clip file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("reencoded clip file is empty")
	}
}

func TestMakeGIF(t *testing.T) {
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not installed")
	}

	dir := t.TempDir()
	videoPath := filepath.Join(dir, "input.mp4")
	makeTestVideo(t, videoPath)

	outputPath := filepath.Join(dir, "clip.gif")
	if err := MakeGIF(context.Background(), videoPath, 0, 1, outputPath, 10, 480); err != nil {
		t.Fatalf("MakeGIF failed: %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("GIF file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("GIF file is empty")
	}
}

func TestConcatClips(t *testing.T) {
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not installed")
	}

	dir := t.TempDir()
	videoPath := filepath.Join(dir, "input.mp4")
	makeTestVideo(t, videoPath)

	// Cut two clips from the same source so they share codec params.
	clip1 := filepath.Join(dir, "clip1.mp4")
	clip2 := filepath.Join(dir, "clip2.mp4")
	if err := CutClip(context.Background(), videoPath, 0, 1, clip1, false); err != nil {
		t.Fatalf("CutClip clip1 failed: %v", err)
	}
	if err := CutClip(context.Background(), videoPath, 1, 2, clip2, false); err != nil {
		t.Fatalf("CutClip clip2 failed: %v", err)
	}

	// Create a concat list file.
	listPath := filepath.Join(dir, "concat_list.txt")
	content := "file '" + clip1 + "'\nfile '" + clip2 + "'\n"
	if err := os.WriteFile(listPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write concat list failed: %v", err)
	}

	outputPath := filepath.Join(dir, "stitched.mp4")
	if err := ConcatClips(context.Background(), listPath, outputPath); err != nil {
		t.Fatalf("ConcatClips failed: %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("stitched file not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("stitched file is empty")
	}
}

func TestProbe(t *testing.T) {
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not installed")
	}

	dir := t.TempDir()
	videoPath := filepath.Join(dir, "input.mp4")
	makeTestVideo(t, videoPath)

	meta, err := Probe(context.Background(), videoPath)
	if err != nil {
		t.Fatalf("Probe failed: %v", err)
	}
	if meta.DurationSeconds < 1.5 || meta.DurationSeconds > 2.5 {
		t.Fatalf("unexpected duration %.2f, expected ~2s", meta.DurationSeconds)
	}
	if meta.Width != 640 || meta.Height != 360 {
		t.Fatalf("unexpected dimensions %dx%d, expected 640x360", meta.Width, meta.Height)
	}
}
