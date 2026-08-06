package handler

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"som/internal/scraper"
)

type LyricsHandler struct {
	Scraper scraper.Scraper
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

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	type result struct {
		source string
		data   []scraper.LyricsData
		err    error
	}
	resCh := make(chan result, 2)

	go func() {
		// VideoMetadata trả track/artist rỗng nếu video không có metadata
		// YouTube Music (rất phổ biến với video tự upload/rap/indie) - khi đó
		// FetchLyrics tự rơi xuống candidate parse từ title, không cần xử lý
		// rẽ nhánh ở đây nữa.
		meta, metaErr := h.Scraper.VideoMetadata(ctx, id)
		if metaErr != nil {
			meta = scraper.MusicMetadata{}
		}
		// artist truyền từ client (vd uploader/tên kênh) dùng làm hint dự phòng
		// nếu metadata YouTube Music không có sẵn artist.
		if meta.Artist == "" && artist != "" {
			meta.Artist = artist
		}
		log.Printf("lyrics: fetching for %s (metadata track=%q artist=%q, title=%q)",
			id, meta.Track, meta.Artist, title)
		data, err := scraper.FetchLyrics(ctx, meta, title, duration)
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

	var combined []scraper.LyricsData
	if lrclibRes != nil && lrclibRes.err == nil && len(lrclibRes.data) > 0 {
		combined = append(combined, lrclibRes.data...)
	}
	if youtubeRes != nil && youtubeRes.err == nil && len(youtubeRes.data) > 0 {
		seen := make(map[string]bool, len(combined))
		for _, d := range combined {
			seen[d.Language] = true
		}
		for _, d := range youtubeRes.data {
			if seen[d.Language] {
				continue
			}
			seen[d.Language] = true
			combined = append(combined, d)
		}
	}

	if len(combined) > 0 {
		log.Printf("lyrics: OK for %s (%d language track(s))", id, len(combined))
		writeJSON(w, http.StatusOK, combined)
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
