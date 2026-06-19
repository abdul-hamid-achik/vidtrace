package studio

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/abdul-hamid-achik/vidtrace/internal/bundle"
	"github.com/abdul-hamid-achik/vidtrace/internal/timeline"
)

type model struct {
	spinner   spinner.Model
	bundleDir string
	bundle    bundle.Bundle
	err       error
	cursor    int
	width     int
	height    int
}

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

	subtitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Render("Bug video evidence extraction for agents")

	body := strings.Join([]string{
		fmt.Sprintf("Status: %s %s", m.spinner.View(), m.statusLine()),
		"",
		m.bundleView(),
		"",
		"Keys: up/k down/j navigate, q quit.",
	}, "\n")

	content := lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "", body)
	if m.width <= 0 || m.height <= 0 {
		view := tea.NewView(content)
		view.AltScreen = true
		return view
	}

	view := tea.NewView(lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		fmt.Sprintf("%s\n", content),
	))
	view.AltScreen = true
	return view
}

func (m model) statusLine() string {
	if m.err != nil {
		return "bundle error"
	}
	if m.bundleDir == "" {
		return "no bundle loaded"
	}
	return "bundle loaded"
}

func (m model) bundleView() string {
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

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.timelineList(),
		"  ",
		m.entryDetail(),
	)
}

func (m model) timelineList() string {
	entries := m.bundle.Timeline.Entries
	start := 0
	if m.cursor > 6 {
		start = m.cursor - 6
	}
	end := min(start+12, len(entries))

	lines := []string{
		fmt.Sprintf("Bundle: %s", shortPath(m.bundle.Dir, 54)),
		fmt.Sprintf("Source: %s", shortPath(m.bundle.Metadata.SourceVideo, 54)),
		fmt.Sprintf("Duration: %.2fs  Entries: %d", m.bundle.Metadata.DurationSeconds, len(entries)),
		"",
		"Timeline",
	}
	for i := start; i < end; i++ {
		entry := entries[i]
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		text := firstNonEmpty(entry.OCR.Text, transcriptLine(entry))
		lines = append(lines, fmt.Sprintf("%s%6.2fs  %s", prefix, entry.TimeSeconds, truncate(text, 48)))
	}
	return strings.Join(lines, "\n")
}

func (m model) entryDetail() string {
	entry := m.bundle.Timeline.Entries[m.cursor]
	lines := []string{
		"Selected evidence",
		fmt.Sprintf("Time: %.2fs", entry.TimeSeconds),
		fmt.Sprintf("Frame: %s", entry.Frame),
		fmt.Sprintf("OCR: %s", entry.OCR.Path),
		"",
		"OCR text:",
		indentOrNone(entry.OCR.Text),
		"",
		"Transcript:",
		indentOrNone(transcriptLine(entry)),
	}
	return strings.Join(lines, "\n")
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

func indentOrNone(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "  none"
	}
	var lines []string
	for _, line := range strings.Split(value, "\n") {
		lines = append(lines, "  "+truncate(strings.TrimSpace(line), 68))
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
