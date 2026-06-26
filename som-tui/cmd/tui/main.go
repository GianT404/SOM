// cmd/tui/main.go
// SOM TUI – Stream music from YouTube in your terminal.
//
// Usage:
//
//	go run ./cmd/tui                     # connect to localhost:8080
//	go run ./cmd/tui -server http://host:8080
//
// Prerequisites:
//   - SOM backend running  (go run ./cmd/server)
//   - mpv installed        (brew install mpv / apt install mpv)
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/GianT404/SOM/tui/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	server := flag.String("server", "http://localhost:8080", "SOM backend URL")
	flag.Parse()

	app := ui.NewApp(*server)

	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),       // full-screen TUI
		tea.WithMouseCellMotion(), // mouse scroll in lists
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
