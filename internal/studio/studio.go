package studio

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/abdul-hamid-achik/vidtrace/internal/bundle"
	"github.com/abdul-hamid-achik/vidtrace/internal/timeline"
)

const (
	defaultContentWidth  = 116
	defaultDetailWidth   = 68
	horizontalBreakpoint = 96
)

type model struct {
	spinner   spinner.Model
	bundleDir string
	bundle    bundle.Bundle
	err       error
	cursor    int
	width     int
	height    int
	metadata  bool
	action    string
}

type actionResultMsg struct {
	success string
	err     error
}

type externalCommand struct {
	path string
	args []string
}

type lookPathFunc func(string) (string, error)

func Run(bundleDir string) error {
	_, err := tea.NewProgram(initialModel(bundleDir)).Run()
	return err
}

func initialModel(bundleDir string) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))

	m := model{spinner: s, bundleDir: strings.TrimSpace(bundleDir)}
	if m.bundleDir != "" {
		doc, err := bundle.Load(m.bundleDir)
		if err != nil {
			m.err = err
		} else {
			m.bundle = doc
		}
	}
	return m
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "down", "j":
			if m.cursor < len(m.bundle.Timeline.Entries)-1 {
				m.cursor++
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "m":
			m.metadata = !m.metadata
			if m.metadata {
				m.action = "metadata shown"
			} else {
				m.action = "evidence shown"
			}
		case "o":
			return m.openSelectedFrame()
		case "r":
			return m.revealSelectedFrame()
		case "c":
			return m.copyEvidenceSummary()
		}
	case actionResultMsg:
		if msg.err != nil {
			m.action = msg.err.Error()
		} else {
			m.action = msg.success
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Render("vidtrace studio")

	summary := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Render(m.headerSummary())

	status := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Render(fmt.Sprintf("status: %s %s", m.spinner.View(), m.statusLine()))

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Render("keys: up/k down/j navigate | m metadata | o open | r reveal | c copy | q quit")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, title, "  ", summary),
		status,
		"",
		m.bundleView(),
		"",
		help,
	)
	if m.width > 0 {
		content = lipgloss.NewStyle().
			Width(max(1, m.width-4)).
			MaxWidth(max(1, m.width-4)).
			Padding(1, 2).
			Render(content)
	}

	view := tea.NewView(content)
	view.AltScreen = true
	return view
}

func (m model) headerSummary() string {
	if m.err != nil {
		return "bundle error"
	}
	if m.bundleDir == "" {
		return "no bundle loaded"
	}
	return fmt.Sprintf("%d entries | %.2fs | %s",
		len(m.bundle.Timeline.Entries),
		m.bundle.Metadata.DurationSeconds,
		shortPath(m.bundle.Metadata.SourceVideo, max(20, m.contentWidth()-28)),
	)
}

func (m model) statusLine() string {
	if m.err != nil {
		return "bundle error"
	}
	if m.bundleDir == "" {
		return "no bundle loaded"
	}
	status := "bundle loaded"
	if strings.TrimSpace(m.action) != "" {
		status += " - " + m.action
	}
	return status
}

func (m model) bundleView() string {
	width := m.contentWidth()
	if m.err != nil {
		return strings.Join([]string{
			"Bundle error:",
			"  " + m.err.Error(),
			"",
			"Run: vidtrace studio /path/to/bundle",
		}, "\n")
	}
	if m.bundleDir == "" {
		return strings.Join([]string{
			"No bundle loaded.",
			"",
			"Run: vidtrace studio /path/to/bundle",
			"Use vidtrace docs artifacts for bundle reading order.",
		}, "\n")
	}
	if len(m.bundle.Timeline.Entries) == 0 {
		return fmt.Sprintf("Bundle: %s\nTimeline entries: 0", m.bundle.Dir)
	}

	if width < horizontalBreakpoint {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.timelineList(width),
			"",
			m.detailView(width),
		)
	}

	timelineWidth := min(62, max(42, width/2))
	detailWidth := max(34, width-timelineWidth-2)
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Width(timelineWidth).MaxWidth(timelineWidth).Render(m.timelineList(timelineWidth)),
		"  ",
		lipgloss.NewStyle().Width(detailWidth).MaxWidth(detailWidth).Render(m.detailView(detailWidth)),
	)
}

func (m model) contentWidth() int {
	if m.width <= 0 {
		return defaultContentWidth
	}
	return max(40, m.width-4)
}

func (m model) detailView(width int) string {
	if m.metadata {
		return m.metadataDetailWidth(width)
	}
	return m.entryDetail(width)
}

