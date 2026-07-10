package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/abdul-hamid-achik/vidtrace/internal/artifacts"
	"github.com/abdul-hamid-achik/vidtrace/internal/bundle"
	"github.com/abdul-hamid-achik/vidtrace/internal/ffmpeg"
	"github.com/abdul-hamid-achik/vidtrace/internal/tesseract"
	"github.com/abdul-hamid-achik/vidtrace/internal/timeline"
	"github.com/abdul-hamid-achik/vidtrace/internal/whisper"
	"golang.org/x/sync/errgroup"
)

func run(ctx context.Context, opts Options) (Summary, error) {
	if opts.FPS <= 0 {
		return Summary{}, fmt.Errorf("fps must be greater than 0")
	}
	if strings.TrimSpace(opts.OCRLanguage) == "" {
		return Summary{}, fmt.Errorf("ocr language is required")
	}
	if strings.TrimSpace(opts.WhisperModel) == "" {
		return Summary{}, fmt.Errorf("whisper model is required")
	}

	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	manifestOptions := artifacts.NormalizeManifestOptions(
		opts.FPS,
		tesseract.SplitLanguages(opts.OCRLanguage),
		opts.WhisperLanguage,
		opts.WhisperModel,
	)
	source, fingerprint, err := artifacts.FingerprintSource(opts.SourceVideo, manifestOptions)
	if err != nil {
		return Summary{}, err
	}
	sourceVideo := source.Path

	outputParentDir := opts.OutputParentDir
	if outputParentDir == "" {
		outputParentDir = "."
	}
	outputParentDir, err = filepath.Abs(outputParentDir)
	if err != nil {
		return Summary{}, fmt.Errorf("resolve output directory: %w", err)
	}

	bundleDir, manifest, fresh, err := selectBundle(opts, outputParentDir, source, manifestOptions, fingerprint, now)
	if err != nil {
		return Summary{}, err
	}
	if fresh {
		if err := os.MkdirAll(outputParentDir, 0o755); err != nil {
			return Summary{}, fmt.Errorf("create output directory: %w", err)
		}
		if err := os.MkdirAll(bundleDir, 0o755); err != nil {
			return Summary{}, fmt.Errorf("create artifact bundle: %w", err)
		}
	}
	bundleLock, err := artifacts.LockBundle(bundleDir)
	if err != nil {
		return Summary{}, err
	}
	defer func() { _ = bundleLock.Close() }()

	store := artifacts.NewManifestStore(filepath.Join(bundleDir, artifacts.StageManifestName), now)
	if fresh {
		if err := artifacts.EnsureBundleDirs(bundleDir); err != nil {
			return Summary{}, fmt.Errorf("create artifact bundle: %w", err)
		}
		if err := store.Initialize(manifest); err != nil {
			return Summary{}, fmt.Errorf("initialize stage manifest: %w", err)
		}
	} else {
		manifest, err = store.Load()
		if err != nil {
			return Summary{}, fmt.Errorf("reload stage manifest under bundle lock: %w", err)
		}
		if manifest.Fingerprint != fingerprint || manifest.Source != source || !manifestOptionsEqual(manifest.Options, manifestOptions) {
			return Summary{}, fmt.Errorf("resume source/options fingerprint mismatch for %s; bundle=%s requested=%s", bundleDir, manifest.Fingerprint, fingerprint)
		}
		if strings.TrimSpace(opts.ResumeFrom) == "" && manifest.Complete() {
			return Summary{}, fmt.Errorf("compatible bundle completed before resume lock was acquired: %s; no incomplete bundle remains", bundleDir)
		}
		if err := artifacts.EnsureBundleDirs(bundleDir); err != nil {
			return Summary{}, fmt.Errorf("repair artifact bundle directories: %w", err)
		}
		manifest, err = reconcileManifest(bundleDir, manifest)
		if err != nil {
			return Summary{}, err
		}
		if _, err := store.Update(func(current *artifacts.StageManifest) error {
			*current = manifest
			return nil
		}); err != nil {
			return Summary{}, fmt.Errorf("persist reconciled stage manifest: %w", err)
		}
	}

	const totalSteps = 7
	reporter := newProgressReporter(opts.Progress, opts.Interactive, totalSteps)
	if fresh {
		reporter.step(1, "bundle", "created "+bundleDir)
	} else {
		reporter.step(1, "bundle", "resuming "+bundleDir)
	}

	needsOCRTool := manifest.Stages[artifacts.StageFrames].Status != artifacts.StageComplete ||
		manifest.Stages[artifacts.StageOCR].Status != artifacts.StageComplete
	if needsOCRTool {
		requestedLanguages := tesseract.SplitLanguages(opts.OCRLanguage)
		availableLanguages, languageErr := tesseract.AvailableLanguages(ctx)
		if languageErr != nil {
			if errors.Is(languageErr, exec.ErrNotFound) {
				return Summary{}, fmt.Errorf("tesseract is not installed; install it and any OCR language data (see docs/INSTALL.md), then run vidtrace doctor")
			}
			return Summary{}, languageErr
		}
		if missing := tesseract.MissingLanguages(requestedLanguages, availableLanguages); len(missing) > 0 {
			return Summary{}, fmt.Errorf("OCR language data not installed: %s; install the tesseract language pack(s) (see docs/INSTALL.md) or change --ocr-lang", strings.Join(missing, ", "))
		}
	}

	manifest, err = store.Load()
	if err != nil {
		return Summary{}, err
	}
	metadataPath := filepath.Join(bundleDir, "metadata.json")
	var mediaMetadata ffmpeg.Metadata
	if manifest.Stages[artifacts.StageMetadata].Status == artifacts.StageComplete {
		metadataDoc, readErr := readMetadataDocument(metadataPath)
		if readErr != nil {
			return Summary{}, readErr
		}
		mediaMetadata = ffmpeg.Metadata{
			DurationSeconds: metadataDoc.DurationSeconds,
			Width:           metadataDoc.Width,
			Height:          metadataDoc.Height,
			VideoCodec:      metadataDoc.VideoCodec,
			AudioCodec:      metadataDoc.AudioCodec,
			FrameRate:       metadataDoc.FrameRate,
		}
		reporter.step(2, "metadata", "resume: metadata.json is complete")
	} else {
		reporter.step(2, "metadata", "capturing video metadata")
		if err := markStageRunning(store, artifacts.StageMetadata, 0, 0, nil); err != nil {
			return Summary{}, err
		}
		mediaMetadata, err = ffmpeg.Probe(ctx, sourceVideo)
		if err != nil {
			return Summary{}, failStage(store, artifacts.StageMetadata, err)
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
		if err := artifacts.WriteJSON(metadataPath, metadataDoc); err != nil {
			return Summary{}, failStage(store, artifacts.StageMetadata, fmt.Errorf("write metadata.json: %w", err))
		}
		if err := markStageComplete(store, artifacts.StageMetadata, 1, 0, nil); err != nil {
			return Summary{}, err
		}
	}

	manifest, err = store.Load()
	if err != nil {
		return Summary{}, err
	}
	framesPattern := filepath.Join(bundleDir, "frames", "frame_%04d.png")
	framePaths, err := frameFiles(bundleDir)
	if err != nil {
		return Summary{}, err
	}
	if manifest.Stages[artifacts.StageFrames].Status == artifacts.StageComplete {
		reporter.step(3, "frames", fmt.Sprintf("resume: %d frames already extracted", len(framePaths)))
	} else {
		reporter.step(3, "frames", "extracting at "+formatFloat(opts.FPS)+" fps")
		if err := markStageRunning(store, artifacts.StageFrames, 0, 0, nil); err != nil {
			return Summary{}, err
		}
		if err := removeMatches(framePaths); err != nil {
			return Summary{}, failStage(store, artifacts.StageFrames, err)
		}
		if err := ffmpeg.ExtractFrames(ctx, sourceVideo, opts.FPS, framesPattern); err != nil {
			return Summary{}, failStage(store, artifacts.StageFrames, err)
		}
		framePaths, err = frameFiles(bundleDir)
		if err != nil {
			return Summary{}, failStage(store, artifacts.StageFrames, err)
		}
		if len(framePaths) == 0 {
			return Summary{}, failStage(store, artifacts.StageFrames, fmt.Errorf("no frames generated"))
		}
		if err := markStageComplete(store, artifacts.StageFrames, len(framePaths), 0, nil); err != nil {
			return Summary{}, err
		}
	}
	if len(framePaths) == 0 {
		return Summary{}, fmt.Errorf("no frames generated")
	}

	manifest, err = store.Load()
	if err != nil {
		return Summary{}, err
	}
	ocrStage := manifest.Stages[artifacts.StageOCR]
	frameIDs := frameIDsFromPaths(framePaths)
	pendingOCR := missingIDs(frameIDs, ocrStage.CompletedFrameIDs)
	transcriptDir := filepath.Join(bundleDir, "transcript")
	transcriptStage := manifest.Stages[artifacts.StageTranscript]

	ocrNeeded := len(pendingOCR) > 0 || ocrStage.Status != artifacts.StageComplete
	transcriptNeeded := transcriptStage.Status != artifacts.StageComplete
	workers := 0
	if ocrNeeded {
		workers = workerCount(opts.Concurrency, len(pendingOCR))
		if err := markStageRunning(store, artifacts.StageOCR, len(ocrStage.CompletedFrameIDs), len(framePaths), ocrStage.CompletedFrameIDs); err != nil {
			return Summary{}, err
		}
	}
	if transcriptNeeded {
		if err := markStageRunning(store, artifacts.StageTranscript, 0, 0, nil); err != nil {
			return Summary{}, err
		}
	}

	group, groupCtx := errgroup.WithContext(ctx)
	if !ocrNeeded {
		reporter.startItems(4, "ocr", fmt.Sprintf("resume: OCR complete for %d frames", len(framePaths)))
		reporter.finishItems()
	} else {
		reporter.startItems(4, "ocr", fmt.Sprintf("running OCR on %d incomplete frames (%d workers)", len(pendingOCR), workers))
		group.Go(func() error {
			defer reporter.finishItems()
			return runOCRFrames(groupCtx, bundleDir, framePaths, pendingOCR, opts.OCRLanguage, workers, len(framePaths), store, reporter)
		})
	}

	if !transcriptNeeded {
		reporter.step(5, "transcript", fmt.Sprintf("resume: %d transcript files are complete", transcriptStage.ArtifactCount))
	} else {
		reporter.step(5, "transcript", "transcribing audio with Whisper "+opts.WhisperModel)
		group.Go(func() error {
			if err := runWhisperAtomic(groupCtx, sourceVideo, transcriptDir, opts.WhisperModel, opts.WhisperLanguage); err != nil {
				return failStage(store, artifacts.StageTranscript, err)
			}
			return markStageComplete(store, artifacts.StageTranscript, len(expectedTranscriptPaths(transcriptDir, sourceVideo)), 0, nil)
		})
	}

	if err := group.Wait(); err != nil {
		if ctx.Err() != nil && err == context.Canceled {
			return Summary{}, ctx.Err()
		}
		return Summary{}, err
	}
	if err := verifySourceStable(store, source, manifestOptions, fingerprint); err != nil {
		return Summary{}, err
	}

	ocrPaths := ocrPathsForFrames(bundleDir, framePaths)
	transcriptFiles := expectedTranscriptPaths(transcriptDir, sourceVideo)
	manifest, err = store.Load()
	if err != nil {
		return Summary{}, err
	}

	combinedOCRPath := filepath.Join(bundleDir, "ocr", "ocr_all_frames.txt")
	if manifest.Stages[artifacts.StageCombinedOCR].Status != artifacts.StageComplete {
		if err := markStageRunning(store, artifacts.StageCombinedOCR, 0, 0, nil); err != nil {
			return Summary{}, err
		}
		if err := writeCombinedOCR(combinedOCRPath, sourceVideo, ocrPaths, now().UTC()); err != nil {
			return Summary{}, failStage(store, artifacts.StageCombinedOCR, err)
		}
		if err := markStageComplete(store, artifacts.StageCombinedOCR, 1, 0, nil); err != nil {
			return Summary{}, err
		}
	}

	manifest, err = store.Load()
	if err != nil {
		return Summary{}, err
	}
	timelinePath := filepath.Join(bundleDir, "timeline.json")
	if manifest.Stages[artifacts.StageTimeline].Status != artifacts.StageComplete {
		reporter.step(6, "timeline", "writing timeline.json")
		if err := markStageRunning(store, artifacts.StageTimeline, 0, 0, nil); err != nil {
			return Summary{}, err
		}
		timelineDoc, buildErr := timeline.Build(bundleDir, framePaths, opts.FPS, whisper.JSONPath(transcriptDir, sourceVideo))
		if buildErr != nil {
			return Summary{}, failStage(store, artifacts.StageTimeline, buildErr)
		}
		if err := artifacts.WriteJSON(timelinePath, timelineDoc); err != nil {
			return Summary{}, failStage(store, artifacts.StageTimeline, fmt.Errorf("write timeline.json: %w", err))
		}
		if err := markStageComplete(store, artifacts.StageTimeline, 1, 0, nil); err != nil {
			return Summary{}, err
		}
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

	manifest, err = store.Load()
	if err != nil {
		return Summary{}, err
	}
	if manifest.Stages[artifacts.StageReadme].Status != artifacts.StageComplete {
		if err := markStageRunning(store, artifacts.StageReadme, 0, 0, nil); err != nil {
			return Summary{}, err
		}
		if err := writeReadme(filepath.Join(bundleDir, "README.txt"), summary); err != nil {
			return Summary{}, failStage(store, artifacts.StageReadme, err)
		}
		if err := markStageComplete(store, artifacts.StageReadme, 1, 0, nil); err != nil {
			return Summary{}, err
		}
	}

	reporter.step(7, "done", bundleDir)
	return summary, nil
}

func selectBundle(opts Options, outputParent string, source artifacts.ManifestSource, options artifacts.ManifestOptions, fingerprint string, now func() time.Time) (string, artifacts.StageManifest, bool, error) {
	if strings.TrimSpace(opts.ResumeFrom) != "" {
		bundleDir, err := filepath.Abs(opts.ResumeFrom)
		if err != nil {
			return "", artifacts.StageManifest{}, false, fmt.Errorf("resolve resume bundle: %w", err)
		}
		manifestPath := filepath.Join(bundleDir, artifacts.StageManifestName)
		manifest, err := artifacts.LoadStageManifest(manifestPath)
		if err != nil {
			if os.IsNotExist(errors.Unwrap(err)) || errors.Is(err, os.ErrNotExist) {
				return "", artifacts.StageManifest{}, false, fmt.Errorf("cannot resume bundle %s without %s; run a fresh extraction without --resume or --resume-from", bundleDir, artifacts.StageManifestName)
			}
			return "", artifacts.StageManifest{}, false, fmt.Errorf("cannot resume bundle %s: %w", bundleDir, err)
		}
		if manifest.Fingerprint != fingerprint || manifest.Source != source || !manifestOptionsEqual(manifest.Options, options) {
			return "", artifacts.StageManifest{}, false, fmt.Errorf("resume source/options fingerprint mismatch for %s; bundle=%s requested=%s", bundleDir, manifest.Fingerprint, fingerprint)
		}
		return bundleDir, manifest, false, nil
	}

	if opts.Resume {
		entries, err := os.ReadDir(outputParent)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", artifacts.StageManifest{}, false, fmt.Errorf("no compatible incomplete bundle found under %s; run a fresh extraction without --resume", outputParent)
			}
			return "", artifacts.StageManifest{}, false, fmt.Errorf("scan resume bundles: %w", err)
		}
		type candidate struct {
			dir      string
			manifest artifacts.StageManifest
		}
		var candidates []candidate
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			dir := filepath.Join(outputParent, entry.Name())
			manifest, loadErr := artifacts.LoadStageManifest(filepath.Join(dir, artifacts.StageManifestName))
			if loadErr != nil || manifest.Fingerprint != fingerprint {
				continue
			}
			reconciled, reconcileErr := reconcileManifest(dir, manifest)
			if reconcileErr != nil || reconciled.Complete() {
				continue
			}
			candidates = append(candidates, candidate{dir: dir, manifest: manifest})
		}
		sort.Slice(candidates, func(i, j int) bool { return candidates[i].dir < candidates[j].dir })
		switch len(candidates) {
		case 0:
			return "", artifacts.StageManifest{}, false, fmt.Errorf("no compatible incomplete bundle found under %s; run a fresh extraction without --resume", outputParent)
		case 1:
			return candidates[0].dir, candidates[0].manifest, false, nil
		default:
			dirs := make([]string, len(candidates))
			for i := range candidates {
				dirs[i] = candidates[i].dir
			}
			return "", artifacts.StageManifest{}, false, fmt.Errorf("multiple compatible incomplete bundles found under %s: %s; use --resume-from <bundle>", outputParent, strings.Join(dirs, ", "))
		}
	}

	bundleName := artifacts.SafeBundleName(source.Path, opts.BundleName)
	bundleDir := artifacts.BundlePathUnique(outputParent, bundleName, now())
	manifest := artifacts.NewStageManifest(source, options, fingerprint, now())
	return bundleDir, manifest, true, nil
}

