package ui

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"som/internal/domain"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func searchCmd(p domain.MusicProvider, q string) tea.Cmd {
	return func() tea.Msg {
		tracks, err := p.Search(context.Background(), q)
		return SearchResultMsg{Tracks: tracks, Err: err}
	}
}

func getDownloadDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, "Music", "SOM_Downloads"), nil
}

func downloadCmd(p domain.MusicProvider, t domain.Track, destDir string) tea.Cmd {
	return func() tea.Msg {
		path, err := p.DownloadOPUS(context.Background(), t.ID, t.Title, destDir)
		if err == nil {
			lr, errLyr := getCachedLyrics(p, t.ID, t.Title, t.Artist, t.Duration)
			if errLyr != nil {
				lr = domain.LyricsResp{}
				lr.Artist = t.Artist
				lr.Title = t.Title
				lr.VideoID = t.ID
			}
			jsonPath := strings.TrimSuffix(path, ".opus") + ".json"
			data, _ := json.MarshalIndent(lr, "", "  ")
			_ = os.WriteFile(jsonPath, data, 0644)
		}
		return DownloadDoneMsg{Path: path, Err: err}
	}
}
