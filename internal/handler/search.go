package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"som/internal/scraper"
)

// SearchHandler handles GET /api/v1/search?q={keyword}.
type SearchHandler struct {
	Scraper scraper.Scraper
}

func (h *SearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "missing required parameter: q")
		return
	}

	// yt-dlp search can be slow on first run or on restricted networks.
	ctx, cancel := context.WithTimeout(r.Context(), 110*time.Second)
	defer cancel()

	results, err := h.Scraper.Search(ctx, q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, results)
}

// --- shared helpers ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	// Escape any quotes in the message for valid JSON.
	safe := strings.ReplaceAll(msg, `"`, `\"`)
	w.Write([]byte(`{"error":"` + safe + `"}`))
}
