package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"som/internal/backend"
	"som/internal/scraper"
)

type stubScraper struct{}

func (stubScraper) Search(ctx context.Context, keyword string) ([]scraper.SearchResult, error) {
	return nil, nil
}
func (stubScraper) GetStreamInfo(ctx context.Context, videoID string) (*scraper.StreamInfo, error) {
	return nil, nil
}
func (stubScraper) DownloadAudio(ctx context.Context, videoID string) (string, error) {
	return "", nil
}
func (stubScraper) VideoTitle(ctx context.Context, videoID string) (string, error) {
	return "", nil
}
func (stubScraper) Lyrics(ctx context.Context, videoID string) ([]scraper.LyricsData, error) {
	return nil, nil
}
func (stubScraper) VideoMetadata(ctx context.Context, videoID string) (scraper.MusicMetadata, error) {
	return scraper.MusicMetadata{}, nil
}

// newTestRouter dựng router với scraper giả; không gọi yt-dlp thật.
func newTestRouter(t *testing.T, apiKey string, reg *backend.DeviceRegistry) http.Handler {
	t.Helper()
	return newRouter(scraper.NewResilientScraper(stubScraper{}), apiKey, reg)
}

// Smoke test wiring chi: trước đây đặt Use() sau route gây panic lúc khởi động.
func TestRouter_WiringAndAuth(t *testing.T) {
	const apiKey = "test-secret-key"
	reg := backend.NewDeviceRegistry()
	h := newTestRouter(t, apiKey, reg)

	// /health public.
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("/health = %d, want 200", rr.Code)
	}

	// /api/v1/auth/register public, trả token.
	body := bytes.NewBufferString(`{"device_id":"device-test-1"}`)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", body))
	if rr.Code != http.StatusOK {
		t.Fatalf("register = %d, want 200", rr.Code)
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil || resp.Token == "" {
		t.Fatalf("register response invalid: token=%q err=%v", resp.Token, err)
	}

	// Route có auth: chưa có credential → 401.
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/search?q=x", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized search = %d, want 401", rr.Code)
	}

	// Dùng token → đi qua auth (handler giả trả 200, không cần yt-dlp).
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=x", nil)
	req.Header.Set("X-Device-Token", resp.Token)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("authed search = %d, want 200", rr.Code)
	}
}
