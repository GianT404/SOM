package ui

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"som/internal/domain"
	"som/internal/scraper"
	"strings"
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
			if t.Thumbnail != "" {
				safe := t.Title
				for _, ch := range []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"} {
					safe = strings.ReplaceAll(safe, ch, "-")
				}
				safe = strings.TrimSpace(safe)
				if safe == "" {
					safe = t.ID
				}
				imgPath := filepath.Join(destDir, safe+".jpg")
				if resp, errImg := http.Get(t.Thumbnail); errImg == nil {
					defer resp.Body.Close()
					if f, errF := os.Create(imgPath); errF == nil {
						_, _ = io.Copy(f, resp.Body)
						f.Close()
					}
				}
			}

			return DownloadDoneMsg{Path: path, Err: err, Track: t}
		}
		return DownloadDoneMsg{Path: path, Err: err}
	}
}
