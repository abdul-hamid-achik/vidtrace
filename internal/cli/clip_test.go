package cli

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestClipHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"clip", "--help"}, &stdout, &stderr, "test")
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	for _, want := range []string{"vidtrace clip", "cut", "gif", "stitch"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("expected clip help to contain %q, got %q", want, stdout.String())
		}
	}
}

func TestClipNoSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"clip"}, &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "vidtrace clip") {
		t.Fatalf("expected clip help on stderr, got %q", stderr.String())
	}
}

func TestClipUnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"clip", "bogus"}, &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown clip subcommand: bogus") {
		t.Fatalf("expected unknown subcommand error, got %q", stderr.String())
	}
}

func TestClipCutRequiresVideoPath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"clip", "cut"}, &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage: vidtrace clip cut") {
		t.Fatalf("expected usage error, got %q", stderr.String())
	}
}

func TestClipCutRequiresRange(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"clip", "cut", "/tmp/video.mp4"}, &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "at least one --range or --label") {
		t.Fatalf("expected range required error, got %q", stderr.String())
	}
}

func TestClipCutVideoNotFoundJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"clip", "cut", "/does/not/exist.mp4", "--range", "0:01-0:02", "--json"}, &stdout, &stderr, "test")
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
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr for json failure, got %q", stderr.String())
	}
}

func TestClipCutInvalidRangeJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"clip", "cut", "/tmp/video.mp4", "--range", "abc-def", "--json"}, &stdout, &stderr, "test")
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stdout.String(), `"ok": false`) {
		t.Fatalf("expected json failure, got %q", stdout.String())
	}
}

func TestClipCutInvertedRangeJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"clip", "cut", "/tmp/video.mp4", "--range", "0:10-0:05", "--json"}, &stdout, &stderr, "test")
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stdout.String(), `"ok": false`) {
		t.Fatalf("expected json failure for inverted range, got %q", stdout.String())
	}
}

func TestClipGifRequiresVideoPath(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"clip", "gif"}, &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage: vidtrace clip gif") {
		t.Fatalf("expected usage error, got %q", stderr.String())
	}
}

func TestClipGifRequiresRange(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"clip", "gif", "/tmp/video.mp4"}, &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "at least one --range or --label") {
		t.Fatalf("expected range required error, got %q", stderr.String())
	}
}

func TestClipStitchRequiresMinTwoClips(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"clip", "stitch"}, &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage: vidtrace clip stitch") {
		t.Fatalf("expected usage error, got %q", stderr.String())
	}
}

func TestClipStitchOneClipFails(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"clip", "stitch", "/tmp/clip1.mp4"}, &stdout, &stderr, "test")
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "usage: vidtrace clip stitch") {
		t.Fatalf("expected usage error, got %q", stderr.String())
	}
}

func TestHelpListsClipCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	Run([]string{"help"}, &stdout, &stderr, "test")
	if !strings.Contains(stdout.String(), "clip") {
		t.Fatalf("expected help to list clip command, got:\n%s", stdout.String())
	}
}

func TestNormalizeClipArgsCollectsRanges(t *testing.T) {
	var ranges []string
	var labels []string
	args, err := normalizeClipArgs(
		[]string{"--range", "0:18-3:40", "--range", "3:40-4:05", "--json", "/path/to/video.mp4"},
		map[string]struct{}{"json": {}},
		map[string]struct{}{"out": {}, "name": {}},
		&ranges, &labels, nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranges) != 2 || ranges[0] != "0:18-3:40" || ranges[1] != "3:40-4:05" {
		t.Fatalf("expected 2 ranges, got %v", ranges)
	}
	if args[len(args)-1] != "/path/to/video.mp4" {
		t.Fatalf("expected positional last, got %v", args)
	}
}

