package ui

import (
	"context"
	"som/internal/domain"
	"sync"
)

var (
	lyricsCacheMu sync.RWMutex
	lyricsCache   = make(map[string]domain.LyricsResp)
)

func getCachedLyrics(p domain.MusicProvider, id, title, artist string, duration int) (domain.LyricsResp, error) {
	lyricsCacheMu.RLock()
	if lr, ok := lyricsCache[id]; ok {
		lyricsCacheMu.RUnlock()
		return lr, nil
	}
	lyricsCacheMu.RUnlock()

	lr, err := p.Lyrics(context.Background(), id, title, artist, duration)
	if err == nil {
		lyricsCacheMu.Lock()
		lyricsCache[id] = lr
		lyricsCacheMu.Unlock()
	}
	return lr, err
}
