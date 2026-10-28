package scraper

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestRankLRCLibSearchResults_PrioritizesSyncedAndLimitsResults(t *testing.T) {
	tracks := make([]lrclibTrack, 0, 10)
	for i := 0; i < 10; i++ {
		tracks = append(tracks, lrclibTrack{ID: i + 1, TrackName: "Don't Say Lazy", ArtistName: "Artist", Duration: 180, PlainLyrics: "plain lyrics"})
	}
	tracks[0].SyncedLyrics = "[00:01.00]first line"
	tracks[2].SyncedLyrics = "[00:02.00]second line"
	results := rankLRCLibSearchResults("Don't Say Lazy", tracks, 7)
	if len(results) != 7 {
		t.Fatalf("got %d results, want 7", len(results))
	}
	if !results[0].HasSynced || !results[1].HasSynced {
		t.Fatal("synced lyrics should rank before plain lyrics")
	}
	for i := 2; i < len(results); i++ {
		if results[i].HasSynced {
			t.Fatalf("synced result unexpectedly ranked after plain at index %d", i)
		}
	}
}

func TestRankLRCLibSearchResults_SkipsEntriesWithoutLyrics(t *testing.T) {
	results := rankLRCLibSearchResults("Track", []lrclibTrack{
		{ID: 1, TrackName: "Track"},
		{ID: 2, TrackName: "Track", PlainLyrics: "  "},
		{ID: 3, TrackName: "Track", PlainLyrics: "line"},
	}, 7)
	if len(results) != 1 || results[0].ID != 3 {
		t.Fatalf("results = %#v, want only result 3", results)
	}
}


func TestFetchLRCLibSearchRetries503(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, "temporarily busy", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]lrclibTrack{{ID: 1, TrackName: "Track", PlainLyrics: "line"}})
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	results, err := fetchLRCLibSearchWithClient(ctx, server.Client(), server.URL)
	if err != nil {
		t.Fatalf("fetchLRCLibSearchWithClient() error = %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("request count = %d, want 2", calls.Load())
	}
	if len(results) != 1 || results[0].TrackName != "Track" {
		t.Fatalf("results = %#v, want one track", results)
	}
}

func TestFetchLRCLibSearchPreserves503AfterRetries(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "0")
		http.Error(w, "temporarily busy", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := fetchLRCLibSearchWithClient(ctx, server.Client(), server.URL)
	var upstreamErr *LRCLibHTTPError
	if !errors.As(err, &upstreamErr) {
		t.Fatalf("error = %v, want LRCLibHTTPError", err)
	}
	if upstreamErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", upstreamErr.StatusCode)
	}
	if upstreamErr.RetryAfter != "0" {
		t.Fatalf("Retry-After = %q, want 0", upstreamErr.RetryAfter)
	}
	if got := calls.Load(); got != lrclibSearchMaxAttempts {
		t.Fatalf("request count = %s, want %d", strconv.Itoa(int(got)), lrclibSearchMaxAttempts)
	}
}
