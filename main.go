package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"lattice/internal/config"
	"lattice/internal/tui"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "unknown"
	}

	if _, err := config.Init(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "error initializing lattice config: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(tui.NewApp(cwd), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running app: %v\n", err)
		os.Exit(1)
	}
}
