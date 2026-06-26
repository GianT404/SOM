package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

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

	// Chạy LRCLib và YouTube captions song song để giảm độ trễ tổng:
	// worst-case giờ là max(lrclib, youtube) thay vì lrclib + youtube.
	// YouTube path tự chia ngân sách 4s (direct) + 6s (yt-dlp) = 10s bên
	// trong; 12s ở đây chỉ là lớp an toàn ngoài cùng, không phải nguồn cắt
	// chính — tránh hai lớp timeout đua nhau gây log nhiễu signal:killed.
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()

	type result struct {
		source string
		data   []scraper.LyricsData
		err    error
	}
	resCh := make(chan result, 2)

	go func() {
		if title == "" {
			resCh <- result{source: "lrclib", err: fmt.Errorf("no title provided")}
			return
		}
		log.Printf("lyrics: searching LRCLib for %q by %q", title, artist)
		data, err := scraper.TryMultipleLyricProviders(ctx, title, artist, duration)
		resCh <- result{source: "lrclib", data: data, err: err}
	}()

	go func() {
		log.Printf("lyrics: trying YouTube captions for %s", id)
		data, err := h.Scraper.Lyrics(ctx, id)
		resCh <- result{source: "youtube", data: data, err: err}
	}()

	var lrclibRes, youtubeRes *result
	for i := 0; i < 2; i++ {
		res := <-resCh
		switch res.source {
		case "lrclib":
			lrclibRes = &res
		case "youtube":
			youtubeRes = &res
		}
	}

	// LRCLib vẫn là nguồn ưu tiên khi có dữ liệu hợp lệ.
	if lrclibRes != nil && lrclibRes.err == nil && len(lrclibRes.data) > 0 {
		log.Printf("lyrics: LRCLib OK for %s", id)
		writeJSON(w, http.StatusOK, lrclibRes.data)
		return
	}

	if youtubeRes != nil && youtubeRes.err == nil && len(youtubeRes.data) > 0 {
		log.Printf("lyrics: YouTube CC fallback OK for %s", id)
		writeJSON(w, http.StatusOK, youtubeRes.data)
		return
	}

	var lrclibErr, youtubeErr error
	if lrclibRes != nil {
		lrclibErr = lrclibRes.err
	}
	if youtubeRes != nil {
		youtubeErr = youtubeRes.err
	}
	log.Printf("lyrics: both LRCLib and YouTube CC failed for %s (lrclib_err=%v youtube_err=%v)",
		id, lrclibErr, youtubeErr)
	writeJSON(w, http.StatusOK, []scraper.LyricsData{})
}
