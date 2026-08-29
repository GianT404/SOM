package main

import (
	"fmt"
	"log"
	"os"

	"som/internal/domain"
	"som/internal/local"
	"som/internal/scraper"
	"som/internal/storage"
	"som/internal/tui/api"
	"som/internal/tui/ui"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

func main() {
	var serverURL string
	var downloadDir string

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

			if downloadDir == "" {
				if env := os.Getenv("SOM_DOWNLOAD_DIR"); env != "" {
					downloadDir = env
				} else {
					downloadDir = storage.DefaultDir()
				}
			}
			if err := storage.MigrateFromLegacy(downloadDir); err != nil {
				log.Printf("[storage] migration warning: %v", err)
			}

			app := ui.NewApp(provider, downloadDir)
			p := tea.NewProgram(app)
			if _, err := p.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVar(&serverURL, "server", "", "SOM backend base URL")
	rootCmd.Flags().StringVar(&downloadDir, "download-dir", "", "Directory to store downloaded tracks (default: ~/.local/share/som)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
