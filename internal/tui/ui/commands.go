package ui

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"som/internal/domain"
	"som/internal/scraper"

	tea "charm.land/bubbletea/v2"
)

func searchCmd(p domain.MusicProvider, q string, offset int) tea.Cmd {
	return func() tea.Msg {
		tracks, err := p.Search(context.Background(), q, offset)
		return SearchResultMsg{Tracks: tracks, Query: q, Offset: offset, Err: err}
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
		startedAt := time.Now()
		log.Printf("[download] started id=%s title=%q", t.ID, t.Title)
		path, err := p.DownloadOPUS(context.Background(), t.ID, t.Title, destDir)
		if err != nil {
			log.Printf("[download] failed error=%v id=%s title=%q elapsed=%s", err, t.ID, t.Title, time.Since(startedAt).Round(time.Millisecond))
			return DownloadDoneMsg{Path: path, Err: err}
		}

		if info, statErr := os.Stat(path); statErr == nil {
			log.Printf("[download] audio saved id=%s path=%q bytes=%d elapsed=%s", t.ID, path, info.Size(), time.Since(startedAt).Round(time.Millisecond))
		} else {
			log.Printf("[download] audio saved id=%s path=%q elapsed=%s stat_error=%v", t.ID, path, time.Since(startedAt).Round(time.Millisecond), statErr)
		}

		if t.Thumbnail != "" {
			imgPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".jpg"
			thumbCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			req, reqErr := http.NewRequestWithContext(thumbCtx, http.MethodGet, t.Thumbnail, nil)
			if reqErr != nil {
				log.Printf("[download] thumbnail request failed id=%s error=%v", t.ID, reqErr)
			} else if resp, errImg := http.DefaultClient.Do(req); errImg != nil {
				log.Printf("[download] thumbnail download failed id=%s error=%v", t.ID, errImg)
			} else {
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					if f, errF := os.Create(imgPath); errF != nil {
						log.Printf("[download] thumbnail save failed id=%s error=%v", t.ID, errF)
					} else {
						n, copyErr := io.Copy(f, resp.Body)
						if copyErr != nil {
							log.Printf("[download] thumbnail copy failed id=%s error=%v", t.ID, copyErr)
						} else {
							log.Printf("[download] thumbnail saved id=%s path=%q bytes=%d", t.ID, imgPath, n)
						}
						_ = f.Close()
					}
				} else {
					log.Printf("[download] thumbnail request returned id=%s status=%s", t.ID, resp.Status)
				}
				_ = resp.Body.Close()
			}
			cancel()
		}

		return DownloadDoneMsg{Path: path, Track: t}
	}
}
