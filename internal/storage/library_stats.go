package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Thống kê số track trong playlist.
type PlaylistLibraryStats struct {
	Name       string
	TrackCount int
}

// Thống kê thư viện nhạc.
type LibraryStats struct {
	DownloadedTracks   int
	ImportedTracks     int
	DownloadedBytes    int64
	ImportedBytes      int64
	PlaylistCount      int
	PlaylistTrackCount int
	Playlists          []PlaylistLibraryStats
}

// Thống kê file nhạc và playlist.
func (db *DB) GetLibraryStats() (LibraryStats, error) {
	var stats LibraryStats

	libraryDir, err := filepath.Abs(db.dir)
	if err != nil {
		return stats, fmt.Errorf("resolve download directory: %w", err)
	}

	entries, err := os.ReadDir(libraryDir)
	if err != nil {
		return stats, fmt.Errorf("read download directory: %w", err)
	}

	// Đếm track đã tải trong thư mục gốc.
	for _, entry := range entries {
		if entry.IsDir() || !IsSupportedAudio(entry.Name()) {
			continue
		}

		path := filepath.Join(libraryDir, entry.Name())
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}

		stats.DownloadedTracks++
		stats.DownloadedBytes += info.Size()
	}

	// Đếm track đã import bên ngoài thư mục tải.
	files, err := db.ListAllLocalFiles()
	if err != nil {
		return LibraryStats{}, fmt.Errorf("list imported tracks: %w", err)
	}

	for _, file := range files {
		if !IsSupportedAudio(file.Path) {
			continue
		}

		path, err := filepath.Abs(file.Path)
		if err != nil || isPathWithinDir(libraryDir, path) {
			continue
		}

		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}

		stats.ImportedTracks++
		stats.ImportedBytes += info.Size()
	}

	// Đếm playlist và số track trong từng playlist.
	playlists, err := db.LoadAllPlaylists()
	if err != nil {
		return LibraryStats{}, fmt.Errorf("load playlists: %w", err)
	}

	stats.PlaylistCount = len(playlists)
	for _, playlist := range playlists {
		count := len(playlist.Tracks)
		stats.PlaylistTrackCount += count
		stats.Playlists = append(stats.Playlists, PlaylistLibraryStats{
			Name:       playlist.Name,
			TrackCount: count,
		})
	}

	return stats, nil
}

// Kiểm tra đường dẫn có nằm trong thư mục không.
func isPathWithinDir(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) {
		return false
	}

	return rel != "" && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
