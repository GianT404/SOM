package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"som/internal/scraper"
)

const maxQueryLen = 100

var reVideoID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

func validateVideoID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing required parameter: id")
		return "", false
	}
	if !reVideoID.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid id")
		return "", false
	}
	return id, true
}

// SearchHandler handles GET /api/v1/search?q={keyword}.
type SearchHandler struct {
	Scraper scraper.Scraper
}

func (h *SearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, http.StatusBadRequest, "missing required parameter: q")
		return
	}
	if len(q) > maxQueryLen {
		writeError(w, http.StatusBadRequest, "q too long, max 100 characters")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 110*time.Second)
	defer cancel()

	results, err := h.Scraper.Search(ctx, q)
	if err != nil {
		if errors.Is(err, scraper.ErrCircuitOpen) {
			w.Header().Set("Retry-After", "30")
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
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
	body, err := json.Marshal(map[string]string{"error": msg})
	if err != nil {
		body = []byte(`{"error":"internal error"}`)
	}
	w.Write(body)
}
