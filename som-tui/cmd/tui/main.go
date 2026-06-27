package main

import (
	"flag"
	"fmt"
	"os"

	"som-tui/internal/tui/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	server := flag.String("server", "http://localhost:8080", "SOM backend base URL")
	flag.Parse()

	app := ui.NewApp(*server)

	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
