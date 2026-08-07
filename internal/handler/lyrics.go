package handler

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"som/internal/cache"
	"som/internal/scraper"
)

type LyricsHandler struct {
	Scraper scraper.Scraper

	// cache lưu kết quả lyrics đã merge (LRCLib + YouTube) theo key video.
	// Lời gắn với video hiếm khi đổi nên cache 7 ngày, tránh spawn yt-dlp
	// mỗi lần chơi lại bài cũ.
	cache *cache.TTLCache[[]scraper.LyricsData]
}

func (h *LyricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, ok := validateVideoID(w, r)
	if !ok {
		return
	}

	title := r.URL.Query().Get("title")
	artist := r.URL.Query().Get("artist")
	durationStr := r.URL.Query().Get("duration")
	var duration float64
	if durationStr != "" {
		d, err := strconv.ParseFloat(durationStr, 64)
		if err != nil || d < 0 {
			writeError(w, http.StatusBadRequest, "invalid duration")
			return
		}
		duration = d
	}

	// Cache key kèm hint từ client để tránh phục vụ nhầm lyrics khi metadata
	// truyền lên khác nhau.
	cacheKey := id + "|" + title + "|" + artist + "|" + durationStr
	if h.cache == nil {
		h.cache = cache.NewTTL[[]scraper.LyricsData](500, 7*24*time.Hour)
	}
	if v, ok := h.cache.Get(cacheKey); ok {
		log.Printf("lyrics: cache hit for %s", id)
		writeJSON(w, http.StatusOK, v)
		return
	}

	// Budget tổng < timeout ~10s của app, để app không timeout.
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	type result struct {
		source string
		data   []scraper.LyricsData
		err    error
	}

	// Ưu tiên LRCLib: dùng hint title/artist/duration TỪ CLIENT luôn, không
	// spawn yt-dlp (vừa nhanh vừa không chiếm slot) — app đã truyền sẵn các
	// field này từ kết quả search.
	lrCh := make(chan result, 1)
	go func() {
		meta := scraper.MusicMetadata{Artist: artist}
		data, err := scraper.FetchLyrics(ctx, meta, title, duration)
		lrCh <- result{source: "lrclib", data: data, err: err}
	}()

	// YouTube captions chạy song song làm fallback khi LRCLib không có.
	ytCh := make(chan result, 1)
	go func() {
		data, err := h.Scraper.Lyrics(ctx, id)
		ytCh <- result{source: "youtube", data: data, err: err}
	}()
	log.Printf("lyrics: fetch for %s (title=%q artist=%q)", id, title, artist)

	// LRCLib là nguồn chính: đã có kết quả tin cậy → trả ngay, không chờ
	// YouTube. Goroutine YouTube bị hủy qua ctx khi hàm return.
	lr := <-lrCh
	if lr.err == nil && len(lr.data) > 0 {
		log.Printf("lyrics: OK for %s via %s (%d track(s))", id, lr.source, len(lr.data))
		h.cache.Put(cacheKey, lr.data)
		writeJSON(w, http.StatusOK, lr.data)
		return
	}

	// LRCLib rỗng → dùng YouTube captions (đang chạy song song).
	select {
	case yt := <-ytCh:
		if yt.err == nil && len(yt.data) > 0 {
			log.Printf("lyrics: OK for %s via %s (%d track(s))", id, yt.source, len(yt.data))
			h.cache.Put(cacheKey, yt.data)
			writeJSON(w, http.StatusOK, yt.data)
			return
		}
	case <-ctx.Done():
	}

	log.Printf("lyrics: no lyrics for %s (lrclib_err=%v, youtube cut or empty)", id, lr.err)
	// Cache cả kết quả rỗng để không phải spawn lại cho bài không có lời.
	empty := []scraper.LyricsData{}
	h.cache.Put(cacheKey, empty)
	writeJSON(w, http.StatusOK, empty)
}
