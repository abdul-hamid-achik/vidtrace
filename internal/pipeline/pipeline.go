package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/abdul-hamid-achik/vidtrace/internal/artifacts"
	"github.com/abdul-hamid-achik/vidtrace/internal/ffmpeg"
	"github.com/abdul-hamid-achik/vidtrace/internal/tesseract"
	"github.com/abdul-hamid-achik/vidtrace/internal/timeline"
	"github.com/abdul-hamid-achik/vidtrace/internal/whisper"
)

type Options struct {
	SourceVideo     string
	FPS             float64
	OCRLanguage     string
	WhisperLanguage string
	WhisperModel    string
	OutputParentDir string
	BundleName      string
	Progress        io.Writer
	Now             func() time.Time
}

type Summary struct {
	OK              bool     `json:"ok"`
	SourceVideo     string   `json:"source_video"`
	OutputDir       string   `json:"output_dir"`
	Frames          int      `json:"frames"`
	OCRFiles        int      `json:"ocr_files"`
	TranscriptFiles []string `json:"transcript_files"`
	MetadataPath    string   `json:"metadata_path"`
	TimelinePath    string   `json:"timeline_path"`
	CombinedOCRPath string   `json:"combined_ocr_path"`
	DurationSeconds float64  `json:"duration_seconds,omitempty"`
}

type MetadataDocument struct {
	SchemaVersion   string   `json:"schema_version"`
	SourceVideo     string   `json:"source_video"`
	GeneratedAt     string   `json:"generated_at"`
	DurationSeconds float64  `json:"duration_seconds,omitempty"`
	Width           int      `json:"width,omitempty"`
	Height          int      `json:"height,omitempty"`
	VideoCodec      string   `json:"video_codec,omitempty"`
	AudioCodec      string   `json:"audio_codec,omitempty"`
	FrameRate       float64  `json:"frame_rate,omitempty"`
	ExtractFPS      float64  `json:"extract_fps"`
	OCRLanguages    []string `json:"ocr_languages"`
	WhisperLanguage string   `json:"whisper_language,omitempty"`
	WhisperModel    string   `json:"whisper_model"`
}

func Run(ctx context.Context, opts Options) (Summary, error) {
	if opts.FPS <= 0 {
		return Summary{}, fmt.Errorf("fps must be greater than 0")
	}
	if strings.TrimSpace(opts.OCRLanguage) == "" {
		return Summary{}, fmt.Errorf("ocr language is required")
	}
	if strings.TrimSpace(opts.WhisperModel) == "" {
		return Summary{}, fmt.Errorf("whisper model is required")
	}

	sourceVideo, err := filepath.Abs(opts.SourceVideo)
	if err != nil {
		return Summary{}, fmt.Errorf("resolve source video: %w", err)
	}
	if info, err := os.Stat(sourceVideo); err != nil {
		return Summary{}, fmt.Errorf("source video not found: %s", sourceVideo)
	} else if info.IsDir() {
		return Summary{}, fmt.Errorf("source video is a directory: %s", sourceVideo)
	}

	outputParentDir := opts.OutputParentDir
	if outputParentDir == "" {
		outputParentDir = "."
	}
	outputParentDir, err = filepath.Abs(outputParentDir)
	if err != nil {
		return Summary{}, fmt.Errorf("resolve output directory: %w", err)
	}
	if err := os.MkdirAll(outputParentDir, 0o755); err != nil {
		return Summary{}, fmt.Errorf("create output directory: %w", err)
	}

	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}

	bundleName := artifacts.SafeBundleName(sourceVideo, opts.BundleName)
	bundleDir := artifacts.BundlePath(outputParentDir, bundleName, now())
	if err := artifacts.EnsureBundleDirs(bundleDir); err != nil {
		return Summary{}, fmt.Errorf("create artifact bundle: %w", err)
	}

	progressf(opts.Progress, "Creating artifact bundle: %s", bundleDir)

	progressf(opts.Progress, "Capturing video metadata")
	mediaMetadata, err := ffmpeg.Probe(ctx, sourceVideo)
	if err != nil {
		return Summary{}, err
	}

	metadataDoc := MetadataDocument{
		SchemaVersion:   "1",
		SourceVideo:     sourceVideo,
		GeneratedAt:     now().UTC().Format(time.RFC3339),
		DurationSeconds: mediaMetadata.DurationSeconds,
		Width:           mediaMetadata.Width,
		Height:          mediaMetadata.Height,
		VideoCodec:      mediaMetadata.VideoCodec,
		AudioCodec:      mediaMetadata.AudioCodec,
		FrameRate:       mediaMetadata.FrameRate,
		ExtractFPS:      opts.FPS,
		OCRLanguages:    tesseract.SplitLanguages(opts.OCRLanguage),
		WhisperLanguage: opts.WhisperLanguage,
		WhisperModel:    opts.WhisperModel,
	}
	metadataPath := filepath.Join(bundleDir, "metadata.json")
	if err := artifacts.WriteJSON(metadataPath, metadataDoc); err != nil {
		return Summary{}, fmt.Errorf("write metadata.json: %w", err)
	}

	progressf(opts.Progress, "Extracting frames at %s fps", formatFloat(opts.FPS))
	framesPattern := filepath.Join(bundleDir, "frames", "frame_%04d.png")
	if err := ffmpeg.ExtractFrames(ctx, sourceVideo, opts.FPS, framesPattern); err != nil {
		return Summary{}, err
	}

	framePaths, err := filepath.Glob(filepath.Join(bundleDir, "frames", "frame_*.png"))
	if err != nil {
		return Summary{}, err
	}
	sort.Strings(framePaths)
	if len(framePaths) == 0 {
		return Summary{}, fmt.Errorf("no frames generated")
	}

	progressf(opts.Progress, "Running OCR on %d frames", len(framePaths))
	for _, framePath := range framePaths {
		base := strings.TrimSuffix(filepath.Base(framePath), filepath.Ext(framePath))
		outputBase := filepath.Join(bundleDir, "ocr", base)
		if err := tesseract.OCR(ctx, framePath, outputBase, opts.OCRLanguage); err != nil {
			return Summary{}, err
		}
	}

	ocrPaths, err := filepath.Glob(filepath.Join(bundleDir, "ocr", "frame_*.txt"))
	if err != nil {
		return Summary{}, err
	}
	sort.Strings(ocrPaths)
	combinedOCRPath := filepath.Join(bundleDir, "ocr", "ocr_all_frames.txt")
	if err := writeCombinedOCR(combinedOCRPath, sourceVideo, ocrPaths); err != nil {
		return Summary{}, err
	}

	progressf(opts.Progress, "Transcribing audio with Whisper %s", opts.WhisperModel)
	transcriptDir := filepath.Join(bundleDir, "transcript")
	if err := whisper.Transcribe(ctx, sourceVideo, transcriptDir, opts.WhisperModel, opts.WhisperLanguage); err != nil {
		return Summary{}, err
	}
	transcriptFiles, err := whisper.TranscriptFiles(transcriptDir)
	if err != nil {
		return Summary{}, err
	}

	progressf(opts.Progress, "Writing timeline.json")
	timelineDoc, err := timeline.Build(bundleDir, framePaths, opts.FPS, whisper.JSONPath(transcriptDir, sourceVideo))
	if err != nil {
		return Summary{}, err
	}
	timelinePath := filepath.Join(bundleDir, "timeline.json")
	if err := artifacts.WriteJSON(timelinePath, timelineDoc); err != nil {
		return Summary{}, fmt.Errorf("write timeline.json: %w", err)
	}

	summary := Summary{
		OK:              true,
		SourceVideo:     sourceVideo,
		OutputDir:       bundleDir,
		Frames:          len(framePaths),
		OCRFiles:        len(ocrPaths),
		TranscriptFiles: relFiles(bundleDir, transcriptFiles),
		MetadataPath:    artifacts.RelSlash(bundleDir, metadataPath),
		TimelinePath:    artifacts.RelSlash(bundleDir, timelinePath),
		CombinedOCRPath: artifacts.RelSlash(bundleDir, combinedOCRPath),
		DurationSeconds: mediaMetadata.DurationSeconds,
	}

	if err := writeReadme(filepath.Join(bundleDir, "README.txt"), summary); err != nil {
		return Summary{}, err
	}

	progressf(opts.Progress, "Done: %s", bundleDir)
	return summary, nil
}