func (m model) timelineList(width int) string {
	entries := m.bundle.Timeline.Entries
	rows := m.timelineRows()
	start := 0
	if m.cursor > rows/2 {
		start = m.cursor - rows/2
	}
	if start+rows > len(entries) {
		start = max(0, len(entries)-rows)
	}
	end := min(start+rows, len(entries))

	pathLimit := max(12, width-8)
	entryLimit := max(16, width-13)

	lines := []string{
		fmt.Sprintf("Bundle: %s", shortPath(m.bundle.Dir, pathLimit)),
		fmt.Sprintf("Source: %s", shortPath(m.bundle.Metadata.SourceVideo, pathLimit)),
		fmt.Sprintf("Duration: %.2fs  Entries: %d", m.bundle.Metadata.DurationSeconds, len(entries)),
		"",
		fmt.Sprintf("Timeline (%d-%d of %d)", start+1, end, len(entries)),
	}
	for i := start; i < end; i++ {
		entry := entries[i]
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		text := firstNonEmpty(entry.OCR.Text, transcriptLine(entry))
		lines = append(lines, fmt.Sprintf("%s%6.2fs  %s", prefix, entry.TimeSeconds, truncate(text, entryLimit)))
	}
	return strings.Join(lines, "\n")
}

func (m model) timelineRows() int {
	if m.height <= 0 {
		return 12
	}
	return min(16, max(6, m.height-15))
}

func (m model) entryDetail(width int) string {
	entry := m.bundle.Timeline.Entries[m.cursor]
	textLimit := max(20, width-2)
	lines := []string{
		"Selected evidence",
		fmt.Sprintf("Time: %.2fs", entry.TimeSeconds),
		fmt.Sprintf("Frame: %s", shortPath(entry.Frame, max(12, width-7))),
		fmt.Sprintf("OCR: %s", shortPath(entry.OCR.Path, max(12, width-5))),
		"",
		"OCR text:",
		indentOrNoneWidth(entry.OCR.Text, textLimit),
		"",
		"Transcript:",
		indentOrNoneWidth(transcriptLine(entry), textLimit),
	}
	return strings.Join(lines, "\n")
}

func (m model) metadataDetail() string {
	return m.metadataDetailWidth(defaultDetailWidth)
}

func (m model) metadataDetailWidth(width int) string {
	metadata := m.bundle.Metadata
	lines := []string{
		"Bundle metadata",
		fmt.Sprintf("Source: %s", shortPath(metadata.SourceVideo, max(12, width-8))),
		fmt.Sprintf("Generated: %s", emptyAsNone(metadata.GeneratedAt)),
		fmt.Sprintf("Duration: %.2fs", metadata.DurationSeconds),
		fmt.Sprintf("Frame rate: %.2f", metadata.FrameRate),
		fmt.Sprintf("Extract FPS: %.2f", metadata.ExtractFPS),
		fmt.Sprintf("Dimensions: %s", dimensions(metadata.Width, metadata.Height)),
		fmt.Sprintf("Video codec: %s", emptyAsNone(metadata.VideoCodec)),
		fmt.Sprintf("Audio codec: %s", emptyAsNone(metadata.AudioCodec)),
		fmt.Sprintf("OCR languages: %s", listOrNone(metadata.OCRLanguages)),
		fmt.Sprintf("Whisper language: %s", emptyAsNone(metadata.WhisperLanguage)),
		fmt.Sprintf("Whisper model: %s", emptyAsNone(metadata.WhisperModel)),
		fmt.Sprintf("Timeline entries: %d", len(m.bundle.Timeline.Entries)),
	}
	return strings.Join(lines, "\n")
}

func (m model) openSelectedFrame() (tea.Model, tea.Cmd) {
	framePath, ok := m.selectedFramePath()
	if !ok {
		m.action = "open failed: no evidence selected"
		return m, nil
	}
	if err := requireFile(framePath); err != nil {
		m.action = "open failed: " + err.Error()
		return m, nil
	}
	command, err := openFrameCommand(runtime.GOOS, framePath, exec.LookPath)
	if err != nil {
		m.action = "open unavailable: " + err.Error()
		return m, nil
	}
	m.action = "opening frame"
	return m, runCommand(command, "", "opened frame")
}

func (m model) revealSelectedFrame() (tea.Model, tea.Cmd) {
	framePath, ok := m.selectedFramePath()
	if !ok {
		m.action = "reveal failed: no evidence selected"
		return m, nil
	}
	if err := requireFile(framePath); err != nil {
		m.action = "reveal failed: " + err.Error()
		return m, nil
	}
	command, err := revealFrameCommand(runtime.GOOS, framePath, exec.LookPath)
	if err != nil {
		m.action = "reveal unavailable: " + err.Error()
		return m, nil
	}
	m.action = "revealing frame"
	return m, runCommand(command, "", "revealed frame")
}

func (m model) copyEvidenceSummary() (tea.Model, tea.Cmd) {
	entry, ok := m.selectedEntry()
	if !ok {
		m.action = "copy failed: no evidence selected"
		return m, nil
	}
	command, err := clipboardCommand(runtime.GOOS, exec.LookPath)
	if err != nil {
		m.action = "copy unavailable: " + err.Error()
		return m, nil
	}
	m.action = "copying evidence summary"
	return m, runCommand(command, evidenceSummary(entry), "copied evidence summary")
}

