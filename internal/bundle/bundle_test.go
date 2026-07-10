package bundle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/abdul-hamid-achik/vidtrace/internal/artifacts"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	dir := writeFixture(t)

	doc, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if doc.Metadata.SchemaVersion != "1" {
		t.Fatalf("metadata schema = %q", doc.Metadata.SchemaVersion)
	}
	if len(doc.Timeline.Entries) != 2 {
		t.Fatalf("timeline entries = %d, want 2", len(doc.Timeline.Entries))
	}
	if !strings.Contains(doc.SearchableText(), "Login failed") {
		t.Fatalf("expected searchable text to include OCR, got %q", doc.SearchableText())
	}
	if !strings.Contains(doc.TranscriptText(), "I cannot log in") {
		t.Fatalf("expected transcript text, got %q", doc.TranscriptText())
	}
}

func TestValidateValidBundle(t *testing.T) {
	t.Parallel()

	dir := writeFixture(t)
	mustWrite(t, filepath.Join(dir, "frames", "frame_0001.png"), "fake frame")
	mustWrite(t, filepath.Join(dir, "frames", "frame_0002.png"), "fake frame")
	mustWrite(t, filepath.Join(dir, "ocr", "frame_0001.txt"), "Login failed")
	mustWrite(t, filepath.Join(dir, "ocr", "frame_0002.txt"), "")
	mustWrite(t, filepath.Join(dir, "timeline.json"), `{
  "schema_version": "1",
  "entries": [
    {
      "time_seconds": 0,
      "frame": "frames/frame_0001.png",
      "ocr": {"path": "ocr/frame_0001.txt", "text": "Login failed"},
      "transcript": [{"start_seconds": 0, "end_seconds": 1, "text": "I cannot log in"}]
    },
    {
      "time_seconds": 1,
      "frame": "frames/frame_0002.png",
      "ocr": {"path": "ocr/frame_0002.txt", "text": ""},
      "transcript": []
    }
  ]
}`)

	report := Validate(dir)

	if !report.OK {
		t.Fatalf("expected valid bundle, got %#v", report)
	}
	if report.TimelineEntries != 2 {
		t.Fatalf("TimelineEntries = %d, want 2", report.TimelineEntries)
	}
	if report.EmptyOCREntries != 1 {
		t.Fatalf("EmptyOCREntries = %d, want 1", report.EmptyOCREntries)
	}
	if !hasCheck(report, "timeline_ocr_paths", true) {
		t.Fatalf("expected passing timeline_ocr_paths check, got %#v", report.Checks)
	}
}

func TestValidateInvalidBundle(t *testing.T) {
	t.Parallel()

	dir := writeFixture(t)
	mustWrite(t, filepath.Join(dir, "frames", "frame_0001.png"), "fake frame")

	report := Validate(dir)

	if report.OK {
		t.Fatalf("expected invalid bundle, got %#v", report)
	}
	if !hasCheck(report, "timeline_frames", false) {
		t.Fatalf("expected failing timeline_frames check, got %#v", report.Checks)
	}
	if !hasCheck(report, "timeline_ocr_paths", false) {
		t.Fatalf("expected failing timeline_ocr_paths check, got %#v", report.Checks)
	}
}

func TestValidateWarnsOnEmptyTranscriptWithWhisperModel(t *testing.T) {
	t.Parallel()

	dir := writeFixture(t)
	mustWrite(t, filepath.Join(dir, "frames", "frame_0001.png"), "fake frame")
	mustWrite(t, filepath.Join(dir, "frames", "frame_0002.png"), "fake frame")
	mustWrite(t, filepath.Join(dir, "ocr", "frame_0001.txt"), "Login failed")
	mustWrite(t, filepath.Join(dir, "ocr", "frame_0002.txt"), "")
	mustWrite(t, filepath.Join(dir, "timeline.json"), `{
  "schema_version": "1",
  "entries": [
    {
      "time_seconds": 0,
      "frame": "frames/frame_0001.png",
      "ocr": {"path": "ocr/frame_0001.txt", "text": "Login failed"},
      "transcript": []
    },
    {
      "time_seconds": 1,
      "frame": "frames/frame_0002.png",
      "ocr": {"path": "ocr/frame_0002.txt", "text": ""},
      "transcript": []
    }
  ]
}`)
	// transcript/ dir exists (created by writeFixture) but is empty.

	report := Validate(dir)
	if !report.OK {
		t.Fatalf("bundle with empty transcript should still be valid (silent video), got %#v", report)
	}
	if !hasWarning(report, "transcript/ is empty") {
		t.Fatalf("expected a transcript-empty warning, got %#v", report.Warnings)
	}
}

