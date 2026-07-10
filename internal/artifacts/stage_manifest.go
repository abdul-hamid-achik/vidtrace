package artifacts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

const (
	StageManifestVersion  = "1"
	StageManifestName     = "stage_manifest.json"
	StageManifestLockName = ".stage_manifest.lock"

	StageMetadata    = "metadata"
	StageFrames      = "frames"
	StageOCR         = "ocr"
	StageTranscript  = "transcript"
	StageCombinedOCR = "combined_ocr"
	StageTimeline    = "timeline"
	StageReadme      = "readme"

	StagePending  = "pending"
	StageRunning  = "running"
	StageComplete = "complete"
	StageFailed   = "failed"
)

var StageOrder = []string{
	StageMetadata,
	StageFrames,
	StageOCR,
	StageTranscript,
	StageCombinedOCR,
	StageTimeline,
	StageReadme,
}

type StageManifest struct {
	SchemaVersion string                   `json:"schema_version"`
	Source        ManifestSource           `json:"source"`
	Options       ManifestOptions          `json:"options"`
	Fingerprint   string                   `json:"fingerprint"`
	CreatedAt     time.Time                `json:"created_at"`
	UpdatedAt     time.Time                `json:"updated_at"`
	Stages        map[string]ManifestStage `json:"stages"`
}

type ManifestSource struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256    string `json:"sha256"`
}

type ManifestOptions struct {
	FPS             float64  `json:"fps"`
	OCRLanguages    []string `json:"ocr_languages"`
	WhisperLanguage string   `json:"whisper_language,omitempty"`
	WhisperModel    string   `json:"whisper_model"`
}

type ManifestStage struct {
	Status            string   `json:"status"`
	ArtifactCount     int      `json:"artifact_count"`
	TotalFrames       int      `json:"total_frames,omitempty"`
	CompletedFrameIDs []string `json:"completed_frame_ids,omitempty"`
	LastError         string   `json:"last_error,omitempty"`
}

func NormalizeManifestOptions(fps float64, ocrLanguages []string, whisperLanguage, whisperModel string) ManifestOptions {
	seen := make(map[string]struct{}, len(ocrLanguages))
	normalizedLanguages := make([]string, 0, len(ocrLanguages))
	for _, language := range ocrLanguages {
		language = strings.TrimSpace(language)
		if language == "" {
			continue
		}
		if _, ok := seen[language]; ok {
			continue
		}
		seen[language] = struct{}{}
		normalizedLanguages = append(normalizedLanguages, language)
	}
	sort.Strings(normalizedLanguages)
	return ManifestOptions{
		FPS:             fps,
		OCRLanguages:    normalizedLanguages,
		WhisperLanguage: strings.TrimSpace(whisperLanguage),
		WhisperModel:    strings.TrimSpace(whisperModel),
	}
}

