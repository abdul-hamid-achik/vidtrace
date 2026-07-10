package artifacts

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestManifestFingerprintIsDeterministic(t *testing.T) {
	t.Parallel()
	sourcePath := filepath.Join(t.TempDir(), "bug.mp4")
	if err := os.WriteFile(sourcePath, []byte("video bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	options := NormalizeManifestOptions(1.5, []string{"spa", "eng", "eng"}, " en ", " small ")
	source1, fingerprint1, err := FingerprintSource(sourcePath, options)
	if err != nil {
		t.Fatal(err)
	}
	source2, fingerprint2, err := FingerprintSource(sourcePath, options)
	if err != nil {
		t.Fatal(err)
	}
	if source1 != source2 || fingerprint1 != fingerprint2 {
		t.Fatalf("fingerprint changed: %#v/%s != %#v/%s", source1, fingerprint1, source2, fingerprint2)
	}
	if got := options.OCRLanguages; len(got) != 2 || got[0] != "eng" || got[1] != "spa" {
		t.Fatalf("languages not normalized: %v", got)
	}
}

func TestManifestStoreAtomicConcurrentUpdates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "bug.mp4")
	if err := os.WriteFile(sourcePath, []byte("video bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	options := NormalizeManifestOptions(1, []string{"eng"}, "en", "small")
	source, fingerprint, err := FingerprintSource(sourcePath, options)
	if err != nil {
		t.Fatal(err)
	}
	stamp := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	path := filepath.Join(dir, StageManifestName)
	store := NewManifestStore(path, func() time.Time { return stamp.Add(time.Minute) })
	if err := store.Initialize(NewStageManifest(source, options, fingerprint, stamp)); err != nil {
		t.Fatal(err)
	}

	const updates = 24
	var wg sync.WaitGroup
	wg.Add(updates)
	for range updates {
		go func() {
			defer wg.Done()
			_, updateErr := store.Update(func(manifest *StageManifest) error {
				stage := manifest.Stages[StageMetadata]
				stage.ArtifactCount++
				manifest.Stages[StageMetadata] = stage
				return nil
			})
			if updateErr != nil {
				t.Errorf("Update() failed: %v", updateErr)
			}
		}()
	}
	wg.Wait()

	manifest, err := LoadStageManifest(path)
	if err != nil {
		t.Fatalf("atomic manifest did not parse: %v", err)
	}
	if got := manifest.Stages[StageMetadata].ArtifactCount; got != updates {
		t.Fatalf("lost concurrent updates: got %d want %d", got, updates)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".stage_manifest-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary manifests remain after atomic writes: %v", matches)
	}
}

func TestLoadStageManifestRejectsTamperedFingerprint(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "bug.mp4")
	if err := os.WriteFile(sourcePath, []byte("video bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	options := NormalizeManifestOptions(1, []string{"eng"}, "en", "small")
	source, fingerprint, err := FingerprintSource(sourcePath, options)
	if err != nil {
		t.Fatal(err)
	}
	manifest := NewStageManifest(source, options, fingerprint, time.Now())
	manifest.Fingerprint = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	path := filepath.Join(dir, StageManifestName)
	data := []byte(`{"schema_version":"1"}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteStageManifestAtomic(path, manifest); err == nil {
		t.Fatal("WriteStageManifestAtomic accepted a mismatched fingerprint")
	}
}

func TestBundleLockRejectsConcurrentResume(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	first, err := LockBundle(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = first.Close() }()

	if _, err := LockBundle(dir); err == nil {
		t.Fatal("second bundle lock unexpectedly succeeded")
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := LockBundle(dir)
	if err != nil {
		t.Fatalf("bundle lock was not released: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
}
