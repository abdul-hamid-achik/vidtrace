package bundle

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/abdul-hamid-achik/vidtrace/internal/artifacts"
	"github.com/abdul-hamid-achik/vidtrace/internal/timeline"
)

type ValidationReport struct {
	OK              bool              `json:"ok"`
	BundleDir       string            `json:"bundle_dir"`
	TimelineEntries int               `json:"timeline_entries"`
	EmptyOCREntries int               `json:"empty_ocr_entries"`
	Checks          []ValidationCheck `json:"checks"`
	Warnings        []string          `json:"warnings,omitempty"`
	Repairs         []RepairAction    `json:"repairs,omitempty"`
	Summary         string            `json:"summary"`
}

type ValidationCheck struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

type RepairAction struct {
	Stage   string   `json:"stage"`
	Reason  string   `json:"reason"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

func Validate(dir string) ValidationReport {
	report := ValidationReport{OK: true}
	if strings.TrimSpace(dir) == "" {
		report.addCheck("bundle_dir", false, "", "bundle path is required")
		return report.finalize()
	}

	resolvedDir, err := filepath.Abs(dir)
	if err != nil {
		report.addCheck("bundle_dir", false, dir, fmt.Sprintf("resolve bundle: %v", err))
		return report.finalize()
	}
	report.BundleDir = resolvedDir

	info, err := os.Stat(resolvedDir)
	if err != nil {
		report.addCheck("bundle_dir", false, resolvedDir, "bundle directory was not found")
		return report.finalize()
	}
	if !info.IsDir() {
		report.addCheck("bundle_dir", false, resolvedDir, "bundle path is not a directory")
		return report.finalize()
	}
	report.addCheck("bundle_dir", true, resolvedDir, "bundle directory exists")

	var trustedManifest *artifacts.StageManifest
	manifestPath := filepath.Join(resolvedDir, artifacts.StageManifestName)
	if _, err := os.Stat(manifestPath); err == nil {
		manifest, loadErr := artifacts.LoadStageManifest(manifestPath)
		if loadErr != nil {
			report.addCheck("stage_manifest", false, artifacts.StageManifestName, loadErr.Error())
		} else {
			trustedManifest = &manifest
			report.addCheck("stage_manifest", true, artifacts.StageManifestName, "stage manifest parses and is internally consistent")
			report.addCheck("stage_manifest_complete", manifest.Complete(), artifacts.StageManifestName, "all extraction stages are complete")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		report.addCheck("stage_manifest", false, artifacts.StageManifestName, err.Error())
	}

	var metadata Metadata
	metadataPath := filepath.Join(resolvedDir, "metadata.json")
	if err := readJSON(metadataPath, &metadata); err != nil {
		report.addCheck("metadata_json", false, "metadata.json", err.Error())
	} else {
		report.addCheck("metadata_json", true, "metadata.json", "metadata.json parses")
		report.addCheck("metadata_schema", metadata.SchemaVersion == artifacts.SchemaVersion, "metadata.json", "schema_version is "+artifacts.SchemaVersion)
	}

	if trustedManifest != nil {
		validity := ManifestStageValidity(resolvedDir, *trustedManifest)
		for _, stageName := range artifacts.StageOrder {
			report.addCheck("manifest_"+stageName, validity[stageName], artifacts.StageManifestName, stageName+" stage artifacts match the manifest")
		}
	}

	var timelineDoc timeline.Document
	timelinePath := filepath.Join(resolvedDir, "timeline.json")
	timelineOK := false
	if err := readJSON(timelinePath, &timelineDoc); err != nil {
		report.addCheck("timeline_json", false, "timeline.json", err.Error())
	} else {
		timelineOK = true
		report.TimelineEntries = len(timelineDoc.Entries)
		report.EmptyOCREntries = countEmptyOCR(timelineDoc)
		report.addCheck("timeline_json", true, "timeline.json", "timeline.json parses")
		report.addCheck("timeline_schema", timelineDoc.SchemaVersion == artifacts.SchemaVersion, "timeline.json", "schema_version is "+artifacts.SchemaVersion)
		report.addCheck("timeline_entries", len(timelineDoc.Entries) > 0, "timeline.json", fmt.Sprintf("%d timeline entries", len(timelineDoc.Entries)))
	}

	combinedOCRPath := filepath.Join(resolvedDir, "ocr", "ocr_all_frames.txt")
	if _, err := os.Stat(combinedOCRPath); err != nil {
		report.addCheck("combined_ocr", false, "ocr/ocr_all_frames.txt", "combined OCR file is missing")
	} else {
		report.addCheck("combined_ocr", true, "ocr/ocr_all_frames.txt", "combined OCR file exists")
	}

	if timelineOK {
		missingFrames, missingOCR := missingTimelinePaths(resolvedDir, timelineDoc)
		report.addCheck("timeline_frames", len(missingFrames) == 0, "timeline.json", missingPathMessage("frame paths exist", missingFrames))
		report.addCheck("timeline_ocr_paths", len(missingOCR) == 0, "timeline.json", missingPathMessage("OCR paths exist", missingOCR))
	}

	// Soft warning: when metadata declares a whisper model, expect at least one
	// transcript file. Silent videos are still valid bundles, so this is a
	// warning, not a hard check.
	if metadata.WhisperModel != "" {
		transcriptDir := filepath.Join(resolvedDir, "transcript")
		if entries, err := os.ReadDir(transcriptDir); err != nil || countFiles(entries) == 0 {
			report.Warnings = append(report.Warnings, "metadata declares a whisper model but transcript/ is empty or missing; silent video or transcription failure")
		}
	}

	// Soft warning: frame and OCR file counts should match. A drift suggests a
	// partial extraction or manual editing. This does not fail validation because
	// timeline-referenced paths are already checked above.
	frameCount := countFilesInDir(filepath.Join(resolvedDir, "frames"))
	ocrFrameCount := countFrameTXTFiles(filepath.Join(resolvedDir, "ocr"))
	if frameCount > 0 && ocrFrameCount > 0 && frameCount != ocrFrameCount {
		report.Warnings = append(report.Warnings, fmt.Sprintf("frame count (%d) differs from OCR frame txt count (%d); partial extraction or manual edit", frameCount, ocrFrameCount))
	}

	if trustedManifest != nil {
		report.Repairs = repairActions(report, *trustedManifest)
	}
	return report.finalize()
}

func (r *ValidationReport) addCheck(name string, ok bool, path, message string) {
	if !ok {
		r.OK = false
	}
	r.Checks = append(r.Checks, ValidationCheck{
		Name:    name,
		OK:      ok,
		Path:    filepath.ToSlash(path),
		Message: message,
	})
}

func (r ValidationReport) finalize() ValidationReport {
	passed := 0
	for _, check := range r.Checks {
		if check.OK {
			passed++
		}
	}
	if r.OK {
		r.Summary = fmt.Sprintf("Bundle is valid. %d/%d checks passed.", passed, len(r.Checks))
	} else {
		r.Summary = fmt.Sprintf("Bundle is invalid. %d/%d checks passed.", passed, len(r.Checks))
	}
	return r
}

func countEmptyOCR(doc timeline.Document) int {
	count := 0
	for _, entry := range doc.Entries {
		if strings.TrimSpace(entry.OCR.Text) == "" {
			count++
		}
	}
	return count
}

func missingTimelinePaths(bundleDir string, doc timeline.Document) ([]string, []string) {
	var missingFrames []string
	var missingOCR []string
	for _, entry := range doc.Entries {
		if !pathExists(bundleDir, entry.Frame) {
			missingFrames = append(missingFrames, entry.Frame)
		}
		if !pathExists(bundleDir, entry.OCR.Path) {
			missingOCR = append(missingOCR, entry.OCR.Path)
		}
	}
	return missingFrames, missingOCR
}

func pathExists(bundleDir, path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	candidate := filepath.FromSlash(path)
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(bundleDir, candidate)
	}
	_, err := os.Stat(candidate)
	return err == nil
}

func missingPathMessage(success string, missing []string) string {
	if len(missing) == 0 {
		return success
	}
	if len(missing) == 1 {
		return fmt.Sprintf("missing %s", missing[0])
	}
	return fmt.Sprintf("missing %d paths; first missing path is %s", len(missing), missing[0])
}

func countFiles(entries []os.DirEntry) int {
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			count++
		}
	}
	return count
}

func countFilesInDir(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	return countFiles(entries)
}

// countFrameTXTFiles counts only files matching the frame_*.txt pattern, so the
// combined ocr_all_frames.txt is excluded from the OCR frame count. This mirrors
// the pipeline's frame_*.txt glob and the AGENTS.md gotcha.
func countFrameTXTFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "frame_") && strings.HasSuffix(name, ".txt") {
			count++
		}
	}
	return count
}

// ManifestStageValidity applies the same artifact predicates used by resume
// reconciliation and validation, preventing the two paths from drifting.
func ManifestStageValidity(bundleDir string, manifest artifacts.StageManifest) map[string]bool {
	validity := make(map[string]bool, len(artifacts.StageOrder))
	stageComplete := func(name string) bool {
		return manifest.Stages[name].Status == artifacts.StageComplete
	}

	var metadata Metadata
	metadataOptions := artifacts.ManifestOptions{}
	metadataOK := readJSON(filepath.Join(bundleDir, "metadata.json"), &metadata) == nil
	if metadataOK {
		metadataOptions = artifacts.NormalizeManifestOptions(
			metadata.ExtractFPS,
			metadata.OCRLanguages,
			metadata.WhisperLanguage,
			metadata.WhisperModel,
		)
	}
	validity[artifacts.StageMetadata] = stageComplete(artifacts.StageMetadata) &&
		manifest.Stages[artifacts.StageMetadata].ArtifactCount == 1 &&
		metadataOK && metadata.SchemaVersion == artifacts.SchemaVersion &&
		filepath.Clean(metadata.SourceVideo) == manifest.Source.Path &&
		manifestOptionsEqual(metadataOptions, manifest.Options)

	framePaths, _ := filepath.Glob(filepath.Join(bundleDir, "frames", "frame_*.png"))
	sort.Strings(framePaths)
	validity[artifacts.StageFrames] = stageComplete(artifacts.StageFrames) &&
		len(framePaths) > 0 && len(framePaths) == manifest.Stages[artifacts.StageFrames].ArtifactCount &&
		allNonEmptyRegularFiles(framePaths)

	frameIDs := make([]string, 0, len(framePaths))
	for _, framePath := range framePaths {
		frameIDs = append(frameIDs, strings.TrimSuffix(filepath.Base(framePath), filepath.Ext(framePath)))
	}
	ocrStage := manifest.Stages[artifacts.StageOCR]
	validity[artifacts.StageOCR] = stageComplete(artifacts.StageOCR) &&
		ocrStage.TotalFrames == len(frameIDs) &&
		equalStrings(ocrStage.CompletedFrameIDs, frameIDs) &&
		allRegularFiles(ocrPathsForIDs(bundleDir, ocrStage.CompletedFrameIDs))

	transcriptPaths := expectedTranscriptPaths(bundleDir, manifest.Source.Path)
	validity[artifacts.StageTranscript] = stageComplete(artifacts.StageTranscript) &&
		manifest.Stages[artifacts.StageTranscript].ArtifactCount == len(transcriptPaths) &&
		validTranscriptSet(transcriptPaths)
	validity[artifacts.StageCombinedOCR] = stageComplete(artifacts.StageCombinedOCR) &&
		manifest.Stages[artifacts.StageCombinedOCR].ArtifactCount == 1 &&
		nonEmptyRegularFile(filepath.Join(bundleDir, "ocr", "ocr_all_frames.txt"))
	validity[artifacts.StageTimeline] = stageComplete(artifacts.StageTimeline) &&
		manifest.Stages[artifacts.StageTimeline].ArtifactCount == 1 &&
		validTimelineArtifacts(filepath.Join(bundleDir, "timeline.json"), bundleDir, framePaths)
	validity[artifacts.StageReadme] = stageComplete(artifacts.StageReadme) &&
		manifest.Stages[artifacts.StageReadme].ArtifactCount == 1 &&
		nonEmptyRegularFile(filepath.Join(bundleDir, "README.txt"))
	return validity
}

func manifestOptionsEqual(a, b artifacts.ManifestOptions) bool {
	return a.FPS == b.FPS && a.WhisperLanguage == b.WhisperLanguage &&
		a.WhisperModel == b.WhisperModel && equalStrings(a.OCRLanguages, b.OCRLanguages)
}

func expectedTranscriptPaths(bundleDir, sourcePath string) []string {
	base := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
	paths := make([]string, 0, 5)
	for _, extension := range []string{".json", ".srt", ".tsv", ".txt", ".vtt"} {
		paths = append(paths, filepath.Join(bundleDir, "transcript", base+extension))
	}
	return paths
}

func ocrPathsForIDs(bundleDir string, ids []string) []string {
	paths := make([]string, 0, len(ids))
	for _, id := range ids {
		paths = append(paths, filepath.Join(bundleDir, "ocr", id+".txt"))
	}
	return paths
}

func validTimelineArtifacts(path, bundleDir string, framePaths []string) bool {
	var document timeline.Document
	if readJSON(path, &document) != nil || document.SchemaVersion != artifacts.SchemaVersion || len(document.Entries) != len(framePaths) {
		return false
	}
	expectedOCR := make(map[string]string, len(framePaths))
	for _, framePath := range framePaths {
		frameRel := artifacts.RelSlash(bundleDir, framePath)
		frameID := strings.TrimSuffix(filepath.Base(framePath), filepath.Ext(framePath))
		expectedOCR[frameRel] = filepath.ToSlash(filepath.Join("ocr", frameID+".txt"))
	}
	seen := make(map[string]struct{}, len(document.Entries))
	for _, entry := range document.Entries {
		wantOCR, ok := expectedOCR[entry.Frame]
		if !ok || entry.OCR.Path != wantOCR {
			return false
		}
		if _, duplicate := seen[entry.Frame]; duplicate {
			return false
		}
		seen[entry.Frame] = struct{}{}
		if !regularFile(resolveBundlePath(bundleDir, entry.Frame)) || !regularFile(resolveBundlePath(bundleDir, entry.OCR.Path)) {
			return false
		}
	}
	return len(seen) == len(expectedOCR)
}

func resolveBundlePath(bundleDir, path string) string {
	path = filepath.FromSlash(path)
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(bundleDir, path)
}

func validTranscriptSet(paths []string) bool {
	if !allRegularFiles(paths) {
		return false
	}
	for _, path := range paths {
		if filepath.Ext(path) != ".json" {
			continue
		}
		data, err := os.ReadFile(path)
		return err == nil && json.Valid(data)
	}
	return false
}

func allRegularFiles(paths []string) bool {
	for _, path := range paths {
		if !regularFile(path) {
			return false
		}
	}
	return true
}

func allNonEmptyRegularFiles(paths []string) bool {
	for _, path := range paths {
		if !nonEmptyRegularFile(path) {
			return false
		}
	}
	return true
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func nonEmptyRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func repairActions(report ValidationReport, manifest artifacts.StageManifest) []RepairAction {
	reasons := make(map[string]string)
	for _, stageName := range artifacts.StageOrder {
		stage := manifest.Stages[stageName]
		if stage.Status == artifacts.StageComplete {
			continue
		}
		switch {
		case stage.Status == artifacts.StageFailed && strings.TrimSpace(stage.LastError) != "":
			reasons[stageName] = "stage failed: " + stage.LastError
		case stage.Status == artifacts.StageRunning:
			reasons[stageName] = "stage was interrupted while running"
		default:
			reasons[stageName] = "stage is " + stage.Status
		}
	}

	checkStages := map[string]string{
		"metadata_json":         artifacts.StageMetadata,
		"manifest_metadata":     artifacts.StageMetadata,
		"manifest_frames":       artifacts.StageFrames,
		"manifest_ocr":          artifacts.StageOCR,
		"manifest_transcript":   artifacts.StageTranscript,
		"manifest_combined_ocr": artifacts.StageCombinedOCR,
		"manifest_timeline":     artifacts.StageTimeline,
		"manifest_readme":       artifacts.StageReadme,
		"metadata_schema":       artifacts.StageMetadata,
		"frame_artifacts":       artifacts.StageFrames,
		"ocr_artifacts":         artifacts.StageOCR,
		"transcript_outputs":    artifacts.StageTranscript,
		"combined_ocr":          artifacts.StageCombinedOCR,
		"timeline_json":         artifacts.StageTimeline,
		"timeline_schema":       artifacts.StageTimeline,
		"timeline_entries":      artifacts.StageTimeline,
		"timeline_frames":       artifacts.StageTimeline,
		"timeline_ocr_paths":    artifacts.StageTimeline,
	}
	for _, check := range report.Checks {
		if check.OK {
			continue
		}
		stageName := checkStages[check.Name]
		if stageName == "" {
			continue
		}
		if _, exists := reasons[stageName]; !exists {
			reasons[stageName] = check.Message
		}
	}

	args := []string{
		"extract",
		"--resume-from", report.BundleDir,
		"--fps", strconv.FormatFloat(manifest.Options.FPS, 'f', -1, 64),
		"--ocr-lang", strings.Join(manifest.Options.OCRLanguages, "+"),
		"--whisper-lang", manifest.Options.WhisperLanguage,
		"--model", manifest.Options.WhisperModel,
		manifest.Source.Path,
	}
	repairs := make([]RepairAction, 0, len(reasons))
	for _, stageName := range artifacts.StageOrder {
		reason, ok := reasons[stageName]
		if !ok {
			continue
		}
		repairs = append(repairs, RepairAction{
			Stage:   stageName,
			Reason:  reason,
			Command: "vidtrace",
			Args:    append([]string(nil), args...),
		})
	}
	return repairs
}
