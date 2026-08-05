package scraper

import (
	"context"
	"sync"
	"time"
)

const (
	searchCacheTTL = 5 * time.Minute
	streamCacheTTL = 50 * time.Minute
)

type searchCacheEntry struct {
	results   []SearchResult
	expiresAt time.Time
}

type streamCacheEntry struct {
	info      *StreamInfo
	expiresAt time.Time
}

type ResilientScraper struct {
	inner Scraper

	searchMu    sync.Mutex
	searchCache map[string]searchCacheEntry

	streamMu    sync.Mutex
	streamCache map[string]streamCacheEntry

	searchCB *circuitBreaker
	streamCB *circuitBreaker
}

func NewResilientScraper(inner Scraper) *ResilientScraper {
	s := &ResilientScraper{
		inner:       inner,
		searchCache: make(map[string]searchCacheEntry),
		streamCache: make(map[string]streamCacheEntry),
		searchCB:    newCircuitBreaker(5, 30*time.Second),
		streamCB:    newCircuitBreaker(5, 30*time.Second),
	}
	go s.cleanupLoop()
	return s
}

func (s *ResilientScraper) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()

		s.searchMu.Lock()
		for k, e := range s.searchCache {
			if now.After(e.expiresAt) {
				delete(s.searchCache, k)
			}
		}
		s.searchMu.Unlock()

		s.streamMu.Lock()
		for k, e := range s.streamCache {
			if now.After(e.expiresAt) {
				delete(s.streamCache, k)
			}
		}
		s.streamMu.Unlock()
	}
}

func (s *ResilientScraper) Search(ctx context.Context, keyword string) ([]SearchResult, error) {
	s.searchMu.Lock()
	if e, ok := s.searchCache[keyword]; ok && time.Now().Before(e.expiresAt) {
		s.searchMu.Unlock()
		return e.results, nil
	}
	s.searchMu.Unlock()

	if !s.searchCB.allow() {
		return nil, ErrCircuitOpen
	}

	results, err := s.inner.Search(ctx, keyword)
	s.searchCB.recordResult(err)
	if err != nil {
		return nil, err
	}

	s.searchMu.Lock()
	s.searchCache[keyword] = searchCacheEntry{results: results, expiresAt: time.Now().Add(searchCacheTTL)}
	s.searchMu.Unlock()

	return results, nil
}

func (s *ResilientScraper) GetStreamInfo(ctx context.Context, videoID string) (*StreamInfo, error) {
	s.streamMu.Lock()
	if e, ok := s.streamCache[videoID]; ok && time.Now().Before(e.expiresAt) {
		s.streamMu.Unlock()
		return e.info, nil
	}
	s.streamMu.Unlock()

	if !s.streamCB.allow() {
		return nil, ErrCircuitOpen
	}

	info, err := s.inner.GetStreamInfo(ctx, videoID)
	s.streamCB.recordResult(err)
	if err != nil {
		return nil, err
	}

	s.streamMu.Lock()
	s.streamCache[videoID] = streamCacheEntry{info: info, expiresAt: time.Now().Add(streamCacheTTL)}
	s.streamMu.Unlock()

	return info, nil
}

func (s *ResilientScraper) DownloadAudio(ctx context.Context, videoID string) (string, error) {
	return s.inner.DownloadAudio(ctx, videoID)
}

func (s *ResilientScraper) VideoTitle(ctx context.Context, videoID string) (string, error) {
	return s.inner.VideoTitle(ctx, videoID)
}

func (s *ResilientScraper) Lyrics(ctx context.Context, videoID string) ([]LyricsData, error) {
	return s.inner.Lyrics(ctx, videoID)
}

func (s *ResilientScraper) VideoMetadata(ctx context.Context, videoID string) (MusicMetadata, error) {
	return s.inner.VideoMetadata(ctx, videoID)
}
