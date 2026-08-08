package scraper

import (
	"context"
	"sync"
	"time"

	"som/internal/cache"
)

const (
	searchCacheTTL = 5 * time.Minute
	// URL chữ ký YouTube sống ~6h nên cache 4h là an toàn, giảm đáng kể số
	// lần phải chạy lại yt-dlp khi replay một bài.
	streamCacheTTL   = 4 * time.Hour
	lyricsCacheTTL   = 7 * 24 * time.Hour
	maxLyricsCache   = 500
	metadataCacheTTL = 24 * time.Hour
	maxMetadataCache = 1000

	bgWorkTimeout = 50 * time.Second
)

type searchCacheEntry struct {
	results   []SearchResult
	expiresAt time.Time
}

type streamCacheEntry struct {
	info      *StreamInfo
	expiresAt time.Time
}

type searchFlight struct {
	done    chan struct{}
	results []SearchResult
	err     error
}

type streamFlight struct {
	done chan struct{}
	info *StreamInfo
	err  error
}

type ResilientScraper struct {
	inner Scraper

	searchMu    sync.Mutex
	searchCache map[string]searchCacheEntry

	streamMu    sync.Mutex
	streamCache map[string]streamCacheEntry

	lyricsCache *cache.TTLCache[[]LyricsData]
	metaCache   *cache.TTLCache[MusicMetadata]

	searchCB *circuitBreaker
	streamCB *circuitBreaker

	searchFlightMu sync.Mutex
	searchFlight   map[string]*searchFlight

	streamFlightMu sync.Mutex
	streamFlight   map[string]*streamFlight
}

func NewResilientScraper(inner Scraper) *ResilientScraper {
	s := &ResilientScraper{
		inner:        inner,
		searchCache:  make(map[string]searchCacheEntry),
		streamCache:  make(map[string]streamCacheEntry),
		lyricsCache:  cache.NewTTL[[]LyricsData](maxLyricsCache, lyricsCacheTTL),
		metaCache:    cache.NewTTL[MusicMetadata](maxMetadataCache, metadataCacheTTL),
		searchCB:     newCircuitBreaker(5, 30*time.Second),
		streamCB:     newCircuitBreaker(5, 30*time.Second),
		searchFlight: make(map[string]*searchFlight),
		streamFlight: make(map[string]*streamFlight),
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

	// Single-flight: nếu đã có request trùng keyword đang chạy, chờ kết quả
	// của nó thay vì spawn thêm 1 process yt-dlp nữa.
	s.searchFlightMu.Lock()
	if inf, ok := s.searchFlight[keyword]; ok {
		s.searchFlightMu.Unlock()
		// Chờ theo deadline của CHÍNH caller này, không phải deadline của
		// caller đã tạo ra flight (caller đó có thể đã bỏ cuộc từ lâu).
		select {
		case <-inf.done:
			return inf.results, inf.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	inf := &searchFlight{done: make(chan struct{})}
	s.searchFlight[keyword] = inf
	s.searchFlightMu.Unlock()

	// Context nền tách khỏi ctx của request: client hủy/timeout không được
	// phép kill process giữa chừng. Để yt-dlp chạy hết, ghi cache lại —
	// request retry ngay sau đó (hoặc waiter khác đang chờ flight) ăn luôn
	// kết quả thay vì phải cold-start lại từ đầu.
	bgCtx, bgCancel := context.WithTimeout(context.Background(), bgWorkTimeout)

	go func() {
		defer bgCancel()

		results, err := s.inner.Search(bgCtx, keyword)
		s.searchCB.recordResult(err)

		s.searchFlightMu.Lock()
		inf.results = results
		inf.err = err
		close(inf.done)
		delete(s.searchFlight, keyword)
		s.searchFlightMu.Unlock()

		if err == nil {
			s.searchMu.Lock()
			s.searchCache[keyword] = searchCacheEntry{results: results, expiresAt: time.Now().Add(searchCacheTTL)}
			s.searchMu.Unlock()
		}
	}()

	select {
	case <-inf.done:
		return inf.results, inf.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
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

	s.streamFlightMu.Lock()
	if inf, ok := s.streamFlight[videoID]; ok {
		s.streamFlightMu.Unlock()
		select {
		case <-inf.done:
			return inf.info, inf.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	inf := &streamFlight{done: make(chan struct{})}
	s.streamFlight[videoID] = inf
	s.streamFlightMu.Unlock()
	bgCtx, bgCancel := context.WithTimeout(context.Background(), bgWorkTimeout)

	go func() {
		defer bgCancel()

		info, err := s.inner.GetStreamInfo(bgCtx, videoID)
		s.streamCB.recordResult(err)

		s.streamFlightMu.Lock()
		inf.info = info
		inf.err = err
		close(inf.done)
		delete(s.streamFlight, videoID)
		s.streamFlightMu.Unlock()

		if err == nil {
			s.streamMu.Lock()
			s.streamCache[videoID] = streamCacheEntry{info: info, expiresAt: time.Now().Add(streamCacheTTL)}
			s.streamMu.Unlock()
		}
	}()

	select {
	case <-inf.done:
		return inf.info, inf.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *ResilientScraper) DownloadAudio(ctx context.Context, videoID string) (string, error) {
	return s.inner.DownloadAudio(ctx, videoID)
}

func (s *ResilientScraper) VideoTitle(ctx context.Context, videoID string) (string, error) {
	return s.inner.VideoTitle(ctx, videoID)
}

func (s *ResilientScraper) Lyrics(ctx context.Context, videoID string) ([]LyricsData, error) {
	if v, ok := s.lyricsCache.Get(videoID); ok {
		return v, nil
	}
	data, err := s.inner.Lyrics(ctx, videoID)
	if err != nil {
		return nil, err
	}
	s.lyricsCache.Put(videoID, data)
	return data, nil
}

func (s *ResilientScraper) VideoMetadata(ctx context.Context, videoID string) (MusicMetadata, error) {
	if v, ok := s.metaCache.Get(videoID); ok {
		return v, nil
	}
	meta, err := s.inner.VideoMetadata(ctx, videoID)
	if err != nil {
		return MusicMetadata{}, err
	}
	s.metaCache.Put(videoID, meta)
	return meta, nil
}
