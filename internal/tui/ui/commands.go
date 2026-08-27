package ui

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"som/internal/domain"
	"som/internal/scraper"
	"time"

	tea "charm.land/bubbletea/v2"
)

func searchCmd(p domain.MusicProvider, q string) tea.Cmd {
	return func() tea.Msg {
		tracks, err := p.Search(context.Background(), q)
		return SearchResultMsg{Tracks: tracks, Err: err}
	}
}

const suggestDebounce = 200 * time.Millisecond

// suggestDebounceCmd chờ 200ms trước khi bắn SuggestDebounceMsg
func suggestDebounceCmd(query string) tea.Cmd {
	return tea.Tick(suggestDebounce, func(t time.Time) tea.Msg {
		return SuggestDebounceMsg{Query: query}
	})
}

func suggestCmd(query string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		items, err := scraper.Suggest(ctx, query)
		return SuggestionsMsg{Query: query, Items: items, Err: err}
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
				lr.Thumbnail = t.Thumbnail
			}
			jsonPath := localFileSidecar(path)
			data, _ := json.MarshalIndent(lr, "", "  ")
			_ = os.WriteFile(jsonPath, data, 0644)
		}
		return DownloadDoneMsg{Path: path, Err: err}
	}
}
