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

var Version = "dev"

func main() {
	upgrade := flag.Bool("upgrade", false, "download and install the latest SOM release from GitHub")
	install := flag.Bool("install", false, "copy this binary to /usr/local/bin (or platform equivalent) so `som` works from anywhere")
	showVersion := flag.Bool("version", false, "print the current version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("som", Version)
		return
	}
	if *install {
		if err := runInstall(); err != nil {
			fmt.Fprintln(os.Stderr, "Cài đặt thất bại:", err)
			os.Exit(1)
		}
		return
	}
	if *upgrade {
		if err := runSelfUpdate(Version); err != nil {
			fmt.Fprintln(os.Stderr, "Cập nhật thất bại:", err)
			os.Exit(1)
		}
		return
	}
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
