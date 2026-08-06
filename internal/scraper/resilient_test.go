package scraper

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type fakeScraper struct {
	searchCalls int32
	streamCalls int32

	searchErr error
	streamErr error

	searchResult []SearchResult
	streamResult *StreamInfo
}

func (f *fakeScraper) Search(ctx context.Context, keyword string) ([]SearchResult, error) {
	atomic.AddInt32(&f.searchCalls, 1)
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	return f.searchResult, nil
}

func (f *fakeScraper) GetStreamInfo(ctx context.Context, videoID string) (*StreamInfo, error) {
	atomic.AddInt32(&f.streamCalls, 1)
	if f.streamErr != nil {
		return nil, f.streamErr
	}
	return f.streamResult, nil
}

func (f *fakeScraper) DownloadAudio(ctx context.Context, videoID string) (string, error) {
	return "/tmp/fake.opus", nil
}

func (f *fakeScraper) VideoTitle(ctx context.Context, videoID string) (string, error) {
	return "fake title", nil
}

func (f *fakeScraper) Lyrics(ctx context.Context, videoID string) ([]LyricsData, error) {
	return nil, nil
}

func (f *fakeScraper) VideoMetadata(ctx context.Context, videoID string) (MusicMetadata, error) {
	return MusicMetadata{}, nil
}

func TestResilientScraper_Search_CachesResult(t *testing.T) {
	inner := &fakeScraper{searchResult: []SearchResult{{ID: "abc"}}}
	s := NewResilientScraper(inner)

	if _, err := s.Search(context.Background(), "khác gì đâu"); err != nil {
		t.Fatalf("error: %v", err)
	}
	if _, err := s.Search(context.Background(), "khác gì đâu"); err != nil {
		t.Fatalf("error: %v", err)
	}

	if got := atomic.LoadInt32(&inner.searchCalls); got != 1 {
		t.Fatalf("cache %d", got)
	}
}

func TestResilientScraper_Search_DifferentKeywordsNotCached(t *testing.T) {
	inner := &fakeScraper{searchResult: []SearchResult{{ID: "abc"}}}
	s := NewResilientScraper(inner)

	s.Search(context.Background(), "bai hat A")
	s.Search(context.Background(), "bai hat B")

	if got := atomic.LoadInt32(&inner.searchCalls); got != 2 {
		t.Fatalf(" %d", got)
	}
}

func TestResilientScraper_Search_ExpiredCacheRefetches(t *testing.T) {
	inner := &fakeScraper{searchResult: []SearchResult{{ID: "abc"}}}
	s := NewResilientScraper(inner)

	s.Search(context.Background(), "song X")

	s.searchMu.Lock()
	e := s.searchCache["song X"]
	e.expiresAt = time.Now().Add(-time.Minute)
	s.searchCache["song X"] = e
	s.searchMu.Unlock()

	s.Search(context.Background(), "song X")

	if got := atomic.LoadInt32(&inner.searchCalls); got != 2 {
		t.Fatalf("cache het han phai goi lai inner, thuc te goi %d lan", got)
	}
}

func TestResilientScraper_Search_CircuitOpensAfterFailures(t *testing.T) {
	inner := &fakeScraper{searchErr: errors.New("yt-dlp loi gia lap")}
	s := NewResilientScraper(inner)
	for i := 0; i < 5; i++ {
		q := "query-" + string(rune('A'+i))
		if _, err := s.Search(context.Background(), q); err == nil {
			t.Fatal("phai tra ve loi tu inner khi inner that bai")
		}
	}

	callsBefore := atomic.LoadInt32(&inner.searchCalls)

	_, err := s.Search(context.Background(), "query-moi-sau-khi-mo-circuit")
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("sau 5 loi lien tiep, circuit phai mo va tra ErrCircuitOpen, got %v", err)
	}
	if got := atomic.LoadInt32(&inner.searchCalls); got != callsBefore {
		t.Fatalf("khi circuit dang mo, khong duoc goi them inner, truoc=%d sau=%d", callsBefore, got)
	}
}

func TestResilientScraper_GetStreamInfo_CachesResult(t *testing.T) {
	inner := &fakeScraper{streamResult: &StreamInfo{URL: "https://cdn.example/audio.opus", Title: "test"}}
	s := NewResilientScraper(inner)

	if _, err := s.GetStreamInfo(context.Background(), "videoID1"); err != nil {
		t.Fatalf("lan goi dau khong duoc loi: %v", err)
	}
	if _, err := s.GetStreamInfo(context.Background(), "videoID1"); err != nil {
		t.Fatalf("lan goi 2 tu cache khong duoc loi: %v", err)
	}

	if got := atomic.LoadInt32(&inner.streamCalls); got != 1 {
		t.Fatalf("cung videoID goi 2 lan, inner.GetStreamInfo chi duoc goi 1 lan, thuc te %d", got)
	}
}

func TestResilientScraper_GetStreamInfo_CircuitOpensAfterFailures(t *testing.T) {
	inner := &fakeScraper{streamErr: errors.New("resolve loi gia lap")}
	s := NewResilientScraper(inner)

	for i := 0; i < 5; i++ {
		id := "video-" + string(rune('A'+i))
		if _, err := s.GetStreamInfo(context.Background(), id); err == nil {
			t.Fatal("phai tra ve loi tu inner khi inner that bai")
		}
	}

	_, err := s.GetStreamInfo(context.Background(), "video-moi")
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("sau 5 loi lien tiep, circuit phai mo va tra ErrCircuitOpen, got %v", err)
	}
}

func TestResilientScraper_DownloadAudio_AlwaysForwardsToInner(t *testing.T) {
	inner := &fakeScraper{}
	s := NewResilientScraper(inner)

	p1, err := s.DownloadAudio(context.Background(), "vid")
	if err != nil {
		t.Fatalf("khong duoc loi: %v", err)
	}
	p2, _ := s.DownloadAudio(context.Background(), "vid")

	if p1 != "/tmp/fake.opus" || p2 != "/tmp/fake.opus" {
		t.Fatal("DownloadAudio phai forward thang ket qua tu inner, khong cache")
	}
}
