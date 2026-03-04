package handler

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"dm4a/internal/scraper"
)

// LyricsHandler handles GET /api/v1/lyrics?id={video_id}&title={title}&artist={artist}&duration={seconds}.
//
// Strategy:
//  1. If title is provided, query LRCLib first (fast, no yt-dlp / no rate-limit risk).
//  2. Fall back to the Scraper (yt-dlp subtitles) only when LRCLib returns nothing.
type LyricsHandler struct {
	Scraper scraper.Scraper
}

func (h *LyricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing required parameter: id")
		return
	}

	title := r.URL.Query().Get("title")
	artist := r.URL.Query().Get("artist")
	durationStr := r.URL.Query().Get("duration")
	var duration float64
	if durationStr != "" {
		duration, _ = strconv.ParseFloat(durationStr, 64)
	}

	// ---- Strategy 1: LRCLib (fast, no rate limit) ----
	if title != "" {
		log.Printf("lyrics: trying LRCLib for %q by %q (%.0fs)", title, artist, duration)
		lrcData, err := scraper.GetLRCLibLyrics(r.Context(), title, artist, duration)
		if err == nil && len(lrcData) > 0 {
			log.Printf("lyrics: LRCLib OK for %s (%d result(s))", id, len(lrcData))
			writeJSON(w, http.StatusOK, lrcData)
			return
		}
		log.Printf("lyrics: LRCLib miss for %q: %v — falling back to yt-dlp", title, err)
	}

	// ---- Strategy 2: yt-dlp subtitles (fallback) ----
	log.Printf("lyrics: using yt-dlp fallback for %s", id)
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	lyrics, err := h.Scraper.Lyrics(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if len(lyrics) == 0 {
		writeJSON(w, http.StatusOK, []scraper.LyricsData{})
		return
	}

	writeJSON(w, http.StatusOK, lyrics)
}