func TestValidateWarnsOnFrameOCRCountDrift(t *testing.T) {
	t.Parallel()

	dir := writeFixture(t)
	mustWrite(t, filepath.Join(dir, "frames", "frame_0001.png"), "fake frame")
	mustWrite(t, filepath.Join(dir, "frames", "frame_0002.png"), "fake frame")
	mustWrite(t, filepath.Join(dir, "ocr", "frame_0001.txt"), "Login failed")
	mustWrite(t, filepath.Join(dir, "timeline.json"), `{
  "schema_version": "1",
  "entries": [
    {
      "time_seconds": 0,
      "frame": "frames/frame_0001.png",
      "ocr": {"path": "ocr/frame_0001.txt", "text": "Login failed"},
      "transcript": []
    },
    {
      "time_seconds": 1,
      "frame": "frames/frame_0002.png",
      "ocr": {"path": "ocr/frame_0002.txt", "text": ""},
      "transcript": []
    }
  ]
}`)
	// 2 frames but only 1 OCR file (frame_0002.txt is missing on disk but
	// referenced in timeline; timeline_ocr_paths will fail, and the count drift
	// warning should also fire).

	report := Validate(dir)
	if !hasWarning(report, "frame count (2) differs from OCR frame txt count (1)") {
		t.Fatalf("expected a frame/OCR count drift warning, got %#v", report.Warnings)
	}
}

