package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/abdul-hamid-achik/vidtrace/internal/artifacts"
	"github.com/abdul-hamid-achik/vidtrace/internal/timeline"
)

func TestResumeFromReusesBundleAndSkipsCompletedEmptyOCR(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	mustMkdirPipeline(t, binDir)
	logPath := filepath.Join(root, "calls.log")
	installFakeResumeTools(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("VIDTRACE_TEST_LOG", logPath)

	sourcePath := filepath.Join(root, "bug.mp4")
	mustWritePipeline(t, sourcePath, "video bytes")
	bundleDir, manifest := writeInterruptedBundle(t, root, sourcePath)
	mustWritePipeline(t, logPath, "")
	beforeEntries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}

	summary, err := Run(context.Background(), Options{
		SourceVideo:     sourcePath,
		FPS:             1,
		OCRLanguage:     "eng",
		WhisperLanguage: "en",
		WhisperModel:    "small",
		OutputParentDir: root,
		Resume:          true,
		ResumeFrom:      bundleDir,
		Concurrency:     2,
		Now:             func() time.Time { return manifest.CreatedAt.Add(time.Minute) },
	})
	if err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if summary.OutputDir != bundleDir {
		t.Fatalf("resume allocated a different bundle: got %s want %s", summary.OutputDir, bundleDir)
	}
	afterEntries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(afterEntries) != len(beforeEntries) {
		t.Fatalf("resume allocated a new top-level entry: before=%d after=%d", len(beforeEntries), len(afterEntries))
	}

	calls, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	log := string(calls)
	if strings.Contains(log, "ocr:already-empty") {
		t.Fatalf("completed empty OCR frame was rerun: %q", log)
	}
	if strings.Count(log, "ocr:needs-ocr") != 1 {
		t.Fatalf("incomplete OCR frame should run once: %q", log)
	}
	if strings.Count(log, "whisper") != 1 {
		t.Fatalf("partial Whisper output should be rebuilt once: %q", log)
	}
	if data, err := os.ReadFile(filepath.Join(bundleDir, "ocr", "frame_0001.txt")); err != nil || len(data) != 0 {
		t.Fatalf("empty OCR completion was not preserved: data=%q err=%v", data, err)
	}
	completed, err := artifacts.LoadStageManifest(filepath.Join(bundleDir, artifacts.StageManifestName))
	if err != nil {
		t.Fatal(err)
	}
	if !completed.Complete() {
		t.Fatalf("resume did not complete all stages: %#v", completed.Stages)
	}
	ocr := completed.Stages[artifacts.StageOCR]
	if len(ocr.CompletedFrameIDs) != 2 || ocr.CompletedFrameIDs[0] != "frame_0001" || ocr.CompletedFrameIDs[1] != "frame_0002" {
		t.Fatalf("unexpected OCR checkpoints: %#v", ocr)
	}
}

