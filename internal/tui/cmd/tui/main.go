package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"som/internal/tui/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	server := flag.String("server", "http://localhost:8080", "SOM backend base URL")
	flag.Parse()
	middleware.DefaultLogger = middleware.RequestLogger(
		&middleware.DefaultLogFormatter{
			Logger:  log.New(io.Discard, "", 0),
			NoColor: true,
		},
	)
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
