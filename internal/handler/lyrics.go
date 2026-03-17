package handler

import (
	"log"
	"net/http"
	"strconv"

	"dm4a/internal/scraper"
)

// LyricsHandler handles GET /api/v1/lyrics?id={video_id}&title={title}&artist={artist}&duration={seconds}.
// Uses LRCLib exclusively. If title is not provided, fetches it from the video metadata.
// If no lyrics are found, returns an empty array.
type LyricsHandler struct {
	Scraper scraper.Scraper // Dùng để lấy tiêu đề video khi không được cung cấp
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

	// 1. Thử gọi LRCLib trước nếu có title
	if title != "" {
		log.Printf("lyrics: LRCLib search for %q by %q", title, artist)
		data, err := scraper.GetLRCLibLyrics(r.Context(), title, artist, duration)
		// Nếu có dữ liệu hợp lệ từ LRCLib thì trả về luôn
		if err == nil && len(data) > 0 {
			log.Printf("lyrics: LRCLib OK for %s", id)
			writeJSON(w, http.StatusOK, data)
			return
		}
	}

	// 2. PHƯƠNG ÁN DỰ PHÒNG: Cào trực tiếp phụ đề từ YouTube
	log.Printf("lyrics: LRCLib failed or no title, falling back to YouTube captions for %s", id)
	fallbackData, err := h.Scraper.Lyrics(r.Context(), id)
	
	if err != nil || len(fallbackData) == 0 {
		log.Printf("lyrics: fallback YouTube CC also failed for %s: %v", id, err)
		writeJSON(w, http.StatusOK, []scraper.LyricsData{})
		return
	}

	log.Printf("lyrics: YouTube CC fallback OK for %s", id)
	writeJSON(w, http.StatusOK, fallbackData)
}
