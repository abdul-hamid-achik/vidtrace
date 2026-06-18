package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type model struct {
	spinner spinner.Model
	width   int
	height  int
}

func Run() error {
	_, err := tea.NewProgram(initialModel()).Run()
	return err
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	return model{spinner: s}
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
		Render("vidtrace")

	subtitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Render("Bug video evidence extraction for agents")

	body := strings.Join([]string{
		fmt.Sprintf("Status: %s TUI shell ready", m.spinner.View()),
		"",
		"Planned panels:",
		"  - Artifact browser",
		"  - Timeline viewer",
		"  - Transcript and OCR inspector",
		"  - Pipeline run monitor",
		"",
		"Press q to quit.",
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