func reconcileManifest(bundleDir string, manifest artifacts.StageManifest) (artifacts.StageManifest, error) {
	framePaths, err := frameFiles(bundleDir)
	if err != nil {
		return artifacts.StageManifest{}, err
	}
	frameIDs := frameIDsFromPaths(framePaths)

	for _, stageName := range artifacts.StageOrder {
		stage := manifest.Stages[stageName]
		if stage.Status == artifacts.StageRunning || stage.Status == artifacts.StageFailed {
			stage.Status = artifacts.StagePending
			stage.LastError = ""
			if stageName != artifacts.StageOCR {
				stage.ArtifactCount = 0
			}
			manifest.Stages[stageName] = stage
			invalidateDependents(&manifest, stageName)
		}
	}

	validity := bundle.ManifestStageValidity(bundleDir, manifest)

	for _, stageName := range artifacts.StageOrder {
		stage := manifest.Stages[stageName]
		if stage.Status != artifacts.StageComplete {
			if stageName == artifacts.StageOCR {
				stage.CompletedFrameIDs = filterValidOCRIDs(bundleDir, frameIDs, stage.CompletedFrameIDs)
				stage.ArtifactCount = len(stage.CompletedFrameIDs)
				stage.TotalFrames = len(frameIDs)
				manifest.Stages[stageName] = stage
			}
			invalidateDependents(&manifest, stageName)
			continue
		}
		if validity[stageName] {
			continue
		}
		stage.Status = artifacts.StagePending
		stage.LastError = ""
		stage.ArtifactCount = 0
		if stageName == artifacts.StageOCR {
			stage.CompletedFrameIDs = filterValidOCRIDs(bundleDir, frameIDs, stage.CompletedFrameIDs)
			stage.ArtifactCount = len(stage.CompletedFrameIDs)
			stage.TotalFrames = len(frameIDs)
		}
		manifest.Stages[stageName] = stage
		invalidateDependents(&manifest, stageName)
	}
	return manifest, nil
}

