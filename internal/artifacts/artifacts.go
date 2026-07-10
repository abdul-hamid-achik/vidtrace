package artifacts

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var unsafeNameChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// SchemaVersion is the artifact bundle schema version emitted by the pipeline
// and enforced by validation. Keep this as the single source of truth so the
// pipeline and the validator can never drift on the expected schema version.
const SchemaVersion = "1"

func SafeBundleName(sourceVideo string, explicitName string) string {
	name := strings.TrimSpace(explicitName)
	if name == "" {
		base := filepath.Base(sourceVideo)
		name = strings.TrimSuffix(base, filepath.Ext(base))
	}
	name = unsafeNameChars.ReplaceAllString(name, "_")
	name = strings.Trim(name, "._-")
	if name == "" {
		return "video"
	}
	return name
}

func BundlePath(parentDir, name string, now time.Time) string {
	return filepath.Join(parentDir, fmt.Sprintf("%s_artifacts_%s", name, now.Format("20060102_150405")))
}

// BundlePathUnique returns BundlePath, appending a numeric suffix (_2, _3, ...)
// if that directory already exists. This prevents two runs in the same second
// from silently overwriting each other's bundle.
func BundlePathUnique(parentDir, name string, now time.Time) string {
	candidate := BundlePath(parentDir, name, now)
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate
	}
	for suffix := 2; ; suffix++ {
		candidate = filepath.Join(parentDir, fmt.Sprintf("%s_artifacts_%s_%d", name, now.Format("20060102_150405"), suffix))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func EnsureBundleDirs(bundleDir string) error {
	for _, dir := range []string{"frames", "ocr", "transcript"} {
		if err := os.MkdirAll(filepath.Join(bundleDir, dir), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func WriteJSON(path string, value any) error {
	return WriteAtomic(path, func(w io.Writer) error {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	})
}

// WriteAtomic replaces one bundle artifact only after its complete contents
// are durable in a same-directory temporary file.
func WriteAtomic(path string, write func(io.Writer) error) (err error) {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer func() {
		_ = temp.Close()
		if err != nil {
			_ = os.Remove(tempPath)
		}
	}()
	if err = temp.Chmod(0o644); err != nil {
		return err
	}
	if err = write(temp); err != nil {
		return err
	}
	if err = temp.Sync(); err != nil {
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	if err = os.Rename(tempPath, path); err != nil {
		return err
	}
	directory, err := os.Open(dir)
	if err != nil {
		return err
	}
	if err = directory.Sync(); err != nil {
		_ = directory.Close()
		return err
	}
	return directory.Close()
}

func RelSlash(baseDir, path string) string {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
