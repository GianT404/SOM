package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"dm4a/internal/scraper"
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

	// 30-second timeout for the search operation.
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
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