func invalidateDependents(manifest *artifacts.StageManifest, stageName string) {
	dependencies := map[string][]string{
		artifacts.StageMetadata:    {artifacts.StageReadme},
		artifacts.StageFrames:      {artifacts.StageOCR, artifacts.StageCombinedOCR, artifacts.StageTimeline, artifacts.StageReadme},
		artifacts.StageOCR:         {artifacts.StageCombinedOCR, artifacts.StageTimeline, artifacts.StageReadme},
		artifacts.StageTranscript:  {artifacts.StageTimeline, artifacts.StageReadme},
		artifacts.StageCombinedOCR: {artifacts.StageReadme},
		artifacts.StageTimeline:    {artifacts.StageReadme},
	}
	for _, dependent := range dependencies[stageName] {
		stage := manifest.Stages[dependent]
		stage.Status = artifacts.StagePending
		stage.LastError = ""
		if dependent != artifacts.StageOCR {
			stage.ArtifactCount = 0
		}
		if stageName == artifacts.StageFrames && dependent == artifacts.StageOCR {
			stage.ArtifactCount = 0
			stage.TotalFrames = 0
			stage.CompletedFrameIDs = nil
		}
		manifest.Stages[dependent] = stage
	}
}

func markStageRunning(store *artifacts.ManifestStore, name string, count, total int, completed []string) error {
	_, err := store.Update(func(manifest *artifacts.StageManifest) error {
		stage := manifest.Stages[name]
		stage.Status = artifacts.StageRunning
		stage.ArtifactCount = count
		stage.LastError = ""
		if name == artifacts.StageOCR {
			stage.TotalFrames = total
			stage.CompletedFrameIDs = sortedUnique(completed)
		}
		manifest.Stages[name] = stage
		return nil
	})
	return err
}