func TestNormalizeClipArgsCollectsLabels(t *testing.T) {
	var ranges []string
	var labels []string
	args, err := normalizeClipArgs(
		[]string{"--label", "issue1=0:18-3:40", "--label", "issue2=3:40-4:05", "--json", "/path/to/video.mp4"},
		map[string]struct{}{"json": {}},
		map[string]struct{}{"out": {}, "name": {}},
		&ranges, &labels, nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(labels) != 2 || labels[0] != "issue1=0:18-3:40" || labels[1] != "issue2=3:40-4:05" {
		t.Fatalf("expected 2 labels, got %v", labels)
	}
	if args[len(args)-1] != "/path/to/video.mp4" {
		t.Fatalf("expected positional last, got %v", args)
	}
}

func TestNormalizeClipArgsInlineRange(t *testing.T) {
	var ranges []string
	args, err := normalizeClipArgs(
		[]string{"--range=0:18-3:40", "/path"},
		map[string]struct{}{"json": {}},
		map[string]struct{}{"out": {}, "name": {}},
		&ranges, nil, nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ranges) != 1 || ranges[0] != "0:18-3:40" {
		t.Fatalf("expected 1 inline range, got %v", ranges)
	}
	if args[len(args)-1] != "/path" {
		t.Fatalf("expected positional last, got %v", args)
	}
}

func TestNormalizeClipArgsMissingRangeValue(t *testing.T) {
	var ranges []string
	_, err := normalizeClipArgs(
		[]string{"--range"},
		nil, nil,
		&ranges, nil, nil,
	)
	if err == nil || !strings.Contains(err.Error(), "missing value for flag") {
		t.Fatalf("expected missing value error, got %v", err)
	}
}

func TestNormalizeClipArgsCollectsTags(t *testing.T) {
	var tags []string
	_, err := normalizeClipArgs(
		[]string{"--tag", "bug", "--tag", "intel", "--json", "/path"},
		map[string]struct{}{"json": {}},
		map[string]struct{}{"out": {}, "name": {}},
		nil, nil, &tags,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 || tags[0] != "bug" || tags[1] != "intel" {
		t.Fatalf("expected tags [bug, intel], got %v", tags)
	}
}

func TestBuildClipSpecsFromRanges(t *testing.T) {
	specs, err := buildClipSpecs([]string{"0:18-3:40", "3:40-4:05"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 2 {
		t.Fatalf("expected 2 specs, got %d", len(specs))
	}
	if specs[0].Label != "clip_01" || specs[0].StartSec != 18 || specs[0].EndSec != 220 {
		t.Fatalf("unexpected spec[0]: %#v", specs[0])
	}
	if specs[1].Label != "clip_02" || specs[1].StartSec != 220 || specs[1].EndSec != 245 {
		t.Fatalf("unexpected spec[1]: %#v", specs[1])
	}
}

func TestBuildClipSpecsFromLabels(t *testing.T) {
	specs, err := buildClipSpecs(nil, []string{"issue1=0:18-3:40", "issue2=3:40-4:05"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(specs) != 2 {
		t.Fatalf("expected 2 specs, got %d", len(specs))
	}
	if specs[0].Label != "issue1" || specs[0].StartSec != 18 || specs[0].EndSec != 220 {
		t.Fatalf("unexpected spec[0]: %#v", specs[0])
	}
	if specs[1].Label != "issue2" || specs[1].StartSec != 220 || specs[1].EndSec != 245 {
		t.Fatalf("unexpected spec[1]: %#v", specs[1])
	}
}

func TestBuildClipSpecsEmpty(t *testing.T) {
	_, err := buildClipSpecs(nil, nil)
	if err == nil {
		t.Fatalf("expected error for empty specs, got nil")
	}
}

func TestClipPrefixFromVideo(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/tmp/bug.mp4", "bug"},
		{"/tmp/Intel Graphite Recording.mp4", "Intel_Graphite_Recording"},
		{"/tmp/my video file.mp4", "my_video_file"},
		{"/tmp/vid@#$.mp4", "vid___"},
		{"/tmp/normal-name.mp4", "normal-name"},
	}
	for _, tc := range tests {
		got := clipPrefixFromVideo(tc.input)
		if got != tc.want {
			t.Fatalf("clipPrefixFromVideo(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// --- Integration tests requiring ffmpeg ---

func ffmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func makeClipTestVideo(t *testing.T, outPath string) {
	t.Helper()
	cmd := exec.Command("ffmpeg",
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "testsrc2=size=640x360:rate=1:duration=2",
		"-f", "lavfi", "-i", "sine=frequency=1000:duration=2",
		"-c:v", "libx264", "-pix_fmt", "yuv420p",
		"-c:a", "aac",
		outPath,
	)
	if err := cmd.Run(); err != nil {
		t.Fatalf("ffmpeg test video generation failed: %v", err)
	}
}

func TestClipCutJSON(t *testing.T) {
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not installed")
	}

	dir := t.TempDir()
	videoPath := filepath.Join(dir, "input.mp4")
	makeClipTestVideo(t, videoPath)

	outDir := filepath.Join(t.TempDir(), "clips")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"clip", "cut", videoPath,
		"--range", "0:01-0:02",
		"--out", outDir,
		"--name", "test-clip",
		"--json",
	}, &stdout, &stderr, "test")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}

	var result struct {
		OK          bool   `json:"ok"`
		SourceVideo string `json:"source_video"`
		OutputDir   string `json:"output_dir"`
		Clips       []struct {
			Label           string  `json:"label"`
			StartSeconds    float64 `json:"start_seconds"`
			EndSeconds      float64 `json:"end_seconds"`
			DurationSeconds float64 `json:"duration_seconds"`
			Path            string  `json:"path"`
		} `json:"clips"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", stdout.String(), err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true, got %v", result.OK)
	}
	if len(result.Clips) != 1 {
		t.Fatalf("expected 1 clip, got %d", len(result.Clips))
	}
	clip := result.Clips[0]
	if clip.Label != "clip_01" {
		t.Fatalf("expected label clip_01, got %q", clip.Label)
	}
	if clip.StartSeconds != 1 || clip.EndSeconds != 2 {
		t.Fatalf("unexpected range %.0f-%.0f, expected 1-2", clip.StartSeconds, clip.EndSeconds)
	}
	if clip.DurationSeconds != 1 {
		t.Fatalf("unexpected duration %.0f, expected 1", clip.DurationSeconds)
	}
	if clip.Path == "" {
		t.Fatalf("expected non-empty clip path")
	}

	// Verify the clip file exists on disk.
	clipFile := filepath.Join(result.OutputDir, clip.Path)
	if !fileExists(clipFile) {
		t.Fatalf("expected clip file at %s", clipFile)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestClipCutLabelJSON(t *testing.T) {
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not installed")
	}

	dir := t.TempDir()
	videoPath := filepath.Join(dir, "input.mp4")
	makeClipTestVideo(t, videoPath)

	outDir := filepath.Join(t.TempDir(), "clips")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"clip", "cut", videoPath,
		"--label", "issue1=0:01-0:02",
		"--out", outDir,
		"--name", "intel-session",
		"--json",
	}, &stdout, &stderr, "test")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}

	var result struct {
		OK    bool `json:"ok"`
		Clips []struct {
			Label string `json:"label"`
			Path  string `json:"path"`
		} `json:"clips"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", stdout.String(), err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true")
	}
	if len(result.Clips) != 1 || result.Clips[0].Label != "issue1" {
		t.Fatalf("expected label issue1, got %#v", result.Clips)
	}
	if result.Clips[0].Path != "issue1.mp4" {
		t.Fatalf("expected path issue1.mp4, got %q", result.Clips[0].Path)
	}
}

func TestClipGifJSON(t *testing.T) {
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not installed")
	}

	dir := t.TempDir()
	videoPath := filepath.Join(dir, "input.mp4")
	makeClipTestVideo(t, videoPath)

	outDir := filepath.Join(t.TempDir(), "gifs")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"clip", "gif", videoPath,
		"--label", "issue1=0:01-0:02",
		"--out", outDir,
		"--name", "test-gif",
		"--fps", "10",
		"--width", "480",
		"--json",
	}, &stdout, &stderr, "test")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}

	var result struct {
		OK   bool `json:"ok"`
		GIFs []struct {
			Label string `json:"label"`
			FPS   int    `json:"fps"`
			Width int    `json:"width"`
			Path  string `json:"path"`
		} `json:"gifs"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", stdout.String(), err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true")
	}
	if len(result.GIFs) != 1 {
		t.Fatalf("expected 1 gif, got %d", len(result.GIFs))
	}
	if result.GIFs[0].Label != "issue1" {
		t.Fatalf("expected label issue1, got %q", result.GIFs[0].Label)
	}
	if result.GIFs[0].FPS != 10 || result.GIFs[0].Width != 480 {
		t.Fatalf("expected fps=10 width=480, got fps=%d width=%d", result.GIFs[0].FPS, result.GIFs[0].Width)
	}
	if result.GIFs[0].Path != "issue1.gif" {
		t.Fatalf("expected path issue1.gif, got %q", result.GIFs[0].Path)
	}
}

func TestClipStitchJSON(t *testing.T) {
	if !ffmpegAvailable() {
		t.Skip("ffmpeg not installed")
	}

	dir := t.TempDir()
	videoPath := filepath.Join(dir, "input.mp4")
	makeClipTestVideo(t, videoPath)

	// Cut two clips to stitch.
	var clip1, clip2 string
	outDir1 := filepath.Join(t.TempDir(), "cuts1")
	var stdout1, stderr1 bytes.Buffer
	Run([]string{"clip", "cut", videoPath, "--range", "0:01-0:02", "--out", outDir1, "--name", "a", "--json"}, &stdout1, &stderr1, "test")
	var cut1 struct {
		Clips []struct {
			Path string `json:"path"`
		} `json:"clips"`
		OutputDir string `json:"output_dir"`
	}
	_ = json.Unmarshal(stdout1.Bytes(), &cut1)
	clip1 = filepath.Join(cut1.OutputDir, cut1.Clips[0].Path)

	outDir2 := filepath.Join(t.TempDir(), "cuts2")
	var stdout2, stderr2 bytes.Buffer
	Run([]string{"clip", "cut", videoPath, "--range", "0-0:01", "--out", outDir2, "--name", "b", "--json"}, &stdout2, &stderr2, "test")
	var cut2 struct {
		Clips []struct {
			Path string `json:"path"`
		} `json:"clips"`
		OutputDir string `json:"output_dir"`
	}
	_ = json.Unmarshal(stdout2.Bytes(), &cut2)
	clip2 = filepath.Join(cut2.OutputDir, cut2.Clips[0].Path)

	stitchOut := filepath.Join(t.TempDir(), "stitch")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"clip", "stitch", clip1, clip2, "--out", stitchOut, "--name", "summary", "--json"}, &stdout, &stderr, "test")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}

	var result struct {
		OK              bool     `json:"ok"`
		Inputs          []string `json:"inputs"`
		OutputPath      string   `json:"output_path"`
		DurationSeconds float64  `json:"duration_seconds"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", stdout.String(), err)
	}
	if !result.OK {
		t.Fatalf("expected ok=true")
	}
	if len(result.Inputs) != 2 {
		t.Fatalf("expected 2 inputs, got %d", len(result.Inputs))
	}
	if !strings.HasSuffix(result.OutputPath, "summary.mp4") {
		t.Fatalf("expected output path ending in summary.mp4, got %q", result.OutputPath)
	}
	if !fileExists(result.OutputPath) {
		t.Fatalf("expected stitched file at %s", result.OutputPath)
	}
}

func fileExists(path string) bool {
	_, err := exec.Command("test", "-f", path).Output()
	return err == nil
}