func PrintHuman(w io.Writer, summary Summary) {
	writeLine(w, "vidtrace extract: ok")
	writeLine(w, "Output: %s", summary.OutputDir)
	writeLine(w, "Frames: %d", summary.Frames)
	writeLine(w, "OCR files: %d", summary.OCRFiles)
	writeLine(w, "Combined OCR: %s", summary.CombinedOCRPath)
	writeLine(w, "Metadata: %s", summary.MetadataPath)
	writeLine(w, "Timeline: %s", summary.TimelinePath)
	if len(summary.TranscriptFiles) > 0 {
		writeLine(w, "Transcript files:")
		for _, path := range summary.TranscriptFiles {
			writeLine(w, "  - %s", path)
		}
	}
}

func writeCombinedOCR(path, sourceVideo string, ocrPaths []string) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create combined OCR: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
	}()

	if _, err := fmt.Fprintf(file, "Video: %s\n", sourceVideo); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "Generated: %s\n\n", time.Now().Format(time.RFC3339)); err != nil {
		return err
	}
	for _, ocrPath := range ocrPaths {
		data, err := os.ReadFile(ocrPath)
		if err != nil {
			return fmt.Errorf("read OCR file: %w", err)
		}
		if _, err := fmt.Fprintf(file, "===== %s =====\n", filepath.Base(ocrPath)); err != nil {
			return err
		}
		if _, err := fmt.Fprint(file, string(data)); err != nil {
			return err
		}
		if len(data) == 0 || data[len(data)-1] != '\n' {
			if _, err := fmt.Fprintln(file); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(file); err != nil {
			return err
		}
	}
	return nil
}

func writeReadme(path string, summary Summary) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create bundle README: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
	}()

	if _, err := fmt.Fprintf(file, "Output folder: %s\n", summary.OutputDir); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "Source video: %s\n", summary.SourceVideo); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "Frames: %d\n", summary.Frames); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "OCR frame txt files: %d\n", summary.OCRFiles); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "Combined OCR: %s\n", summary.CombinedOCRPath); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "Metadata: %s\n", summary.MetadataPath); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file, "Timeline: %s\n", summary.TimelinePath); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(file, "Transcript files:"); err != nil {
		return err
	}
	for _, path := range summary.TranscriptFiles {
		if _, err := fmt.Fprintf(file, "- %s\n", path); err != nil {
			return err
		}
	}
	return nil
}

func relFiles(bundleDir string, paths []string) []string {
	files := make([]string, 0, len(paths))
	for _, path := range paths {
		files = append(files, artifacts.RelSlash(bundleDir, path))
	}
	sort.Strings(files)
	return files
}

func progressf(w io.Writer, format string, args ...any) {
	if w == nil {
		return
	}
	writeLine(w, ">>> "+format, args...)
}

func writeLine(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format+"\n", args...)
}

func formatFloat(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", value), "0"), ".")
}