func FingerprintSource(path string, options ManifestOptions) (ManifestSource, string, error) {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return ManifestSource{}, "", fmt.Errorf("resolve source video: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return ManifestSource{}, "", fmt.Errorf("source video not found: %s", resolved)
	}
	if info.IsDir() {
		return ManifestSource{}, "", fmt.Errorf("source video is a directory: %s", resolved)
	}
	file, err := os.Open(resolved)
	if err != nil {
		return ManifestSource{}, "", fmt.Errorf("open source video: %w", err)
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if copyErr != nil {
		return ManifestSource{}, "", fmt.Errorf("hash source video: %w", copyErr)
	}
	if closeErr != nil {
		return ManifestSource{}, "", fmt.Errorf("close source video: %w", closeErr)
	}
	source := ManifestSource{
		Path:      filepath.Clean(resolved),
		SizeBytes: info.Size(),
		SHA256:    hex.EncodeToString(hash.Sum(nil)),
	}
	fingerprint, err := CompositeFingerprint(source, options)
	if err != nil {
		return ManifestSource{}, "", err
	}
	return source, fingerprint, nil
}

func CompositeFingerprint(source ManifestSource, options ManifestOptions) (string, error) {
	payload := struct {
		Source  ManifestSource  `json:"source"`
		Options ManifestOptions `json:"options"`
	}{Source: source, Options: options}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode extraction fingerprint: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func NewStageManifest(source ManifestSource, options ManifestOptions, fingerprint string, now time.Time) StageManifest {
	stages := make(map[string]ManifestStage, len(StageOrder))
	for _, stage := range StageOrder {
		stages[stage] = ManifestStage{Status: StagePending}
	}
	now = now.UTC()
	return StageManifest{
		SchemaVersion: StageManifestVersion,
		Source:        source,
		Options:       options,
		Fingerprint:   fingerprint,
		CreatedAt:     now,
		UpdatedAt:     now,
		Stages:        stages,
	}
}

func LoadStageManifest(path string) (StageManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return StageManifest{}, fmt.Errorf("read %s: %w", filepath.Base(path), err)
	}
	var manifest StageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return StageManifest{}, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	if err := manifest.Validate(); err != nil {
		return StageManifest{}, fmt.Errorf("validate %s: %w", filepath.Base(path), err)
	}
	return manifest, nil
}

func (m StageManifest) Validate() error {
	if m.SchemaVersion != StageManifestVersion {
		return fmt.Errorf("unsupported schema_version %q (want %s)", m.SchemaVersion, StageManifestVersion)
	}
	if !filepath.IsAbs(m.Source.Path) || strings.TrimSpace(m.Source.Path) == "" {
		return fmt.Errorf("source.path must be absolute")
	}
	if m.Source.SizeBytes < 0 {
		return fmt.Errorf("source.size_bytes must be non-negative")
	}
	if !validSHA256(m.Source.SHA256) {
		return fmt.Errorf("source.sha256 must be 64 lowercase hexadecimal characters")
	}
	if m.Options.FPS <= 0 {
		return fmt.Errorf("options.fps must be greater than 0")
	}
	if len(m.Options.OCRLanguages) == 0 {
		return fmt.Errorf("options.ocr_languages must not be empty")
	}
	normalized := NormalizeManifestOptions(m.Options.FPS, m.Options.OCRLanguages, m.Options.WhisperLanguage, m.Options.WhisperModel)
	if strings.Join(normalized.OCRLanguages, "\x00") != strings.Join(m.Options.OCRLanguages, "\x00") || normalized.WhisperLanguage != m.Options.WhisperLanguage || normalized.WhisperModel != m.Options.WhisperModel {
		return fmt.Errorf("options are not normalized")
	}
	if normalized.WhisperModel == "" {
		return fmt.Errorf("options.whisper_model is required")
	}
	wantFingerprint, err := CompositeFingerprint(m.Source, m.Options)
	if err != nil {
		return err
	}
	if m.Fingerprint != wantFingerprint {
		return fmt.Errorf("fingerprint does not match source and options")
	}
	if m.CreatedAt.IsZero() || m.UpdatedAt.IsZero() {
		return fmt.Errorf("created_at and updated_at are required")
	}
	if m.UpdatedAt.Before(m.CreatedAt) {
		return fmt.Errorf("updated_at precedes created_at")
	}
	if len(m.Stages) != len(StageOrder) {
		return fmt.Errorf("stages must contain exactly %d entries", len(StageOrder))
	}
	for _, name := range StageOrder {
		stage, ok := m.Stages[name]
		if !ok {
			return fmt.Errorf("stage %q is missing", name)
		}
		if err := validateManifestStage(name, stage); err != nil {
			return err
		}
	}
	return nil
}

func (m StageManifest) Complete() bool {
	for _, name := range StageOrder {
		if m.Stages[name].Status != StageComplete {
			return false
		}
	}
	return true
}

func validateManifestStage(name string, stage ManifestStage) error {
	switch stage.Status {
	case StagePending, StageRunning, StageComplete, StageFailed:
	default:
		return fmt.Errorf("stage %q has invalid status %q", name, stage.Status)
	}
	if stage.ArtifactCount < 0 || stage.TotalFrames < 0 {
		return fmt.Errorf("stage %q has a negative artifact count", name)
	}
	previous := ""
	for _, id := range stage.CompletedFrameIDs {
		if strings.TrimSpace(id) == "" || (previous != "" && id <= previous) {
			return fmt.Errorf("stage %q completed_frame_ids must be sorted and unique", name)
		}
		previous = id
	}
	if name != StageOCR && (stage.TotalFrames != 0 || len(stage.CompletedFrameIDs) != 0) {
		return fmt.Errorf("only the OCR stage may contain per-frame completion")
	}
	if name == StageOCR {
		if stage.ArtifactCount != len(stage.CompletedFrameIDs) {
			return fmt.Errorf("OCR artifact_count must equal completed_frame_ids length")
		}
		if len(stage.CompletedFrameIDs) > stage.TotalFrames {
			return fmt.Errorf("OCR completed frames exceed total_frames")
		}
		if stage.Status == StageComplete && stage.ArtifactCount != stage.TotalFrames {
			return fmt.Errorf("complete OCR stage must contain every frame")
		}
	}
	return nil
}

func validSHA256(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

type BundleLock struct {
	file *os.File
}

func LockBundle(bundleDir string) (*BundleLock, error) {
	path := filepath.Join(bundleDir, StageManifestLockName)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open bundle lock: %w", err)
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			return nil, fmt.Errorf("artifact bundle is already being extracted or resumed: %s", bundleDir)
		}
		return nil, fmt.Errorf("lock artifact bundle: %w", err)
	}
	return &BundleLock{file: file}, nil
}

