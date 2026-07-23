package handler

import (
	"context"
	"fmt"
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

	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()

	type result struct {
		source string
		data   []scraper.LyricsData
		err    error
	}
	resCh := make(chan result, 2)

	go func() {
		// Step 1: try fetching structured metadata from YouTube Music
		meta, metaErr := h.Scraper.VideoMetadata(ctx, id)
		if metaErr == nil && meta.Track != "" {
			log.Printf("lyrics: trying metadata candidate %q by %q", meta.Track, meta.Artist)
			if data, err := scraper.GetLRCLibLyrics(ctx, meta.Track, meta.Artist, duration); err == nil && len(data) > 0 {
				resCh <- result{source: "lrclib", data: data}
				return
			}
		}
		if title == "" {
			resCh <- result{source: "lrclib", err: fmt.Errorf("no title provided")}
			return
		}
		log.Printf("lyrics: searching LRCLib via title for %q by %q", title, artist)
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

	var merged []scraper.LyricsData
	if lrclibRes != nil && lrclibRes.err == nil {
		merged = append(merged, lrclibRes.data...)
	}
	if youtubeRes != nil && youtubeRes.err == nil {
		merged = append(merged, youtubeRes.data...)
	}

	if len(merged) > 0 {
		log.Printf("lyrics: merged %d language track(s) for %s", len(merged), id)
		writeJSON(w, http.StatusOK, merged)
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
