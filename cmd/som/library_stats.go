package main

import (
	"fmt"
	"log"
	"os"

	"som/internal/storage"
)

func runLibraryStats(downloadDir string) error {
	if downloadDir == "" {
		if env := os.Getenv("SOM_DOWNLOAD_DIR"); env != "" {
			downloadDir = env
		} else {
			downloadDir = storage.DefaultDir()
		}
	}

	store, err := storage.Open(downloadDir)
	if err != nil {
		return fmt.Errorf("open library database: %w", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			log.Printf("[storage] close warning: %v", err)
		}
	}()

	stats, err := store.GetLibraryStats()
	if err != nil {
		return err
	}

	fmt.Printf("Total tracks    : %d (%s)\n", stats.DownloadedTracks, formatBytes(stats.DownloadedBytes))

	if len(stats.Playlists) > 0 {
		fmt.Printf("Playlists       : %d\n", stats.PlaylistCount)
		for _, playlist := range stats.Playlists {
			fmt.Printf("  - %s: %d tracks\n", playlist.Name, playlist.TrackCount)
		}
	}
	return nil
}

func formatBytes(n int64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB"}
	value := float64(n)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", n, units[unit])
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}
