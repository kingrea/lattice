package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"lattice/internal/tui"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "unknown"
	}

	p := tea.NewProgram(tui.NewApp(cwd), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running app: %v\n", err)
		os.Exit(1)
	}
}
