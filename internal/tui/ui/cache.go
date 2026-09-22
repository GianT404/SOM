package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"som/internal/domain"
	"som/internal/storage"
)

func getCachedLyrics(p domain.MusicProvider, store *storage.DB, id, title, artist string, duration int) (domain.LyricsResp, error) {
	if store == nil {
		return p.Lyrics(context.Background(), id, title, artist, duration)
	}
	cacheKey := fmt.Sprintf("%s|%s|%s", id, title, artist)

	cachedJSON, err := store.GetLyricsCache(cacheKey)
	if err == nil && cachedJSON != "" {
		var lr domain.LyricsResp
		if json.Unmarshal([]byte(cachedJSON), &lr) == nil {
			return lr, nil
		}
	}

	lr, err := p.Lyrics(context.Background(), id, title, artist, duration)

	isEmpty := (err != nil) || (len(lr.AllTracks) == 0 && lr.Plain == "")

	if lrJSON, jsonErr := json.Marshal(lr); jsonErr == nil {
		_ = store.PutLyricsCache(cacheKey, string(lrJSON), isEmpty)
	}

	return lr, err
}