func markStageComplete(store *artifacts.ManifestStore, name string, count, total int, completed []string) error {
	_, err := store.Update(func(manifest *artifacts.StageManifest) error {
		stage := manifest.Stages[name]
		stage.Status = artifacts.StageComplete
		stage.ArtifactCount = count
		stage.LastError = ""
		if name == artifacts.StageOCR {
			stage.TotalFrames = total
			stage.CompletedFrameIDs = sortedUnique(completed)
		}
		manifest.Stages[name] = stage
		return nil
	})
	return err
}

func failStage(store *artifacts.ManifestStore, name string, stageErr error) error {
	_, manifestErr := store.Update(func(manifest *artifacts.StageManifest) error {
		stage := manifest.Stages[name]
		stage.Status = artifacts.StageFailed
		stage.LastError = stageErr.Error()
		manifest.Stages[name] = stage
		return nil
	})
	if manifestErr != nil {
		return fmt.Errorf("%v (also failed to checkpoint stage error: %w)", stageErr, manifestErr)
	}
	return stageErr
}

func verifySourceStable(store *artifacts.ManifestStore, expected artifacts.ManifestSource, options artifacts.ManifestOptions, fingerprint string) error {
	actual, actualFingerprint, err := artifacts.FingerprintSource(expected.Path, options)
	if err == nil && actual == expected && actualFingerprint == fingerprint {
		return nil
	}
	reason := "source video changed during extraction; all source-dependent checkpoints were invalidated"
	if err != nil {
		reason += ": " + err.Error()
	}
	_, checkpointErr := store.Update(func(manifest *artifacts.StageManifest) error {
		for _, stageName := range artifacts.StageOrder {
			manifest.Stages[stageName] = artifacts.ManifestStage{Status: artifacts.StagePending}
		}
		metadata := manifest.Stages[artifacts.StageMetadata]
		metadata.Status = artifacts.StageFailed
		metadata.LastError = reason
		manifest.Stages[artifacts.StageMetadata] = metadata
		return nil
	})
	if checkpointErr != nil {
		return fmt.Errorf("%s (also failed to invalidate checkpoints: %w)", reason, checkpointErr)
	}
	return fmt.Errorf("%s", reason)
}

