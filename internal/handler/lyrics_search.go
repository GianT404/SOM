package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"som/internal/domain"
	"som/internal/scraper"
)

const maxLyricsSearchResults = 7

type LyricsSearchHandler struct{}

// ServeHTTP xử lý truy vấn lyrics thủ công qua LRCLib.
func (h *LyricsSearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeError(w, http.StatusBadRequest, "missing required parameter: q")
		return
	}
	if len(query) > maxQueryLen {
		writeError(w, http.StatusBadRequest, "q too long, max 100 characters")
		return
	}

	limit := maxLyricsSearchResults
	if raw := r.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > maxLyricsSearchResults {
			writeError(w, http.StatusBadRequest, "invalid limit, must be between 1 and 7")
			return
		}
		limit = value
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	results, err := scraper.SearchLRCLib(ctx, query, limit)
	if err != nil {
		var upstreamErr *scraper.LRCLibHTTPError
		if errors.As(err, &upstreamErr) &&
			(upstreamErr.StatusCode == http.StatusTooManyRequests || upstreamErr.StatusCode >= 500) {
			retryAfter := upstreamErr.RetryAfter
			if retryAfter == "" {
				retryAfter = "2"
			}
			w.Header().Set("Retry-After", retryAfter)
			writeError(w, http.StatusServiceUnavailable, "LRCLib is temporarily unavailable; retry later")
			return
		}
		writeError(w, http.StatusBadGateway, "LRCLib search unavailable")
		return
	}

	response := make([]domain.LyricsSearchResult, 0, len(results))
	for _, result := range results {
		lyrics := domain.LyricsTrack{
			Language: "lrclib", Plain: result.Plain,
			TrackName: result.TrackName, ArtistName: result.ArtistName,
		}
		for _, line := range result.Synced {
			lyrics.Synced = append(lyrics.Synced, domain.LyricLine{Time: line.Start, End: line.End, Text: line.Text})
		}
		response = append(response, domain.LyricsSearchResult{
			ID: result.ID, TrackName: result.TrackName, ArtistName: result.ArtistName,
			AlbumName: result.AlbumName, Duration: result.Duration, HasSynced: result.HasSynced,
			Lyrics: lyrics,
		})
	}
	writeJSON(w, http.StatusOK, response)
}
