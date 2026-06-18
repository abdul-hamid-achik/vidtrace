package tesseract

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

func OCR(ctx context.Context, imagePath, outputBase, language string) error {
	image, err := os.Open(imagePath)
	if err != nil {
		return fmt.Errorf("open image for OCR: %w", err)
	}
	defer func() {
		_ = image.Close()
	}()

	cmd := exec.CommandContext(ctx, "tesseract", "stdin", outputBase, "-l", language)
	cmd.Stdin = image
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		output := strings.TrimSpace(stderr.String())
		if output == "" {
			output = strings.TrimSpace(stdout.String())
		}
		return fmt.Errorf("tesseract failed: %w: %s", err, output)
	}
	return nil
}

func SplitLanguages(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == '+' || r == ','
	})

	var languages []string
	seen := make(map[string]struct{})
	for _, field := range fields {
		language := strings.TrimSpace(field)
		if language == "" {
			continue
		}
		if _, ok := seen[language]; ok {
			continue
		}
		seen[language] = struct{}{}
		languages = append(languages, language)
	}
	sort.Strings(languages)
	return languages
}
