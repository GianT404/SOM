package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"dm4a/internal/scraper"
)

// StreamHandler handles GET /api/v1/stream?id={video_id}.
// Uses yt-dlp to download audio and pipe it directly to the client.
// This bypasses YouTube's n-parameter throttle (which limits raw URL
// downloads to ~30 KB/s) because yt-dlp decrypts the throttle internally.
type StreamHandler struct {
	Scraper scraper.Scraper

	// Title cache to avoid extra yt-dlp calls.
	cacheMu sync.RWMutex
	cache   map[string]*titleCache
}

type titleCache struct {
	title     string
	expiresAt time.Time
}

// NewStreamHandler creates a StreamHandler.
func NewStreamHandler(sc scraper.Scraper) *StreamHandler {
	return &StreamHandler{
		Scraper: sc,
		cache:   make(map[string]*titleCache),
	}
}

// safeFilename strips characters that are invalid in filenames.
var reUnsafe = regexp.MustCompile(`[<>:"/\\|?*]`)

func safeFilename(title string) string {
	safe := reUnsafe.ReplaceAllString(title, "")
	safe = strings.TrimSpace(safe)
	if safe == "" {
		safe = "audio"
	}
	return safe
}

// getCachedTitle returns a cached title or fetches it.
func (h *StreamHandler) getCachedTitle(ctx context.Context, id string) string {
	h.cacheMu.RLock()
	if cached, ok := h.cache[id]; ok && time.Now().Before(cached.expiresAt) {
		h.cacheMu.RUnlock()
		return cached.title
	}
	h.cacheMu.RUnlock()

	title, err := h.Scraper.VideoTitle(ctx, id)
	if err != nil {
		return id // fallback
	}

	h.cacheMu.Lock()
	h.cache[id] = &titleCache{
		title:     title,
		expiresAt: time.Now().Add(1 * time.Hour),
	}
	h.cacheMu.Unlock()

	return title
}

func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing required parameter: id")
		return
	}

	// Get the title (with 10s timeout, runs in parallel with stream start).
	titleCtx, titleCancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer titleCancel()

	titleCh := make(chan string, 1)
	go func() {
		titleCh <- h.getCachedTitle(titleCtx, id)
	}()

	log.Printf("stream: starting download for %s", id)
	start := time.Now()

	tempFile, err := h.Scraper.DownloadAudio(r.Context(), id)
	if err != nil {
		log.Printf("stream: download error for %s after %v: %v", id, time.Since(start), err)
		writeError(w, http.StatusInternalServerError, "failed to download audio")
		return
	}

	log.Printf("stream: downloaded %s to %s in %v", id, tempFile, time.Since(start))

	title := <-titleCh
	filename := safeFilename(title) + ".dm4a"

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	// Serve the file (supports Range requests automatically)
	http.ServeFile(w, r, tempFile)
}