func runOCRFrames(ctx context.Context, bundleDir string, framePaths, pendingIDs []string, language string, workers, total int, store *artifacts.ManifestStore, reporter *progressReporter) error {
	pathsByID := make(map[string]string, len(framePaths))
	for _, framePath := range framePaths {
		pathsByID[frameID(framePath)] = framePath
	}
	sem := make(chan struct{}, workers)
	var perFrame sync.WaitGroup
	errCh := make(chan error, 1)
	completedAtStart := total - len(pendingIDs)
	done := completedAtStart
	var doneMu sync.Mutex

	for _, id := range pendingIDs {
		if err := ctx.Err(); err != nil {
			break
		}
		sem <- struct{}{}
		perFrame.Add(1)
		go func(id, framePath string) {
			defer perFrame.Done()
			defer func() { <-sem }()
			if err := runOneOCRAtomic(ctx, bundleDir, id, framePath, language); err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
			if _, err := store.Update(func(manifest *artifacts.StageManifest) error {
				stage := manifest.Stages[artifacts.StageOCR]
				stage.CompletedFrameIDs = sortedUnique(append(stage.CompletedFrameIDs, id))
				stage.ArtifactCount = len(stage.CompletedFrameIDs)
				manifest.Stages[artifacts.StageOCR] = stage
				return nil
			}); err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
			doneMu.Lock()
			done++
			reporter.item(4, "ocr", done, total, filepath.Base(framePath))
			doneMu.Unlock()
		}(id, pathsByID[id])
	}
	perFrame.Wait()

	select {
	case err := <-errCh:
		return failStage(store, artifacts.StageOCR, err)
	default:
		if err := ctx.Err(); err != nil {
			return failStage(store, artifacts.StageOCR, err)
		}
	}
	manifest, err := store.Load()
	if err != nil {
		return err
	}
	completed := manifest.Stages[artifacts.StageOCR].CompletedFrameIDs
	if len(completed) != total {
		return failStage(store, artifacts.StageOCR, fmt.Errorf("OCR completed %d of %d frames", len(completed), total))
	}
	return markStageComplete(store, artifacts.StageOCR, len(completed), total, completed)
}

