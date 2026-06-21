package pipeline

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/progress"
	"github.com/abdul-hamid-achik/vidtrace/internal/artifacts"
	"github.com/abdul-hamid-achik/vidtrace/internal/ffmpeg"
	"github.com/abdul-hamid-achik/vidtrace/internal/tesseract"
	"github.com/abdul-hamid-achik/vidtrace/internal/timeline"
	"github.com/abdul-hamid-achik/vidtrace/internal/whisper"
	"golang.org/x/sync/errgroup"
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
	// Interactive renders a live, in-place progress bar (for a TTY). When false,
	// progress is emitted as plain one-line-per-step output suitable for logs and
	// non-interactive callers.
	Interactive bool
	Now         func() time.Time
	// Concurrency caps the number of parallel OCR workers. When zero or negative,
	// it defaults to the number of available CPUs (capped to 8). OCR frames are
	// independent of each other and of Whisper, so they can run concurrently.
	Concurrency int
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

	// Fail fast when a requested OCR language is not installed, before creating a
	// bundle or extracting frames, so the user fixes language data up front
	// instead of after a long extraction. This runs after the cheaper input
	// checks so an invalid path still reports the clearer error first.
	requestedLanguages := tesseract.SplitLanguages(opts.OCRLanguage)
	availableLanguages, err := tesseract.AvailableLanguages(ctx)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return Summary{}, fmt.Errorf("tesseract is not installed; install it and any OCR language data (see docs/INSTALL.md), then run vidtrace doctor")
		}
		return Summary{}, err
	}
	if missing := tesseract.MissingLanguages(requestedLanguages, availableLanguages); len(missing) > 0 {
		return Summary{}, fmt.Errorf("OCR language data not installed: %s; install the tesseract language pack(s) (see docs/INSTALL.md) or change --ocr-lang", strings.Join(missing, ", "))
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
	bundleDir := artifacts.BundlePathUnique(outputParentDir, bundleName, now())
	if err := artifacts.EnsureBundleDirs(bundleDir); err != nil {
		return Summary{}, fmt.Errorf("create artifact bundle: %w", err)
	}

	const totalSteps = 7

	reporter := newProgressReporter(opts.Progress, opts.Interactive, totalSteps)

	reporter.step(1, "bundle", "created "+bundleDir)

	reporter.step(2, "metadata", "capturing video metadata")
	mediaMetadata, err := ffmpeg.Probe(ctx, sourceVideo)
	if err != nil {
		return Summary{}, err
	}

	metadataDoc := MetadataDocument{
		SchemaVersion:   artifacts.SchemaVersion,
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

	reporter.step(3, "frames", "extracting at "+formatFloat(opts.FPS)+" fps")
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

	// OCR (step 4) and Whisper transcription (step 5) are independent: OCR only
	// needs the extracted frames and Whisper only needs the source video. Run
	// them concurrently to cut wall-clock time on long videos. OCR is further
	// parallelized across frames with a bounded worker pool, while Whisper runs
	// as a single external process (its model is already CPU/GPU-bound).
	workers := opts.Concurrency
	if workers <= 0 {
		workers = runtime.NumCPU()
		if workers > 8 {
			workers = 8
		}
	}
	if workers > len(framePaths) {
		workers = len(framePaths)
	}

	transcriptDir := filepath.Join(bundleDir, "transcript")
	var (
		ocrPaths        []string
		transcriptFiles []string
	)

	group, groupCtx := errgroup.WithContext(ctx)

	reporter.startItems(4, "ocr", fmt.Sprintf("running OCR on %d frames (%d workers)", len(framePaths), workers))
	group.Go(func() error {
		defer reporter.finishItems()

		sem := make(chan struct{}, workers)
		var perFrame sync.WaitGroup
		// errCh captures the first per-frame error; it is buffered so a single
		// writer never blocks, even if other workers are still in flight.
		errCh := make(chan error, 1)
		done := uint64(0)
		var doneMu sync.Mutex

		for _, framePath := range framePaths {
			if err := groupCtx.Err(); err != nil {
				break
			}
			sem <- struct{}{}
			perFrame.Add(1)
			go func(framePath string) {
				defer perFrame.Done()
				defer func() { <-sem }()

				base := strings.TrimSuffix(filepath.Base(framePath), filepath.Ext(framePath))
				outputBase := filepath.Join(bundleDir, "ocr", base)
				if err := tesseract.OCR(groupCtx, framePath, outputBase, opts.OCRLanguage); err != nil {
					select {
					case errCh <- err:
					default:
					}
					return
				}

				doneMu.Lock()
				done++
				reporter.item(4, "ocr", int(done), len(framePaths), filepath.Base(framePath))
				doneMu.Unlock()
			}(framePath)
		}
		perFrame.Wait()

		select {
		case err := <-errCh:
			return err
		default:
			return groupCtx.Err()
		}
	})

	reporter.step(5, "transcript", "transcribing audio with Whisper "+opts.WhisperModel)
	group.Go(func() error {
		if err := whisper.Transcribe(groupCtx, sourceVideo, transcriptDir, opts.WhisperModel, opts.WhisperLanguage); err != nil {
			return err
		}
		files, err := whisper.TranscriptFiles(transcriptDir)
		if err != nil {
			return err
		}
		transcriptFiles = files
		return nil
	})

	if err := group.Wait(); err != nil {
		// errgroup returns the first non-nil error. If it is context.Canceled it
		// may mask a real tesseract/whisper failure, so prefer the original error
		// when available by re-checking ctx.
		if ctx.Err() != nil && err == context.Canceled {
			return Summary{}, ctx.Err()
		}
		return Summary{}, err
	}

	ocrPaths, err = filepath.Glob(filepath.Join(bundleDir, "ocr", "frame_*.txt"))
	if err != nil {
		return Summary{}, err
	}
	sort.Strings(ocrPaths)
	combinedOCRPath := filepath.Join(bundleDir, "ocr", "ocr_all_frames.txt")
	if err := writeCombinedOCR(combinedOCRPath, sourceVideo, ocrPaths, now().UTC()); err != nil {
		return Summary{}, err
	}

	reporter.step(6, "timeline", "writing timeline.json")
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

	reporter.step(7, "done", bundleDir)
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

func writeCombinedOCR(path, sourceVideo string, ocrPaths []string, generatedAt time.Time) (err error) {
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
	if _, err := fmt.Fprintf(file, "Generated: %s\n\n", generatedAt.Format(time.RFC3339)); err != nil {
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

// progressReporter renders pipeline progress. On a TTY (Interactive) it draws a
// styled bubbles progress bar and redraws the per-item OCR line in place; off a
// TTY it emits one plain line per step, which is friendly to logs and agents.
// All methods are safe for concurrent use because OCR frames run in parallel.
type progressReporter struct {
	w           io.Writer
	interactive bool
	totalSteps  int
	bar         progress.Model
	itemsOpen   bool
	mu          sync.Mutex
}

func newProgressReporter(w io.Writer, interactive bool, totalSteps int) *progressReporter {
	r := &progressReporter{w: w, interactive: interactive, totalSteps: totalSteps}
	if interactive && w != nil {
		r.bar = progress.New(progress.WithWidth(24), progress.WithoutPercentage())
	}
	return r
}

func (r *progressReporter) step(step int, label, detail string) {
	if r == nil || r.w == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.endItemsLine()
	writeLine(r.w, "[%d/%d] %-10s %s %s", step, r.totalSteps, label, r.renderBar(step, r.totalSteps), detail)
}

// startItems begins a per-item phase. Off a TTY it prints a single step line;
// on a TTY the live item() redraws carry the phase.
func (r *progressReporter) startItems(step int, label, detail string) {
	if r == nil || r.w == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.interactive {
		r.itemsOpen = true
		return
	}
	writeLine(r.w, "[%d/%d] %-10s %s %s", step, r.totalSteps, label, r.renderBar(step, r.totalSteps), detail)
}

func (r *progressReporter) item(step int, label string, done, itemTotal int, detail string) {
	if r == nil || r.w == nil || !r.interactive {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.itemsOpen = true
	// \r returns to column 0; \x1b[K clears any leftover from a longer prior line.
	_, _ = fmt.Fprintf(r.w, "\r[%d/%d] %-10s %s %d/%d %s\x1b[K", step, r.totalSteps, label, r.renderBar(done, itemTotal), done, itemTotal, detail)
}

func (r *progressReporter) finishItems() {
	if r == nil || r.w == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.endItemsLine()
}

func (r *progressReporter) endItemsLine() {
	if r.itemsOpen {
		_, _ = fmt.Fprint(r.w, "\n")
		r.itemsOpen = false
	}
}

func (r *progressReporter) renderBar(done, total int) string {
	if r.interactive {
		return r.bar.ViewAs(fraction(done, total))
	}
	return progressBar(done, total, 18)
}

func fraction(done, total int) float64 {
	if total <= 0 {
		return 0
	}
	f := float64(done) / float64(total)
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

func progressBar(done, total, width int) string {
	if width <= 0 {
		return "[]"
	}
	if total <= 0 {
		total = 1
	}
	if done < 0 {
		done = 0
	}
	if done > total {
		done = total
	}
	filled := done * width / total
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat(".", width-filled) + "]"
}

func writeLine(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format+"\n", args...)
}

func formatFloat(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", value), "0"), ".")
}
