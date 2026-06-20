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

// AvailableLanguages returns the OCR languages tesseract has data installed for.
func AvailableLanguages(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "tesseract", "--list-langs")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		out := strings.TrimSpace(stderr.String())
		if out == "" {
			out = strings.TrimSpace(stdout.String())
		}
		return nil, fmt.Errorf("list tesseract languages: %w: %s", err, out)
	}
	// Different tesseract versions print the list to stdout or stderr.
	combined := stdout.String()
	if strings.TrimSpace(combined) == "" {
		combined = stderr.String()
	}
	return parseLanguages(combined), nil
}

func parseLanguages(output string) []string {
	var languages []string
	seen := make(map[string]struct{})
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of available languages") {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		languages = append(languages, line)
	}
	sort.Strings(languages)
	return languages
}

// MissingLanguages returns the requested languages that are not installed,
// preserving request order and ignoring blanks and duplicates.
func MissingLanguages(requested, available []string) []string {
	have := make(map[string]struct{}, len(available))
	for _, a := range available {
		have[a] = struct{}{}
	}
	var missing []string
	seen := make(map[string]struct{})
	for _, r := range requested {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if _, ok := have[r]; ok {
			continue
		}
		if _, dup := seen[r]; dup {
			continue
		}
		seen[r] = struct{}{}
		missing = append(missing, r)
	}
	return missing
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
