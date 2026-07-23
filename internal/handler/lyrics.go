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

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	type result struct {
		source string
		data   []scraper.LyricsData
		err    error
	}
	resCh := make(chan result, 2)

	go func() {
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