func runOneOCRAtomic(ctx context.Context, bundleDir, id, framePath, language string) error {
	ocrDir := filepath.Join(bundleDir, "ocr")
	temp, err := os.CreateTemp(ocrDir, "."+id+"-*.base")
	if err != nil {
		return fmt.Errorf("create OCR temp output: %w", err)
	}
	tempBase := temp.Name()
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempBase)
		return err
	}
	if err := os.Remove(tempBase); err != nil {
		return err
	}
	tempOutput := tempBase + ".txt"
	defer func() { _ = os.Remove(tempOutput) }()
	if err := tesseract.OCR(ctx, framePath, tempBase, language); err != nil {
		return err
	}
	if err := syncFile(tempOutput); err != nil {
		return err
	}
	finalPath := filepath.Join(ocrDir, id+".txt")
	if err := os.Rename(tempOutput, finalPath); err != nil {
		return fmt.Errorf("promote OCR output: %w", err)
	}
	return syncDirectory(ocrDir)
}

func runWhisperAtomic(ctx context.Context, sourceVideo, transcriptDir, model, language string) error {
	bundleDir := filepath.Dir(transcriptDir)
	tempDir, err := os.MkdirTemp(bundleDir, ".transcript-*")
	if err != nil {
		return fmt.Errorf("create transcript temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()
	if err := whisper.Transcribe(ctx, sourceVideo, tempDir, model, language); err != nil {
		return err
	}
	tempPaths := expectedTranscriptPaths(tempDir, sourceVideo)
	if !allRegularFiles(tempPaths) {
		return fmt.Errorf("whisper did not produce the expected output set: %s", strings.Join(baseNames(tempPaths), ", "))
	}
	for _, tempPath := range tempPaths {
		if err := syncFile(tempPath); err != nil {
			return err
		}
		finalPath := filepath.Join(transcriptDir, filepath.Base(tempPath))
		if err := os.Rename(tempPath, finalPath); err != nil {
			return fmt.Errorf("promote transcript output %s: %w", filepath.Base(tempPath), err)
		}
	}
	return syncDirectory(transcriptDir)
}

func expectedTranscriptPaths(dir, sourceVideo string) []string {
	base := strings.TrimSuffix(filepath.Base(sourceVideo), filepath.Ext(sourceVideo))
	exts := []string{".json", ".srt", ".tsv", ".txt", ".vtt"}
	paths := make([]string, 0, len(exts))
	for _, ext := range exts {
		paths = append(paths, filepath.Join(dir, base+ext))
	}
	return paths
}

func frameFiles(bundleDir string) ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(bundleDir, "frames", "frame_*.png"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func ocrPathsForFrames(bundleDir string, framePaths []string) []string {
	paths := make([]string, 0, len(framePaths))
	for _, framePath := range framePaths {
		paths = append(paths, filepath.Join(bundleDir, "ocr", frameID(framePath)+".txt"))
	}
	return paths
}

func frameIDsFromPaths(paths []string) []string {
	ids := make([]string, 0, len(paths))
	for _, path := range paths {
		ids = append(ids, frameID(path))
	}
	return ids
}

func frameID(path string) string {
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

func missingIDs(all, completed []string) []string {
	seen := make(map[string]struct{}, len(completed))
	for _, id := range completed {
		seen[id] = struct{}{}
	}
	var missing []string
	for _, id := range all {
		if _, ok := seen[id]; !ok {
			missing = append(missing, id)
		}
	}
	return missing
}

func filterValidOCRIDs(bundleDir string, frameIDs, completed []string) []string {
	frameSet := make(map[string]struct{}, len(frameIDs))
	for _, id := range frameIDs {
		frameSet[id] = struct{}{}
	}
	valid := make([]string, 0, len(completed))
	for _, id := range completed {
		if _, ok := frameSet[id]; ok && regularFile(filepath.Join(bundleDir, "ocr", id+".txt")) {
			valid = append(valid, id)
		}
	}
	return sortedUnique(valid)
}

func validOCRIDs(bundleDir string, ids []string) bool {
	for _, id := range ids {
		if !regularFile(filepath.Join(bundleDir, "ocr", id+".txt")) {
			return false
		}
	}
	return true
}

func readMetadataDocument(path string) (MetadataDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return MetadataDocument{}, err
	}
	var document MetadataDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return MetadataDocument{}, err
	}
	return document, nil
}

func allRegularFiles(paths []string) bool {
	for _, path := range paths {
		if !regularFile(path) {
			return false
		}
	}
	return true
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func allNonEmptyRegularFiles(paths []string) bool {
	for _, path := range paths {
		if !nonEmptyRegularFile(path) {
			return false
		}
	}
	return true
}

func nonEmptyRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
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

func removeMatches(paths []string) error {
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func sortedUnique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func manifestOptionsEqual(left, right artifacts.ManifestOptions) bool {
	return left.FPS == right.FPS && left.WhisperLanguage == right.WhisperLanguage && left.WhisperModel == right.WhisperModel && equalStrings(left.OCRLanguages, right.OCRLanguages)
}

func workerCount(configured, jobs int) int {
	if jobs <= 0 {
		return 0
	}
	workers := configured
	if workers <= 0 {
		workers = runtime.NumCPU()
		if workers > 8 {
			workers = 8
		}
	}
	if workers > jobs {
		workers = jobs
	}
	return workers
}

func syncFile(path string) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := dir.Sync(); err != nil {
		_ = dir.Close()
		return err
	}
	return dir.Close()
}

func baseNames(paths []string) []string {
	result := make([]string, len(paths))
	for i := range paths {
		result[i] = filepath.Base(paths[i])
	}
	return result
}
