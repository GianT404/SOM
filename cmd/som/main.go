package main

import (
	"fmt"
	"io"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-chi/chi/v5/middleware"

	"som/internal/tui/ui"
)

func main() {
	// Silence chi request logs to avoid noise in the Logs tab.
	middleware.DefaultLogger = middleware.RequestLogger(
		&middleware.DefaultLogFormatter{
			Logger:  log.New(io.Discard, "", 0),
			NoColor: true,
		},
	)

	app := ui.NewApp("")

	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Lỗi giao diện TUI: %v\n", err)
		os.Exit(1)
	}
}
