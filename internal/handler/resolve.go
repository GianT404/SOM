package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"som/internal/scraper"
)

type ResolveHandler struct {
	Scraper scraper.Scraper
}

type resolveResponse struct {
	URL      string `json:"url"`
	Title    string `json:"title"`
	SafeName string `json:"safeName"`
}

func (h *ResolveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, ok := validateVideoID(w, r)
	if !ok {
		return
	}

	log.Printf("resolve: fetching stream info for %s", id)
	start := time.Now()

	info, err := h.Scraper.GetStreamInfo(r.Context(), id)
	if err != nil {
		log.Printf("resolve: error for %s after %v: %v", id, time.Since(start), err)
		if errors.Is(err, scraper.ErrCircuitOpen) {
			w.Header().Set("Retry-After", "30")
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
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
