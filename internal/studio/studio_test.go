package studio

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/abdul-hamid-achik/vidtrace/internal/bundle"
	"github.com/abdul-hamid-achik/vidtrace/internal/timeline"
)

func TestRunRejectsNonInteractiveTerminal(t *testing.T) {
	original := interactive
	t.Cleanup(func() { interactive = original })
	interactive = func() bool { return false }

	err := Run(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "interactive terminal") {
		t.Fatalf("expected an interactive-terminal error, got %v", err)
	}
}

func TestMetadataDetailIncludesReviewFields(t *testing.T) {
	m := model{bundle: bundle.Bundle{
		Metadata: bundle.Metadata{
			SourceVideo:     "/tmp/login-bug.mp4",
			GeneratedAt:     "2026-06-19T10:00:00Z",
			DurationSeconds: 12.5,
			Width:           1280,
			Height:          720,
			FrameRate:       30,
			ExtractFPS:      1,
			OCRLanguages:    []string{"eng", "spa"},
			WhisperLanguage: "en",
			WhisperModel:    "small",
		},
		Timeline: timeline.Document{Entries: []timeline.Entry{{Frame: "frames/frame_0001.png"}}},
	}}

	output := m.metadataDetail()
	for _, want := range []string{
		"Bundle metadata",
		"Source: /tmp/login-bug.mp4",
		"Duration: 12.50s",
		"Extract FPS: 1.00",
		"Dimensions: 1280x720",
		"OCR languages: eng+spa",
		"Whisper model: small",
		"Timeline entries: 1",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected metadata detail to contain %q, got %q", want, output)
		}
	}
}

func TestBundleViewUsesColumnsOnWideTerminal(t *testing.T) {
	m := studioFixtureModel()
	m.width = 120
	m.height = 32

	firstLine := strings.Split(m.bundleView(), "\n")[0]
	if !strings.Contains(firstLine, "Bundle:") || !strings.Contains(firstLine, "Selected evidence") {
		t.Fatalf("expected wide bundle view to use side-by-side columns, first line: %q", firstLine)
	}
}

func TestBundleViewStacksOnNarrowTerminal(t *testing.T) {
	m := studioFixtureModel()
	m.width = 72
	m.height = 32

	output := m.bundleView()
	lines := strings.Split(output, "\n")
	if len(lines) == 0 || strings.Contains(lines[0], "Selected evidence") || !strings.Contains(output, "Selected evidence") {
		t.Fatalf("expected narrow bundle view to stack timeline and detail panes, got %q", output)
	}
}

func TestViewUsesCompactTopAlignedShell(t *testing.T) {
	m := studioFixtureModel()
	m.width = 120
	m.height = 32

	view := m.View()
	if !view.AltScreen {
		t.Fatalf("expected Studio view to use alt screen")
	}
	if strings.HasPrefix(view.Content, "\n\n\n") {
		t.Fatalf("expected compact top-aligned view, got leading blank lines: %q", view.Content[:min(len(view.Content), 40)])
	}
	for _, want := range []string{"vidtrace studio", "status:", "keys:", "Selected evidence"} {
		if !strings.Contains(view.Content, want) {
			t.Fatalf("expected view to contain %q, got %q", want, view.Content)
		}
	}
}

func TestMetadataKeyTogglesDetailMode(t *testing.T) {
	m := model{}

	updated, _ := m.Update(tea.KeyPressMsg(tea.Key{Text: "m", Code: 'm'}))
	toggled := updated.(model)
	if !toggled.metadata {
		t.Fatalf("expected metadata mode after m key")
	}
	if toggled.action != "metadata shown" {
		t.Fatalf("unexpected action status: %q", toggled.action)
	}
}

func TestEvidenceSummaryFormatting(t *testing.T) {
	entry := timeline.Entry{
		TimeSeconds: 1.25,
		Frame:       "frames/frame_0002.png",
		OCR: timeline.OCR{
			Path: "ocr/frame_0002.txt",
			Text: "Login failed\nRetry button visible",
		},
		Transcript: []timeline.Segment{
			{StartSeconds: 1, EndSeconds: 2, Text: "I cannot log in"},
			{StartSeconds: 2, EndSeconds: 3, Text: "The retry button is visible"},
		},
	}

	want := strings.Join([]string{
		"Evidence 1.25s",
		"Frame: frames/frame_0002.png",
		"OCR: Login failed Retry button visible",
		"Transcript: I cannot log in The retry button is visible",
	}, "\n")
	if got := evidenceSummary(entry); got != want {
		t.Fatalf("unexpected evidence summary:\ngot  %q\nwant %q", got, want)
	}
}

