package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-chi/chi/v5/middleware"

	"som/internal/tui/ui"
)

// Version is set at build time via:
//
//	go build -ldflags "-X main.Version=v0.4.0" ./cmd/som
//
// A plain `go build` (no ldflags) leaves it as "dev" — --upgrade refuses to
// run in that case since there's no real version to compare against GitHub.
var Version = "dev"

func main() {
	upgrade := flag.Bool("upgrade", false, "download and install the latest SOM release from GitHub")
	showVersion := flag.Bool("version", false, "print the current version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("som", Version)
		return
	}
	if *upgrade {
		if err := runSelfUpdate(Version); err != nil {
			fmt.Fprintln(os.Stderr, "Cập nhật thất bại:", err)
			os.Exit(1)
		}
		return
	}

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
