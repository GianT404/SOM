package main

import (
	"flag"
	"fmt"
	"os"

	"som/internal/domain"
	"som/internal/local"
	"som/internal/scraper"
	"som/internal/tui/api"
	"som/internal/tui/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	serverURL := flag.String("server", "", "SOM backend base URL (Để trống để chạy nội bộ/Local)")
	flag.Parse()

	var provider domain.MusicProvider

	if *serverURL == "" {
		ytdlpPath := os.Getenv("YTDLP_PATH")
		if ytdlpPath == "" {
			ytdlpPath = "yt-dlp"
		}
		sc := scraper.NewYtdlpScraper(ytdlpPath)
		provider = &local.DirectProvider{Scraper: sc}
	} else {
		// Bật Remote Mode
		provider = api.NewHTTPProvider(*serverURL)
	}

	app := ui.NewApp(provider)
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
