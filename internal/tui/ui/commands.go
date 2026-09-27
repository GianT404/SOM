package ui

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"som/internal/domain"
	"som/internal/scraper"

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
				thumbCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				req, reqErr := http.NewRequestWithContext(thumbCtx, http.MethodGet, t.Thumbnail, nil)
				if reqErr == nil {
					resp, errImg := http.DefaultClient.Do(req)
					if errImg == nil {
						if resp.StatusCode >= 200 && resp.StatusCode < 300 {
							if f, errF := os.Create(imgPath); errF == nil {
								_, _ = io.Copy(f, resp.Body)
								_ = f.Close()
							}
						}
						_ = resp.Body.Close()
					}
				}
				cancel()
			}

			return DownloadDoneMsg{Path: path, Err: err, Track: t}
		}
		return DownloadDoneMsg{Path: path, Err: err}
	}
}
