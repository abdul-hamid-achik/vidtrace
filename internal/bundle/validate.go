package bundle

import (
	"fmt"
	"os"
	"path/filepath"
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
	Summary         string            `json:"summary"`
}

type ValidationCheck struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
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

	var metadata Metadata
	metadataPath := filepath.Join(resolvedDir, "metadata.json")
	if err := readJSON(metadataPath, &metadata); err != nil {
		report.addCheck("metadata_json", false, "metadata.json", err.Error())
	} else {
		report.addCheck("metadata_json", true, "metadata.json", "metadata.json parses")
		report.addCheck("metadata_schema", metadata.SchemaVersion == artifacts.SchemaVersion, "metadata.json", "schema_version is "+artifacts.SchemaVersion)
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
