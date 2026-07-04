package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"som/internal/tui/api"

	tea "github.com/charmbracelet/bubbletea"
)

func searchCmd(c *api.Client, q string) tea.Cmd {
	return func() tea.Msg {
		tracks, err := c.Search(q)
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

func downloadCmd(c *api.Client, t api.Track, destDir string) tea.Cmd {
	return func() tea.Msg {
		path, err := c.DownloadM4A(t.ID, t.Title, destDir)

		if err == nil {
			lr, errLyr := c.Lyrics(t.ID, t.Title, t.Artist, t.Duration)
			if errLyr == nil {
				jsonPath := strings.TrimSuffix(path, ".m4a") + ".json"
				data, _ := json.MarshalIndent(lr, "", "  ")
				_ = os.WriteFile(jsonPath, data, 0644)
			}
		}
		return DownloadDoneMsg{Path: path, Err: err}
	}
}