func TestValidateEmitsStructuredDeduplicatedRepairArgv(t *testing.T) {
	t.Parallel()

	dir := writeFixture(t)
	mustWrite(t, filepath.Join(dir, "frames", "frame_0001.png"), "fake frame")
	mustWrite(t, filepath.Join(dir, "frames", "frame_0002.png"), "fake frame")
	mustWrite(t, filepath.Join(dir, "ocr", "frame_0001.txt"), "")
	mustWrite(t, filepath.Join(dir, "ocr", "frame_0002.txt"), "Retry")
	sourcePath := filepath.Join(dir, "source video.mp4")
	mustWrite(t, sourcePath, "video bytes")
	alignManifestFixture(t, dir, sourcePath)
	for _, extension := range []string{".json", ".srt", ".tsv", ".txt", ".vtt"} {
		value := ""
		if extension == ".json" {
			value = `{"segments":[]}`
		}
		mustWrite(t, filepath.Join(dir, "transcript", "source video"+extension), value)
	}

	options := artifacts.NormalizeManifestOptions(1, []string{"eng"}, "en", "small")
	source, fingerprint, err := artifacts.FingerprintSource(sourcePath, options)
	if err != nil {
		t.Fatal(err)
	}
	manifest := artifacts.NewStageManifest(source, options, fingerprint, time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC))
	for _, stageName := range artifacts.StageOrder {
		manifest.Stages[stageName] = artifacts.ManifestStage{Status: artifacts.StageComplete, ArtifactCount: 1}
	}
	manifest.Stages[artifacts.StageFrames] = artifacts.ManifestStage{Status: artifacts.StageComplete, ArtifactCount: 2}
	manifest.Stages[artifacts.StageTranscript] = artifacts.ManifestStage{Status: artifacts.StageComplete, ArtifactCount: 5}
	manifest.Stages[artifacts.StageOCR] = artifacts.ManifestStage{
		Status: artifacts.StageFailed, LastError: "interrupted", ArtifactCount: 1, TotalFrames: 2, CompletedFrameIDs: []string{"frame_0001"},
	}
	store := artifacts.NewManifestStore(filepath.Join(dir, artifacts.StageManifestName), nil)
	if err := store.Initialize(manifest); err != nil {
		t.Fatal(err)
	}

	report := Validate(dir)
	if report.OK {
		t.Fatalf("incomplete manifest should fail validation: %#v", report)
	}
	if len(report.Repairs) != 1 {
		t.Fatalf("repair actions were not deduplicated by stage: %#v", report.Repairs)
	}
	repair := report.Repairs[0]
	if repair.Stage != artifacts.StageOCR || repair.Command != "vidtrace" {
		t.Fatalf("unexpected repair action: %#v", repair)
	}
	wantArgs := []string{
		"extract", "--resume-from", dir, "--fps", "1", "--ocr-lang", "eng",
		"--whisper-lang", "en", "--model", "small", sourcePath,
	}
	if strings.Join(repair.Args, "\x00") != strings.Join(wantArgs, "\x00") {
		t.Fatalf("repair argv mismatch: got %q want %q", repair.Args, wantArgs)
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"repairs":[{"stage":"ocr"`) || strings.Contains(string(data), `"command":"vidtrace extract`) {
		t.Fatalf("repair JSON is not structured argv: %s", data)
	}
}

func TestValidateRepairsMissingTranscriptFormat(t *testing.T) {
	t.Parallel()

	dir := writeFixture(t)
	mustWrite(t, filepath.Join(dir, "frames", "frame_0001.png"), "fake frame")
	mustWrite(t, filepath.Join(dir, "frames", "frame_0002.png"), "fake frame")
	mustWrite(t, filepath.Join(dir, "ocr", "frame_0001.txt"), "")
	mustWrite(t, filepath.Join(dir, "ocr", "frame_0002.txt"), "Retry")
	sourcePath := filepath.Join(dir, "source.mp4")
	mustWrite(t, sourcePath, "video bytes")
	alignManifestFixture(t, dir, sourcePath)
	for _, extension := range []string{".json", ".tsv", ".txt", ".vtt"} {
		value := ""
		if extension == ".json" {
			value = `{"segments":[]}`
		}
		mustWrite(t, filepath.Join(dir, "transcript", "source"+extension), value)
	}

	options := artifacts.NormalizeManifestOptions(1, []string{"eng"}, "en", "small")
	source, fingerprint, err := artifacts.FingerprintSource(sourcePath, options)
	if err != nil {
		t.Fatal(err)
	}
	manifest := artifacts.NewStageManifest(source, options, fingerprint, time.Now())
	for _, stageName := range artifacts.StageOrder {
		manifest.Stages[stageName] = artifacts.ManifestStage{Status: artifacts.StageComplete, ArtifactCount: 1}
	}
	manifest.Stages[artifacts.StageFrames] = artifacts.ManifestStage{Status: artifacts.StageComplete, ArtifactCount: 2}
	manifest.Stages[artifacts.StageOCR] = artifacts.ManifestStage{
		Status: artifacts.StageComplete, ArtifactCount: 2, TotalFrames: 2, CompletedFrameIDs: []string{"frame_0001", "frame_0002"},
	}
	manifest.Stages[artifacts.StageTranscript] = artifacts.ManifestStage{Status: artifacts.StageComplete, ArtifactCount: 5}
	store := artifacts.NewManifestStore(filepath.Join(dir, artifacts.StageManifestName), nil)
	if err := store.Initialize(manifest); err != nil {
		t.Fatal(err)
	}

	report := Validate(dir)
	if report.OK || len(report.Repairs) != 1 || report.Repairs[0].Stage != artifacts.StageTranscript {
		t.Fatalf("missing transcript format did not produce one transcript repair: %#v", report)
	}
	if !hasCheck(report, "manifest_transcript", false) {
		t.Fatalf("missing transcript format did not fail manifest_transcript: %#v", report.Checks)
	}

	mustWrite(t, filepath.Join(dir, "transcript", "source.srt"), "")
	timelinePath := filepath.Join(dir, "timeline.json")
	timelineData, err := os.ReadFile(timelinePath)
	if err != nil {
		t.Fatal(err)
	}
	corruptTimeline := strings.Replace(string(timelineData), "frames/frame_0001.png", "frames/stale.png", 1)
	mustWrite(t, timelinePath, corruptTimeline)
	report = Validate(dir)
	if report.OK || len(report.Repairs) != 1 || report.Repairs[0].Stage != artifacts.StageTimeline {
		t.Fatalf("corrupt timeline reference did not produce one timeline repair: %#v", report)
	}
}

func TestValidateManifestChecksEveryDerivedArtifactStage(t *testing.T) {
	tests := []struct {
		name   string
		stage  string
		mutate func(*testing.T, string, string)
	}{
		{
			name: "metadata identity", stage: artifacts.StageMetadata,
			mutate: func(t *testing.T, dir, sourcePath string) {
				data, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
				if err != nil {
					t.Fatal(err)
				}
				mustWrite(t, filepath.Join(dir, "metadata.json"), strings.Replace(string(data), sourcePath, sourcePath+".other", 1))
			},
		},
		{
			name: "combined OCR content", stage: artifacts.StageCombinedOCR,
			mutate: func(t *testing.T, dir, _ string) {
				mustWrite(t, filepath.Join(dir, "ocr", "ocr_all_frames.txt"), "")
			},
		},
		{
			name: "timeline completeness", stage: artifacts.StageTimeline,
			mutate: func(t *testing.T, dir, _ string) {
				path := filepath.Join(dir, "timeline.json")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				value := strings.ReplaceAll(string(data), "frame_0002", "frame_0001")
				mustWrite(t, path, value)
			},
		},
		{
			name: "README presence", stage: artifacts.StageReadme,
			mutate: func(t *testing.T, dir, _ string) {
				if err := os.Remove(filepath.Join(dir, "README.txt")); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir, sourcePath := writeCompleteManifestFixture(t)
			if report := Validate(dir); !report.OK {
				t.Fatalf("complete manifest fixture is invalid before mutation: %#v", report)
			}
			test.mutate(t, dir, sourcePath)
			report := Validate(dir)
			if report.OK || len(report.Repairs) != 1 || report.Repairs[0].Stage != test.stage {
				t.Fatalf("%s mutation repair = %#v", test.stage, report)
			}
		})
	}
}

func writeCompleteManifestFixture(t *testing.T) (string, string) {
	t.Helper()
	dir := writeFixture(t)
	mustWrite(t, filepath.Join(dir, "frames", "frame_0001.png"), "fake frame")
	mustWrite(t, filepath.Join(dir, "frames", "frame_0002.png"), "fake frame")
	mustWrite(t, filepath.Join(dir, "ocr", "frame_0001.txt"), "")
	mustWrite(t, filepath.Join(dir, "ocr", "frame_0002.txt"), "Retry")
	sourcePath := filepath.Join(dir, "source.mp4")
	mustWrite(t, sourcePath, "video bytes")
	alignManifestFixture(t, dir, sourcePath)
	for _, extension := range []string{".json", ".srt", ".tsv", ".txt", ".vtt"} {
		value := ""
		if extension == ".json" {
			value = `{"segments":[]}`
		}
		mustWrite(t, filepath.Join(dir, "transcript", "source"+extension), value)
	}
	options := artifacts.NormalizeManifestOptions(1, []string{"eng"}, "en", "small")
	source, fingerprint, err := artifacts.FingerprintSource(sourcePath, options)
	if err != nil {
		t.Fatal(err)
	}
	manifest := artifacts.NewStageManifest(source, options, fingerprint, time.Now())
	for _, stageName := range artifacts.StageOrder {
		manifest.Stages[stageName] = artifacts.ManifestStage{Status: artifacts.StageComplete, ArtifactCount: 1}
	}
	manifest.Stages[artifacts.StageFrames] = artifacts.ManifestStage{Status: artifacts.StageComplete, ArtifactCount: 2}
	manifest.Stages[artifacts.StageOCR] = artifacts.ManifestStage{
		Status: artifacts.StageComplete, ArtifactCount: 2, TotalFrames: 2, CompletedFrameIDs: []string{"frame_0001", "frame_0002"},
	}
	manifest.Stages[artifacts.StageTranscript] = artifacts.ManifestStage{Status: artifacts.StageComplete, ArtifactCount: 5}
	store := artifacts.NewManifestStore(filepath.Join(dir, artifacts.StageManifestName), nil)
	if err := store.Initialize(manifest); err != nil {
		t.Fatal(err)
	}
	return dir, sourcePath
}

func alignManifestFixture(t *testing.T, dir, sourcePath string) {
	t.Helper()
	metadata := Metadata{
		SchemaVersion: "1", SourceVideo: sourcePath, DurationSeconds: 2, ExtractFPS: 1,
		OCRLanguages: []string{"eng"}, WhisperLanguage: "en", WhisperModel: "small",
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(dir, "metadata.json"), string(data))
	mustWrite(t, filepath.Join(dir, "README.txt"), "complete bundle\n")
}

func writeFixture(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "ocr"))
	mustMkdir(t, filepath.Join(dir, "frames"))
	mustMkdir(t, filepath.Join(dir, "transcript"))
	mustWrite(t, filepath.Join(dir, "metadata.json"), `{
  "schema_version": "1",
  "source_video": "/tmp/bug.mp4",
  "duration_seconds": 2,
  "extract_fps": 1,
  "ocr_languages": ["eng"],
  "whisper_language": "en",
  "whisper_model": "small"
}`)
	mustWrite(t, filepath.Join(dir, "timeline.json"), `{
  "schema_version": "1",
  "entries": [
    {
      "time_seconds": 0,
      "frame": "frames/frame_0001.png",
      "ocr": {"path": "ocr/frame_0001.txt", "text": "Login failed"},
      "transcript": [{"start_seconds": 0, "end_seconds": 1, "text": "I cannot log in"}]
    },
    {
      "time_seconds": 1,
      "frame": "frames/frame_0002.png",
      "ocr": {"path": "ocr/frame_0002.txt", "text": "Retry button"},
      "transcript": []
    }
  ]
}`)
	mustWrite(t, filepath.Join(dir, "ocr", "ocr_all_frames.txt"), "Login failed\nRetry button\n")
	return dir
}

func hasCheck(report ValidationReport, name string, ok bool) bool {
	for _, check := range report.Checks {
		if check.Name == name && check.OK == ok {
			return true
		}
	}
	return false
}

func hasWarning(report ValidationReport, substr string) bool {
	for _, w := range report.Warnings {
		if strings.Contains(w, substr) {
			return true
		}
	}
	return false
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
