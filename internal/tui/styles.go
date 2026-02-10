package tui

import "github.com/charmbracelet/lipgloss"

const (
	colorPrimary   = lipgloss.Color("86")
	colorSecondary = lipgloss.Color("252")
	colorMuted     = lipgloss.Color("241")
	colorAccent    = lipgloss.Color("212")
	colorSuccess   = lipgloss.Color("42")
	colorDanger    = lipgloss.Color("203")
)

// Styles defines common visual primitives shared by TUI components.
type Styles struct {
	Header      lipgloss.Style
	Subheader   lipgloss.Style
	Body        lipgloss.Style
	Muted       lipgloss.Style
	ListItem    lipgloss.Style
	Selected    lipgloss.Style
	Help        lipgloss.Style
	Success     lipgloss.Style
	Error       lipgloss.Style
	FocusedMark lipgloss.Style
}

// DefaultStyles returns the baseline style set for the app TUI.
func DefaultStyles() Styles {
	return Styles{
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary),
		Subheader: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent),
		Body: lipgloss.NewStyle().
			Foreground(colorSecondary),
		Muted: lipgloss.NewStyle().
			Foreground(colorMuted),
		ListItem: lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(colorSecondary),
		Selected: lipgloss.NewStyle().
			PaddingLeft(1).
			Foreground(colorPrimary).
			Bold(true),
		Help: lipgloss.NewStyle().
			Foreground(colorMuted),
		Success: lipgloss.NewStyle().
			Foreground(colorSuccess),
		Error: lipgloss.NewStyle().
			Foreground(colorDanger),
		FocusedMark: lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true),
	}
}
