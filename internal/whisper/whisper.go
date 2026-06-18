package whisper

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func Transcribe(ctx context.Context, videoPath, outputDir, model, language string) error {
	args := []string{
		videoPath,
		"--model", model,
		"--output_dir", outputDir,
		"--output_format", "all",
		"--verbose", "False",
		"--fp16", "False",
	}
	if strings.TrimSpace(language) != "" {
		args = append(args, "--language", language)
	}

	cmd := exec.CommandContext(ctx, "whisper", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		output := strings.TrimSpace(stderr.String())
		if output == "" {
			output = strings.TrimSpace(stdout.String())
		}
		return fmt.Errorf("whisper failed: %w: %s", err, output)
	}
	return nil
}

func TranscriptFiles(outputDir string) ([]string, error) {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, filepath.Join(outputDir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}

func JSONPath(outputDir, sourceVideo string) string {
	base := filepath.Base(sourceVideo)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(outputDir, base+".json")
}
