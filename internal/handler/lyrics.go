package handler

import (
	"log"
	"net/http"
	"strconv"

	"dm4a/internal/scraper"
)

// LyricsHandler handles GET /api/v1/lyrics?id={video_id}&title={title}&artist={artist}&duration={seconds}.
// Uses LRCLib exclusively. If no lyrics are found, returns an empty array.
type LyricsHandler struct {
	Scraper scraper.Scraper // kept for interface compatibility, no longer used
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

	if title == "" {
		// No metadata to search with — report no lyrics immediately.
		log.Printf("lyrics: no title provided for %s, returning empty", id)
		writeJSON(w, http.StatusOK, []scraper.LyricsData{})
		return
	}

	log.Printf("lyrics: LRCLib search for %q by %q (%.0fs)", title, artist, duration)
	data, err := scraper.GetLRCLibLyrics(r.Context(), title, artist, duration)
	if err != nil || len(data) == 0 {
		log.Printf("lyrics: no LRCLib results for %q: %v", title, err)
		writeJSON(w, http.StatusOK, []scraper.LyricsData{})
		return
	}

	log.Printf("lyrics: LRCLib OK for %s", id)
	writeJSON(w, http.StatusOK, data)
}
