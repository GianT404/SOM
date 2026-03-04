package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"dm4a/internal/scraper"
)

// ResolveHandler handles GET /api/v1/resolve?id={video_id}.
// It returns the direct audio URL and title for the given video ID,
// so the client can download the file directly from the CDN
// without waiting for the server to proxy the entire file.
type ResolveHandler struct {
	Scraper scraper.Scraper
}

type resolveResponse struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	SafeName string `json:"safeName"` // Safe filename (without extension)
}

func (h *ResolveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing required parameter: id")
		return
	}

	log.Printf("resolve: fetching stream info for %s", id)
	start := time.Now()

	info, err := h.Scraper.GetStreamInfo(r.Context(), id)
	if err != nil {
		log.Printf("resolve: error for %s after %v: %v", id, time.Since(start), err)
		writeError(w, http.StatusInternalServerError, "failed to resolve audio URL")
		return
	}

	log.Printf("resolve: got URL for %s in %v", id, time.Since(start))

	resp := resolveResponse{
		URL:      info.URL,
		Title:    info.Title,
		SafeName: safeFilename(info.Title),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
