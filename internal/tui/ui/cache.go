package ui

import (
	"som/internal/tui/api"
	"sync"
)

var (
	lyricsCacheMu sync.RWMutex
	lyricsCache   = make(map[string]api.LyricsResp)
)

func getCachedLyrics(c *api.Client, id, title, artist string, duration int) (api.LyricsResp, error) {
	lyricsCacheMu.RLock()
	if lr, ok := lyricsCache[id]; ok {
		lyricsCacheMu.RUnlock()
		return lr, nil
	}
	lyricsCacheMu.RUnlock()

	lr, err := c.Lyrics(id, title, artist, duration)
	if err == nil {
		lyricsCacheMu.Lock()
		lyricsCache[id] = lr
		lyricsCacheMu.Unlock()
	}

	return lr, err
}
