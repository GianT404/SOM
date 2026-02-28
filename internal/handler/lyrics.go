package handler

import (
	"context"
	"net/http"
	"time"

	"dm4a/internal/scraper"
)

// LyricsHandler handles GET /api/v1/lyrics?id={video_id}.
type LyricsHandler struct {
	Scraper scraper.Scraper
}

func (h *LyricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing required parameter: id")
		return
	}

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
