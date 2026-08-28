package main

import (
	"fmt"
	"os"

	"som/internal/domain"
	"som/internal/local"
	"som/internal/scraper"
	"som/internal/tui/api"
	"som/internal/tui/ui"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

func main() {
	var serverURL string

	var rootCmd = &cobra.Command{
		Use:   "som",
		Short: "SOM - Terminal Music Player",
		Run: func(cmd *cobra.Command, args []string) {
			var provider domain.MusicProvider
			if serverURL == "" {
				ytdlpPath := os.Getenv("YTDLP_PATH")
				if ytdlpPath == "" {
					ytdlpPath = "yt-dlp"
				}
				sc := scraper.NewYtdlpScraper(ytdlpPath)
				provider = &local.DirectProvider{Scraper: sc}
			} else {
				provider = api.NewHTTPProvider(serverURL)
			}
			app := ui.NewApp(provider)
			p := tea.NewProgram(app)
			if _, err := p.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVar(&serverURL, "server", "", "SOM backend base URL")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