func TestEvidenceSummaryUsesNoneForEmptyText(t *testing.T) {
	entry := timeline.Entry{
		TimeSeconds: 0,
		Frame:       "frames/frame_0001.png",
		OCR:         timeline.OCR{Path: "ocr/frame_0001.txt"},
	}

	output := evidenceSummary(entry)
	for _, want := range []string{"OCR: none", "Transcript: none"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected summary to contain %q, got %q", want, output)
		}
	}
}

func TestSelectedFramePathResolvesRelativeTimelinePath(t *testing.T) {
	dir := t.TempDir()
	m := model{
		cursor: 0,
		bundle: bundle.Bundle{
			Dir: dir,
			Timeline: timeline.Document{Entries: []timeline.Entry{{
				Frame: "frames/frame_0001.png",
			}}},
		},
	}

	got, ok := m.selectedFramePath()
	if !ok {
		t.Fatalf("expected selected frame path")
	}
	want := filepath.Join(dir, "frames", "frame_0001.png")
	if got != want {
		t.Fatalf("unexpected frame path: got %q want %q", got, want)
	}
}

func TestResolveBundlePathKeepsAbsolutePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "frame.png")

	if got := resolveBundlePath("/bundle", path); got != path {
		t.Fatalf("unexpected absolute path: got %q want %q", got, path)
	}
}

func TestOpenRevealAndClipboardCommandSelection(t *testing.T) {
	lookup := func(name string) (string, error) {
		return "/mock/" + name, nil
	}

	openCommand, err := openFrameCommand("darwin", "/tmp/frame.png", lookup)
	if err != nil {
		t.Fatalf("open command: %v", err)
	}
	if openCommand.path != "/mock/open" || strings.Join(openCommand.args, " ") != "/tmp/frame.png" {
		t.Fatalf("unexpected open command: %#v", openCommand)
	}

	revealCommand, err := revealFrameCommand("darwin", "/tmp/frame.png", lookup)
	if err != nil {
		t.Fatalf("reveal command: %v", err)
	}
	if revealCommand.path != "/mock/open" || strings.Join(revealCommand.args, " ") != "-R /tmp/frame.png" {
		t.Fatalf("unexpected reveal command: %#v", revealCommand)
	}

	clipboard, err := clipboardCommand("darwin", lookup)
	if err != nil {
		t.Fatalf("clipboard command: %v", err)
	}
	if clipboard.path != "/mock/pbcopy" || len(clipboard.args) != 0 {
		t.Fatalf("unexpected clipboard command: %#v", clipboard)
	}
}

func TestRevealCommandUnsupportedPlatform(t *testing.T) {
	_, err := revealFrameCommand("linux", "/tmp/frame.png", func(name string) (string, error) {
		return "/mock/" + name, nil
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported platform linux") {
		t.Fatalf("expected unsupported platform error, got %v", err)
	}
}

func TestClipboardCommandFallsBackOnLinux(t *testing.T) {
	lookup := func(name string) (string, error) {
		if name == "wl-copy" {
			return "", errors.New("missing")
		}
		return "/mock/" + name, nil
	}

	command, err := clipboardCommand("linux", lookup)
	if err != nil {
		t.Fatalf("clipboard command: %v", err)
	}
	if command.path != "/mock/xclip" || strings.Join(command.args, " ") != "-selection clipboard" {
		t.Fatalf("unexpected linux clipboard fallback: %#v", command)
	}
}

func studioFixtureModel() model {
	return model{
		bundleDir: "/tmp/sample_artifacts",
		bundle: bundle.Bundle{
			Dir: "/tmp/sample_artifacts",
			Metadata: bundle.Metadata{
				SourceVideo:     "/tmp/login-bug.mp4",
				DurationSeconds: 2,
				ExtractFPS:      1,
				OCRLanguages:    []string{"eng"},
				WhisperLanguage: "en",
				WhisperModel:    "small",
			},
			Timeline: timeline.Document{Entries: []timeline.Entry{
				{
					TimeSeconds: 0,
					Frame:       "frames/frame_0001.png",
					OCR: timeline.OCR{
						Path: "ocr/frame_0001.txt",
						Text: "Login failed after submit",
					},
					Transcript: []timeline.Segment{{StartSeconds: 0, EndSeconds: 1, Text: "I cannot log in"}},
				},
				{
					TimeSeconds: 1,
					Frame:       "frames/frame_0002.png",
					OCR: timeline.OCR{
						Path: "ocr/frame_0002.txt",
						Text: "Retry button visible",
					},
				},
			}},
		},
	}
}