func (l *BundleLock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	unlockErr := unix.Flock(int(l.file.Fd()), unix.LOCK_UN)
	closeErr := l.file.Close()
	l.file = nil
	if unlockErr != nil {
		return fmt.Errorf("unlock artifact bundle: %w", unlockErr)
	}
	return closeErr
}

var manifestLocks sync.Map

type ManifestStore struct {
	path string
	now  func() time.Time
	mu   *sync.Mutex
}

func NewManifestStore(path string, now func() time.Time) *ManifestStore {
	if now == nil {
		now = time.Now
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		resolved = filepath.Clean(path)
	}
	lock, _ := manifestLocks.LoadOrStore(resolved, &sync.Mutex{})
	return &ManifestStore{path: resolved, now: now, mu: lock.(*sync.Mutex)}
}

func (s *ManifestStore) Path() string { return s.path }

func (s *ManifestStore) Initialize(manifest StageManifest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return WriteStageManifestAtomic(s.path, manifest)
}

func (s *ManifestStore) Load() (StageManifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return LoadStageManifest(s.path)
}

func (s *ManifestStore) Update(update func(*StageManifest) error) (StageManifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	manifest, err := LoadStageManifest(s.path)
	if err != nil {
		return StageManifest{}, err
	}
	if err := update(&manifest); err != nil {
		return StageManifest{}, err
	}
	manifest.UpdatedAt = s.now().UTC()
	if manifest.UpdatedAt.Before(manifest.CreatedAt) {
		manifest.UpdatedAt = manifest.CreatedAt
	}
	if err := WriteStageManifestAtomic(s.path, manifest); err != nil {
		return StageManifest{}, err
	}
	return manifest, nil
}

func WriteStageManifestAtomic(path string, manifest StageManifest) (err error) {
	if err := manifest.Validate(); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".stage_manifest-*.tmp")
	if err != nil {
		return fmt.Errorf("create stage manifest temp file: %w", err)
	}
	tempPath := temp.Name()
	defer func() {
		_ = temp.Close()
		if err != nil {
			_ = os.Remove(tempPath)
		}
	}()
	if err = temp.Chmod(0o644); err != nil {
		return fmt.Errorf("chmod stage manifest temp file: %w", err)
	}
	encoder := json.NewEncoder(temp)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(manifest); err != nil {
		return fmt.Errorf("encode stage manifest: %w", err)
	}
	if err = temp.Sync(); err != nil {
		return fmt.Errorf("sync stage manifest temp file: %w", err)
	}
	if err = temp.Close(); err != nil {
		return fmt.Errorf("close stage manifest temp file: %w", err)
	}
	if err = os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace stage manifest: %w", err)
	}
	directory, openErr := os.Open(dir)
	if openErr != nil {
		return fmt.Errorf("open stage manifest directory: %w", openErr)
	}
	if syncErr := directory.Sync(); syncErr != nil {
		_ = directory.Close()
		return fmt.Errorf("sync stage manifest directory: %w", syncErr)
	}
	if closeErr := directory.Close(); closeErr != nil {
		return fmt.Errorf("close stage manifest directory: %w", closeErr)
	}
	return nil
}