func (m model) selectedEntry() (timeline.Entry, bool) {
	if m.cursor < 0 || m.cursor >= len(m.bundle.Timeline.Entries) {
		return timeline.Entry{}, false
	}
	return m.bundle.Timeline.Entries[m.cursor], true
}

func (m model) selectedFramePath() (string, bool) {
	entry, ok := m.selectedEntry()
	if !ok {
		return "", false
	}
	return resolveBundlePath(m.bundle.Dir, entry.Frame), true
}

func resolveBundlePath(bundleDir, value string) string {
	path := filepath.FromSlash(strings.TrimSpace(value))
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(bundleDir, path))
}

func requireFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("frame not found: %s", filepath.ToSlash(path))
		}
		return fmt.Errorf("frame unavailable: %v", err)
	}
	if info.IsDir() {
		return fmt.Errorf("frame path is a directory: %s", filepath.ToSlash(path))
	}
	return nil
}

func evidenceSummary(entry timeline.Entry) string {
	lines := []string{
		fmt.Sprintf("Evidence %.2fs", entry.TimeSeconds),
		fmt.Sprintf("Frame: %s", entry.Frame),
		fmt.Sprintf("OCR: %s", oneLineOrNone(entry.OCR.Text)),
		fmt.Sprintf("Transcript: %s", oneLineOrNone(transcriptLine(entry))),
	}
	return strings.Join(lines, "\n")
}

func openFrameCommand(goos, path string, lookPath lookPathFunc) (externalCommand, error) {
	switch goos {
	case "darwin":
		return commandFromPath("open", []string{path}, lookPath)
	case "linux":
		return commandFromPath("xdg-open", []string{path}, lookPath)
	case "windows":
		return commandFromPath("cmd", []string{"/c", "start", "", path}, lookPath)
	default:
		return externalCommand{}, fmt.Errorf("unsupported platform %s", goos)
	}
}

func revealFrameCommand(goos, path string, lookPath lookPathFunc) (externalCommand, error) {
	switch goos {
	case "darwin":
		return commandFromPath("open", []string{"-R", path}, lookPath)
	case "windows":
		return commandFromPath("explorer", []string{"/select," + path}, lookPath)
	default:
		return externalCommand{}, fmt.Errorf("unsupported platform %s", goos)
	}
}

func clipboardCommand(goos string, lookPath lookPathFunc) (externalCommand, error) {
	switch goos {
	case "darwin":
		return commandFromPath("pbcopy", nil, lookPath)
	case "linux":
		if command, err := commandFromPath("wl-copy", nil, lookPath); err == nil {
			return command, nil
		}
		if command, err := commandFromPath("xclip", []string{"-selection", "clipboard"}, lookPath); err == nil {
			return command, nil
		}
		return commandFromPath("xsel", []string{"--clipboard", "--input"}, lookPath)
	case "windows":
		return commandFromPath("clip", nil, lookPath)
	default:
		return externalCommand{}, fmt.Errorf("unsupported platform %s", goos)
	}
}

func commandFromPath(name string, args []string, lookPath lookPathFunc) (externalCommand, error) {
	path, err := lookPath(name)
	if err != nil {
		return externalCommand{}, fmt.Errorf("%s not found", name)
	}
	return externalCommand{path: path, args: args}, nil
}

func runCommand(command externalCommand, input, success string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command(command.path, command.args...)
		if input != "" {
			cmd.Stdin = strings.NewReader(input)
		}
		if output, err := cmd.CombinedOutput(); err != nil {
			message := strings.TrimSpace(string(output))
			if message != "" {
				return actionResultMsg{err: fmt.Errorf("%s failed: %s", filepath.Base(command.path), truncate(message, 80))}
			}
			return actionResultMsg{err: fmt.Errorf("%s failed: %v", filepath.Base(command.path), err)}
		}
		return actionResultMsg{success: success}
	}
}

func transcriptLine(entry timeline.Entry) string {
	var parts []string
	for _, segment := range entry.Transcript {
		if text := strings.TrimSpace(segment.Text); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return "no text"
}

func emptyAsNone(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "none"
	}
	return value
}

func listOrNone(values []string) string {
	var parts []string
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, "+")
}

func dimensions(width, height int) string {
	if width <= 0 || height <= 0 {
		return "none"
	}
	return fmt.Sprintf("%dx%d", width, height)
}

func oneLineOrNone(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return "none"
	}
	return value
}

func indentOrNoneWidth(value string, limit int) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "  none"
	}
	limit = max(2, limit)
	var lines []string
	for _, line := range strings.Split(value, "\n") {
		lines = append(lines, "  "+truncate(strings.TrimSpace(line), limit))
	}
	return strings.Join(lines, "\n")
}

func truncate(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}

func shortPath(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return "..." + value[len(value)-limit+3:]
}
