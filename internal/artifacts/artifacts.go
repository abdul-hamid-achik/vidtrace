package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var unsafeNameChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

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

func EnsureBundleDirs(bundleDir string) error {
	for _, dir := range []string{"frames", "ocr", "transcript"} {
		if err := os.MkdirAll(filepath.Join(bundleDir, dir), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func WriteJSON(path string, value any) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func RelSlash(baseDir, path string) string {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