func TestResumeFingerprintMismatchDoesNotMutateBundle(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	sourcePath := filepath.Join(root, "bug.mp4")
	mustWritePipeline(t, sourcePath, "video bytes")
	bundleDir, _ := writeInterruptedBundle(t, root, sourcePath)
	manifestPath := filepath.Join(bundleDir, artifacts.StageManifestName)
	before, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeEntries, err := snapshotTree(bundleDir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Run(context.Background(), Options{
		SourceVideo:     sourcePath,
		FPS:             2,
		OCRLanguage:     "eng",
		WhisperLanguage: "en",
		WhisperModel:    "small",
		ResumeFrom:      bundleDir,
	})
	if err == nil || !strings.Contains(err.Error(), "fingerprint mismatch") {
		t.Fatalf("expected fingerprint mismatch, got %v", err)
	}
	after, readErr := os.ReadFile(manifestPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != string(before) {
		t.Fatal("mismatched resume changed stage_manifest.json")
	}
	afterEntries, err := snapshotTree(bundleDir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(afterEntries, "\n") != strings.Join(beforeEntries, "\n") {
		t.Fatalf("mismatched resume mutated bundle tree: before=%v after=%v", beforeEntries, afterEntries)
	}
}

func TestResumeFromLegacyBundleRefusesWithFreshExtractionAction(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	sourcePath := filepath.Join(root, "bug.mp4")
	mustWritePipeline(t, sourcePath, "video bytes")
	legacyBundle := filepath.Join(root, "legacy_bundle")
	if err := artifacts.EnsureBundleDirs(legacyBundle); err != nil {
		t.Fatal(err)
	}

	_, err := Run(context.Background(), Options{
		SourceVideo: sourcePath, FPS: 1, OCRLanguage: "eng", WhisperLanguage: "en", WhisperModel: "small",
		ResumeFrom: legacyBundle,
	})
	if err == nil || !strings.Contains(err.Error(), "without stage_manifest.json") || !strings.Contains(err.Error(), "fresh extraction without --resume or --resume-from") {
		t.Fatalf("unexpected legacy resume refusal: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(legacyBundle, artifacts.StageManifestName)); !os.IsNotExist(statErr) {
		t.Fatalf("legacy resume refusal mutated the bundle: %v", statErr)
	}
}

func TestBareResumeRejectsZeroAndAmbiguousCandidates(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	sourcePath := filepath.Join(root, "bug.mp4")
	mustWritePipeline(t, sourcePath, "video bytes")
	options := artifacts.NormalizeManifestOptions(1, []string{"eng"}, "en", "small")
	source, fingerprint, err := artifacts.FingerprintSource(sourcePath, options)
	if err != nil {
		t.Fatal(err)
	}
	now := func() time.Time { return time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC) }
	opts := Options{SourceVideo: sourcePath, FPS: 1, OCRLanguage: "eng", WhisperLanguage: "en", WhisperModel: "small", Resume: true}

	_, _, _, err = selectBundle(opts, root, source, options, fingerprint, now)
	if err == nil || !strings.Contains(err.Error(), "no compatible incomplete bundle") || !strings.Contains(err.Error(), "fresh extraction") {
		t.Fatalf("unexpected zero-candidate error: %v", err)
	}
	for _, name := range []string{"one", "two"} {
		dir := filepath.Join(root, name)
		mustMkdirPipeline(t, dir)
		manifest := artifacts.NewStageManifest(source, options, fingerprint, now())
		store := artifacts.NewManifestStore(filepath.Join(dir, artifacts.StageManifestName), now)
		if err := store.Initialize(manifest); err != nil {
			t.Fatal(err)
		}
	}
	_, _, _, err = selectBundle(opts, root, source, options, fingerprint, now)
	if err == nil || !strings.Contains(err.Error(), "multiple compatible incomplete bundles") || !strings.Contains(err.Error(), "--resume-from <bundle>") {
		t.Fatalf("unexpected ambiguous-candidate error: %v", err)
	}
}

func TestReconcileInvalidOCRInvalidatesOnlyDependents(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	sourcePath := filepath.Join(root, "bug.mp4")
	mustWritePipeline(t, sourcePath, "video bytes")
	bundleDir, manifest := writeCompleteBundle(t, root, sourcePath)
	if err := os.Remove(filepath.Join(bundleDir, "ocr", "frame_0002.txt")); err != nil {
		t.Fatal(err)
	}

	reconciled, err := reconcileManifest(bundleDir, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.Stages[artifacts.StageFrames].Status != artifacts.StageComplete {
		t.Fatal("valid frames stage was invalidated")
	}
	if reconciled.Stages[artifacts.StageTranscript].Status != artifacts.StageComplete {
		t.Fatal("independent transcript stage was invalidated")
	}
	for _, stageName := range []string{artifacts.StageOCR, artifacts.StageCombinedOCR, artifacts.StageTimeline, artifacts.StageReadme} {
		if got := reconciled.Stages[stageName].Status; got != artifacts.StagePending {
			t.Fatalf("stage %s status=%s want pending", stageName, got)
		}
	}
	ocr := reconciled.Stages[artifacts.StageOCR]
	if len(ocr.CompletedFrameIDs) != 1 || ocr.CompletedFrameIDs[0] != "frame_0001" {
		t.Fatalf("valid per-frame OCR checkpoint was not retained: %#v", ocr)
	}
}

func TestReconcileRejectsDuplicateTimelineFrames(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	sourcePath := filepath.Join(root, "bug.mp4")
	mustWritePipeline(t, sourcePath, "video bytes")
	bundleDir, manifest := writeCompleteBundle(t, root, sourcePath)
	duplicate := timeline.Document{SchemaVersion: artifacts.SchemaVersion, Entries: []timeline.Entry{
		{TimeSeconds: 0, Frame: "frames/frame_0001.png", OCR: timeline.OCR{Path: "ocr/frame_0001.txt"}},
		{TimeSeconds: 1, Frame: "frames/frame_0001.png", OCR: timeline.OCR{Path: "ocr/frame_0001.txt"}},
	}}
	if err := artifacts.WriteJSON(filepath.Join(bundleDir, "timeline.json"), duplicate); err != nil {
		t.Fatal(err)
	}

	reconciled, err := reconcileManifest(bundleDir, manifest)
	if err != nil {
		t.Fatal(err)
	}
	if got := reconciled.Stages[artifacts.StageTimeline].Status; got != artifacts.StagePending {
		t.Fatalf("duplicate timeline remained complete: %s", got)
	}
	if got := reconciled.Stages[artifacts.StageReadme].Status; got != artifacts.StagePending {
		t.Fatalf("timeline dependent remained complete: %s", got)
	}
	if got := reconciled.Stages[artifacts.StageOCR].Status; got != artifacts.StageComplete {
		t.Fatalf("independent valid OCR was invalidated: %s", got)
	}
}

func TestOCRPathsForFramesExcludeStaleOutputs(t *testing.T) {
	t.Parallel()
	bundleDir := t.TempDir()
	frames := []string{
		filepath.Join(bundleDir, "frames", "frame_0001.png"),
		filepath.Join(bundleDir, "frames", "frame_0002.png"),
	}
	paths := ocrPathsForFrames(bundleDir, frames)
	if len(paths) != 2 || strings.Contains(strings.Join(paths, "\n"), "frame_9999") {
		t.Fatalf("OCR paths were not derived from frames: %v", paths)
	}
}

func TestFreshRunCompletesManifestWithFakeTools(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	mustMkdirPipeline(t, binDir)
	installFakeResumeTools(t, binDir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	logPath := filepath.Join(root, "calls.log")
	mustWritePipeline(t, logPath, "")
	t.Setenv("VIDTRACE_TEST_LOG", logPath)
	sourcePath := filepath.Join(root, "bug.mp4")
	mustWritePipeline(t, sourcePath, "video bytes")
	outputDir := filepath.Join(root, "out")
	stamp := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)

	summary, err := Run(context.Background(), Options{
		SourceVideo: sourcePath, FPS: 1, OCRLanguage: "eng", WhisperLanguage: "en", WhisperModel: "small",
		OutputParentDir: outputDir, Concurrency: 2, Now: func() time.Time { return stamp },
	})
	if err != nil {
		t.Fatalf("fresh fake-tool run failed: %v", err)
	}
	wantBundle := artifacts.BundlePath(outputDir, "bug", stamp)
	if summary.OutputDir != wantBundle || summary.Frames != 2 || summary.OCRFiles != 2 {
		t.Fatalf("unexpected fresh summary: %#v", summary)
	}
	manifest, err := artifacts.LoadStageManifest(filepath.Join(wantBundle, artifacts.StageManifestName))
	if err != nil {
		t.Fatal(err)
	}
	if !manifest.Complete() {
		t.Fatalf("fresh run did not complete manifest: %#v", manifest.Stages)
	}
	calls, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(calls), "ocr:") != 2 || strings.Count(string(calls), "whisper") != 1 {
		t.Fatalf("unexpected external work: %q", calls)
	}
}

func TestFreshRunCreatesManifestBeforeToolFailure(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	mustMkdirPipeline(t, binDir)
	writeExecutable(t, filepath.Join(binDir, "tesseract"), "#!/bin/sh\nexit 7\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	sourcePath := filepath.Join(root, "bug.mp4")
	mustWritePipeline(t, sourcePath, "video bytes")
	outputDir := filepath.Join(root, "out")
	stamp := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)

	_, err := Run(context.Background(), Options{
		SourceVideo: sourcePath, FPS: 1, OCRLanguage: "eng", WhisperLanguage: "en", WhisperModel: "small",
		OutputParentDir: outputDir, Now: func() time.Time { return stamp },
	})
	if err == nil {
		t.Fatal("expected fake tesseract failure")
	}
	bundleDir := artifacts.BundlePath(outputDir, "bug", stamp)
	manifest, loadErr := artifacts.LoadStageManifest(filepath.Join(bundleDir, artifacts.StageManifestName))
	if loadErr != nil {
		t.Fatalf("fresh run did not initialize manifest before external tool work: %v", loadErr)
	}
	if manifest.Complete() {
		t.Fatal("failed fresh run unexpectedly has a complete manifest")
	}
}

func writeInterruptedBundle(t *testing.T, root, sourcePath string) (string, artifacts.StageManifest) {
	t.Helper()
	bundleDir, manifest := writeCompleteBundle(t, root, sourcePath)
	manifest.Stages[artifacts.StageOCR] = artifacts.ManifestStage{
		Status: artifacts.StageRunning, ArtifactCount: 1, TotalFrames: 2, CompletedFrameIDs: []string{"frame_0001"},
	}
	manifest.Stages[artifacts.StageTranscript] = artifacts.ManifestStage{Status: artifacts.StageRunning}
	for _, stageName := range []string{artifacts.StageCombinedOCR, artifacts.StageTimeline, artifacts.StageReadme} {
		manifest.Stages[stageName] = artifacts.ManifestStage{Status: artifacts.StagePending}
	}
	manifest.UpdatedAt = manifest.CreatedAt.Add(time.Second)
	store := artifacts.NewManifestStore(filepath.Join(bundleDir, artifacts.StageManifestName), nil)
	if err := store.Initialize(manifest); err != nil {
		t.Fatal(err)
	}
	mustWritePipeline(t, filepath.Join(bundleDir, "frames", "frame_0001.png"), "already-empty")
	mustWritePipeline(t, filepath.Join(bundleDir, "frames", "frame_0002.png"), "needs-ocr")
	mustWritePipeline(t, filepath.Join(bundleDir, "ocr", "frame_0001.txt"), "")
	_ = os.Remove(filepath.Join(bundleDir, "ocr", "frame_0002.txt"))
	mustWritePipeline(t, filepath.Join(bundleDir, "transcript", "bug.txt"), "partial")
	return bundleDir, manifest
}

func writeCompleteBundle(t *testing.T, root, sourcePath string) (string, artifacts.StageManifest) {
	t.Helper()
	bundleDir := filepath.Join(root, "existing_bundle")
	if err := artifacts.EnsureBundleDirs(bundleDir); err != nil {
		t.Fatal(err)
	}
	options := artifacts.NormalizeManifestOptions(1, []string{"eng"}, "en", "small")
	source, fingerprint, err := artifacts.FingerprintSource(sourcePath, options)
	if err != nil {
		t.Fatal(err)
	}
	stamp := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	manifest := artifacts.NewStageManifest(source, options, fingerprint, stamp)
	manifest.Stages[artifacts.StageMetadata] = artifacts.ManifestStage{Status: artifacts.StageComplete, ArtifactCount: 1}
	manifest.Stages[artifacts.StageFrames] = artifacts.ManifestStage{Status: artifacts.StageComplete, ArtifactCount: 2}
	manifest.Stages[artifacts.StageOCR] = artifacts.ManifestStage{Status: artifacts.StageComplete, ArtifactCount: 2, TotalFrames: 2, CompletedFrameIDs: []string{"frame_0001", "frame_0002"}}
	manifest.Stages[artifacts.StageTranscript] = artifacts.ManifestStage{Status: artifacts.StageComplete, ArtifactCount: 5}
	manifest.Stages[artifacts.StageCombinedOCR] = artifacts.ManifestStage{Status: artifacts.StageComplete, ArtifactCount: 1}
	manifest.Stages[artifacts.StageTimeline] = artifacts.ManifestStage{Status: artifacts.StageComplete, ArtifactCount: 1}
	manifest.Stages[artifacts.StageReadme] = artifacts.ManifestStage{Status: artifacts.StageComplete, ArtifactCount: 1}

	metadata := MetadataDocument{SchemaVersion: artifacts.SchemaVersion, SourceVideo: source.Path, GeneratedAt: stamp.Format(time.RFC3339), DurationSeconds: 2, ExtractFPS: 1, OCRLanguages: []string{"eng"}, WhisperLanguage: "en", WhisperModel: "small"}
	if err := artifacts.WriteJSON(filepath.Join(bundleDir, "metadata.json"), metadata); err != nil {
		t.Fatal(err)
	}
	mustWritePipeline(t, filepath.Join(bundleDir, "frames", "frame_0001.png"), "already-empty")
	mustWritePipeline(t, filepath.Join(bundleDir, "frames", "frame_0002.png"), "needs-ocr")
	mustWritePipeline(t, filepath.Join(bundleDir, "ocr", "frame_0001.txt"), "")
	mustWritePipeline(t, filepath.Join(bundleDir, "ocr", "frame_0002.txt"), "text")
	mustWritePipeline(t, filepath.Join(bundleDir, "ocr", "ocr_all_frames.txt"), "combined")
	for _, path := range expectedTranscriptPaths(filepath.Join(bundleDir, "transcript"), sourcePath) {
		value := ""
		if filepath.Ext(path) == ".json" {
			value = `{"segments":[]}`
		}
		mustWritePipeline(t, path, value)
	}
	doc := timeline.Document{SchemaVersion: artifacts.SchemaVersion, Entries: []timeline.Entry{
		{TimeSeconds: 0, Frame: "frames/frame_0001.png", OCR: timeline.OCR{Path: "ocr/frame_0001.txt"}},
		{TimeSeconds: 1, Frame: "frames/frame_0002.png", OCR: timeline.OCR{Path: "ocr/frame_0002.txt"}},
	}}
	if err := artifacts.WriteJSON(filepath.Join(bundleDir, "timeline.json"), doc); err != nil {
		t.Fatal(err)
	}
	mustWritePipeline(t, filepath.Join(bundleDir, "README.txt"), "readme")
	store := artifacts.NewManifestStore(filepath.Join(bundleDir, artifacts.StageManifestName), nil)
	if err := store.Initialize(manifest); err != nil {
		t.Fatal(err)
	}
	return bundleDir, manifest
}

func installFakeResumeTools(t *testing.T, binDir string) {
	t.Helper()
	writeExecutable(t, filepath.Join(binDir, "ffprobe"), `#!/bin/sh
printf '{"streams":[{"codec_name":"h264","codec_type":"video","width":1280,"height":720,"avg_frame_rate":"30/1"},{"codec_name":"aac","codec_type":"audio"}],"format":{"duration":"2"}}'
`)
	writeExecutable(t, filepath.Join(binDir, "ffmpeg"), `#!/bin/sh
for arg in "$@"; do
  pattern=$arg
done
dir=$(dirname "$pattern")
printf 'frame-one' > "$dir/frame_0001.png"
printf 'frame-two' > "$dir/frame_0002.png"
`)
	writeExecutable(t, filepath.Join(binDir, "tesseract"), `#!/bin/sh
if [ "$1" = "--list-langs" ]; then
  printf 'List of available languages (1):\neng\n'
  exit 0
fi
input=$(cat)
printf 'ocr:%s\n' "$input" >> "$VIDTRACE_TEST_LOG"
: > "$2.txt"
`)
	writeExecutable(t, filepath.Join(binDir, "whisper"), `#!/bin/sh
printf 'whisper\n' >> "$VIDTRACE_TEST_LOG"
source=$1
shift
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--output_dir" ]; then
    out=$2
    shift 2
  else
    shift
  fi
done
base=$(basename "$source")
base=${base%.*}
printf '{"segments":[]}' > "$out/$base.json"
: > "$out/$base.srt"
: > "$out/$base.tsv"
: > "$out/$base.txt"
: > "$out/$base.vtt"
`)
}

func snapshotTree(root string) ([]string, error) {
	var result []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		result = append(result, rel)
		return nil
	})
	return result, err
}

func writeExecutable(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustMkdirPipeline(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWritePipeline(t *testing.T, path, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestExpectedTranscriptSetRequiresEveryFormat(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	source := filepath.Join(dir, "clip.mov")
	paths := expectedTranscriptPaths(dir, source)
	if len(paths) != 5 {
		t.Fatalf("expected five Whisper all-format outputs, got %v", paths)
	}
	for _, path := range paths[:len(paths)-1] {
		mustWritePipeline(t, path, "")
	}
	if allRegularFiles(paths) {
		t.Fatal("partial transcript output set was accepted as complete")
	}
}

func TestManifestJSONRemainsValidAfterRecovery(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	source := filepath.Join(dir, "video.mp4")
	mustWritePipeline(t, source, "x")
	options := artifacts.NormalizeManifestOptions(1, []string{"eng"}, "en", "small")
	identity, fingerprint, err := artifacts.FingerprintSource(source, options)
	if err != nil {
		t.Fatal(err)
	}
	manifest := artifacts.NewStageManifest(identity, options, fingerprint, time.Now())
	data, err := json.Marshal(manifest)
	if err != nil || !json.Valid(data) {
		t.Fatalf("manifest JSON invalid: %v", err)
	}
}

func TestVerifySourceStableInvalidatesEveryCheckpoint(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "source.mp4")
	mustWritePipeline(t, sourcePath, "original bytes")
	options := artifacts.NormalizeManifestOptions(1, []string{"eng"}, "en", "small")
	source, fingerprint, err := artifacts.FingerprintSource(sourcePath, options)
	if err != nil {
		t.Fatal(err)
	}
	bundleDir := filepath.Join(root, "bundle")
	mustMkdirPipeline(t, bundleDir)
	manifest := artifacts.NewStageManifest(source, options, fingerprint, time.Now())
	for _, stageName := range artifacts.StageOrder {
		manifest.Stages[stageName] = artifacts.ManifestStage{Status: artifacts.StageComplete, ArtifactCount: 1}
	}
	manifest.Stages[artifacts.StageOCR] = artifacts.ManifestStage{
		Status: artifacts.StageComplete, ArtifactCount: 1, TotalFrames: 1, CompletedFrameIDs: []string{"frame_0001"},
	}
	store := artifacts.NewManifestStore(filepath.Join(bundleDir, artifacts.StageManifestName), nil)
	if err := store.Initialize(manifest); err != nil {
		t.Fatal(err)
	}
	mustWritePipeline(t, sourcePath, "different bytes")
	if err := verifySourceStable(store, source, options, fingerprint); err == nil || !strings.Contains(err.Error(), "changed during extraction") {
		t.Fatalf("source mutation error = %v", err)
	}
	updated, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, stageName := range artifacts.StageOrder {
		if updated.Stages[stageName].Status == artifacts.StageComplete {
			t.Fatalf("source mutation left %s complete: %+v", stageName, updated.Stages[stageName])
		}
	}
	if updated.Stages[artifacts.StageMetadata].Status != artifacts.StageFailed {
		t.Fatalf("metadata stage = %+v, want failed source identity checkpoint", updated.Stages[artifacts.StageMetadata])
	}
}
