package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type appConfig struct {
	Title string `toml:"title"`
}

type model struct {
	spinner spinner.Model
	cwd     string
	title   string
}

func initialModel(cwd string) model {
	cfg := appConfig{Title: "Lattice"}
	_, _ = toml.Decode("title = \"Lattice\"", &cfg)

	s := spinner.New()
	s.Spinner = spinner.Dot
	return model{
		spinner: s,
		cwd:     cwd,
		title:   cfg.Title,
	}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m model) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(fmt.Sprintf("%s %s", m.spinner.View(), m.title)),
		bodyStyle.Render("Hello from the Bubble Tea placeholder app."),
		bodyStyle.Render(fmt.Sprintf("Working directory: %s", m.cwd)),
		helpStyle.Render("Press q, esc, or ctrl+c to quit."),
	) + "\n"
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "unknown"
	}

	p := tea.NewProgram(initialModel(cwd), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running app: %v\n", err)
		os.Exit(1)
	}
}
