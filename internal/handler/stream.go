package handler

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"som/internal/scraper"
)

// StreamHandler redirects to the direct audio URL resolved by yt-dlp.
// Trước đây endpoint này tải cả file về temp rồi mới serve -> first byte
// phải chờ toàn bộ quá trình download (30s+). Giờ nó 302 sang URL CDN mà
// yt-dlp đã resolve sẵn (đã qua xử lý signature/throttling) và được cache
// 50 phút trong ResilientScraper: client nhận response tức thì, backend
// không còn proxy nguyên file qua mạng.
type StreamHandler struct {
	Scraper scraper.Scraper
}

// NewStreamHandler creates a StreamHandler.
func NewStreamHandler(sc scraper.Scraper) *StreamHandler {
	return &StreamHandler{Scraper: sc}
}

// safeFilename strips characters that are invalid in filenames.
// \x00-\x1f loại bỏ control characters (gồm CRLF) để chống header injection
// trong Content-Disposition.
var reUnsafe = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

func safeFilename(title string) string {
	safe := reUnsafe.ReplaceAllString(title, "")
	safe = strings.TrimSpace(safe)
	if safe == "" {
		safe = "audio"
	}
	return safe
}

func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, ok := validateVideoID(w, r)
	if !ok {
		return
	}

	info, err := h.Scraper.GetStreamInfo(r.Context(), id)
	if err != nil {
		log.Printf("stream: resolve error for %s: %v", id, err)
		if errors.Is(err, scraper.ErrCircuitOpen) {
			w.Header().Set("Retry-After", "30")
			writeError(w, http.StatusServiceUnavailable, "stream temporarily unavailable, try again later")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to resolve audio URL")
		return
	}

	filename := safeFilename(info.Title) + ".opus"
	w.Header().Set("Content-Type", "audio/ogg")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	http.Redirect(w, r, info.URL, http.StatusFound)
}
