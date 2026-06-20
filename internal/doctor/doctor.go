package doctor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Result struct {
	OK                   bool         `json:"ok"`
	Tools                []ToolStatus `json:"tools"`
	OptionalTools        []ToolStatus `json:"optional_tools,omitempty"`
	TesseractLanguages   []string     `json:"tesseract_languages,omitempty"`
	WhisperCachedModels  []string     `json:"whisper_cached_models,omitempty"`
	RecommendedNextSteps []string     `json:"recommended_next_steps,omitempty"`
}

type ToolStatus struct {
	Name    string `json:"name"`
	Found   bool   `json:"found"`
	Path    string `json:"path,omitempty"`
	Version string `json:"version,omitempty"`
}

func Check() Result {
	tools := []ToolStatus{
		checkTool("ffmpeg", "--version"),
		checkTool("ffprobe", "--version"),
		checkTool("tesseract", "--version"),
		checkTool("whisper", "--help"),
	}

	// Optional tools enable extra features but never fail the doctor check.
	optionalTools := []ToolStatus{
		checkTool("ollama", "--version"),
	}

	result := Result{
		OK:                  true,
		Tools:               tools,
		OptionalTools:       optionalTools,
		TesseractLanguages:  tesseractLanguages(),
		WhisperCachedModels: whisperCachedModels(),
	}

	for _, tool := range tools {
		if !tool.Found {
			result.OK = false
			result.RecommendedNextSteps = append(result.RecommendedNextSteps, fmt.Sprintf("Install %s and make sure it is on PATH.", tool.Name))
		}
	}

	if !optionalTools[0].Found {
		result.RecommendedNextSteps = append(result.RecommendedNextSteps, "Optional: install Ollama for semantic and hybrid evidence search (vidtrace index/search --embed ollama).")
	}

	if !contains(result.TesseractLanguages, "eng") {
		result.OK = false
		result.RecommendedNextSteps = append(result.RecommendedNextSteps, "Install the Tesseract English language data package.")
	}
	if !contains(result.TesseractLanguages, "spa") {
		result.RecommendedNextSteps = append(result.RecommendedNextSteps, "Install Tesseract Spanish language data before enabling eng+spa OCR.")
	}
	if !contains(result.WhisperCachedModels, "small.pt") {
		result.RecommendedNextSteps = append(result.RecommendedNextSteps, "Run a first transcription with Whisper small, or prefetch ~/.cache/whisper/small.pt.")
	}

	return result
}

func PrintHuman(w io.Writer, result Result) {
	if result.OK {
		writeLine(w, "vidtrace doctor: ok")
	} else {
		writeLine(w, "vidtrace doctor: missing requirements")
	}

	writeLine(w, "\nTools:")
	for _, tool := range result.Tools {
		status := "missing"
		if tool.Found {
			status = "found"
		}
		if tool.Version != "" {
			writeLine(w, "  - %s: %s (%s) [%s]", tool.Name, status, tool.Path, tool.Version)
		} else if tool.Path != "" {
			writeLine(w, "  - %s: %s (%s)", tool.Name, status, tool.Path)
		} else {
			writeLine(w, "  - %s: %s", tool.Name, status)
		}
	}

	if len(result.OptionalTools) > 0 {
		writeLine(w, "\nOptional tools:")
		for _, tool := range result.OptionalTools {
			status := "missing"
			if tool.Found {
				status = "found"
			}
			if tool.Path != "" {
				writeLine(w, "  - %s: %s (%s)", tool.Name, status, tool.Path)
			} else {
				writeLine(w, "  - %s: %s", tool.Name, status)
			}
		}
	}

	if len(result.TesseractLanguages) > 0 {
		writeLine(w, "\nTesseract languages: %s", strings.Join(result.TesseractLanguages, ", "))
	}
	if len(result.WhisperCachedModels) > 0 {
		writeLine(w, "Whisper cached models: %s", strings.Join(result.WhisperCachedModels, ", "))
	}
	if len(result.RecommendedNextSteps) > 0 {
		writeLine(w, "\nRecommended next steps:")
		for _, step := range result.RecommendedNextSteps {
			writeLine(w, "  - %s", step)
		}
	}
}

func writeLine(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format+"\n", args...)
}

func checkTool(name string, versionArg string) ToolStatus {
	path, err := exec.LookPath(name)
	if err != nil {
		return ToolStatus{Name: name}
	}

	if name == "whisper" {
		return ToolStatus{
			Name:    name,
			Found:   true,
			Path:    path,
			Version: "available",
		}
	}

	return ToolStatus{
		Name:    name,
		Found:   true,
		Path:    path,
		Version: firstLine(runCommand(name, versionArg)),
	}
}

func runCommand(name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return stderr.String()
	}
	if stdout.Len() > 0 {
		return stdout.String()
	}
	return stderr.String()
}

func firstLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func tesseractLanguages() []string {
	output := runCommand("tesseract", "--list-langs")
	var languages []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of available languages") {
			continue
		}
		languages = append(languages, line)
	}
	sort.Strings(languages)
	return languages
}

func whisperCachedModels() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	entries, err := os.ReadDir(filepath.Join(home, ".cache", "whisper"))
	if err != nil {
		return nil
	}

	var models []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".pt") {
			models = append(models, name)
		}
	}
	sort.Strings(models)
	return models
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
