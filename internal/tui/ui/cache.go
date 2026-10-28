package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"som/internal/domain"
	"som/internal/storage"
)

const manualLyricsOverridePrefix = "lyrics_manual_override:"

func manualLyricsOverrideKey(id string) string {
	return manualLyricsOverridePrefix + id
}

func getCachedLyrics(p domain.MusicProvider, store *storage.DB, id, title, artist string, duration int) (domain.LyricsResp, error) {
	if store == nil {
		return p.Lyrics(context.Background(), id, title, artist, duration)
	}
	if id != "" {
		if saved := store.GetSetting(manualLyricsOverrideKey(id)); saved != "" {
			var lr domain.LyricsResp
			if json.Unmarshal([]byte(saved), &lr) == nil &&
				(len(lr.AllTracks) > 0 || len(lr.Synced) > 0 || lr.Plain != "") {
				log.Printf("[lyrics] manual override hit id=%s title=%q", id, lr.Title)
				return lr, nil
			}
		}
	}
	cacheKey := fmt.Sprintf("%s|%s|%s", id, title, artist)

	cachedJSON, err := store.GetLyricsCache(cacheKey)
	if err == nil && cachedJSON != "" {
		var lr domain.LyricsResp
		if json.Unmarshal([]byte(cachedJSON), &lr) == nil &&
			(len(lr.AllTracks) > 0 || len(lr.Synced) > 0 || lr.Plain != "") {
			log.Printf("[lyrics] local cache hit id=%s title=%q tracks=%d", id, title, len(lr.AllTracks))
			return lr, nil
		}
	}

	lr, err := p.Lyrics(context.Background(), id, title, artist, duration)

	if err == nil && (len(lr.AllTracks) > 0 || len(lr.Synced) > 0 || lr.Plain != "") {
		if lrJSON, jsonErr := json.Marshal(lr); jsonErr == nil {
			_ = store.PutLyricsCache(cacheKey, string(lrJSON), false)
		}
	}

	return lr, err
}
