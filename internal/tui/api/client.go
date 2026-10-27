package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"som/internal/domain"
	"som/internal/fileutil"
)

type HTTPProvider struct {
	base   string
	apiKey string
	short  *http.Client
	stream *http.Client
}

// NewHTTPProvider tạo client cho remote mode (--server). apiKey được gắn vào header X-API-Key trên mọi request — backend (AuthMiddleware)
func NewHTTPProvider(baseURL, apiKey string) *HTTPProvider {
	return &HTTPProvider{
		base:   strings.TrimRight(baseURL, "/"),
		apiKey: apiKey,
		short:  &http.Client{Timeout: 30 * time.Second},
		stream: &http.Client{Timeout: 0},
	}
}

func (c *HTTPProvider) getShort(ctx context.Context, path string, params url.Values) (*http.Response, error) {
	return c.doGet(ctx, c.short, path, params)
}

func (c *HTTPProvider) getStream(ctx context.Context, path string, params url.Values) (*http.Response, error) {
	return c.doGet(ctx, c.stream, path, params)
}

func (c *HTTPProvider) doGet(ctx context.Context, hc *http.Client, path string, params url.Values) (*http.Response, error) {
	u := c.base + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	return hc.Do(req)
}

func (c *HTTPProvider) Search(ctx context.Context, q string) ([]domain.Track, error) {
	resp, err := c.getShort(ctx, "/api/v1/search", url.Values{"q": {q}})
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search: server %s - %s", resp.Status, truncateMsg(body))
	}
	var tracks []domain.Track
	if err := json.Unmarshal(body, &tracks); err != nil {
		return nil, fmt.Errorf("search decode: %w", err)
	}
	return tracks, nil
}

func (c *HTTPProvider) ResolveStream(ctx context.Context, id string) (*domain.StreamInfo, error) {
	resp, err := c.getShort(ctx, "/api/v1/resolve", url.Values{"id": {id}})
	if err != nil {
		return nil, fmt.Errorf("resolve request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("resolve: server %s - %s", resp.Status, truncateMsg(body))
	}
	var rr struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &rr); err != nil {
		return nil, fmt.Errorf("resolve decode: %w", err)
	}
	return &domain.StreamInfo{URL: rr.URL}, nil
}

func (c *HTTPProvider) Lyrics(ctx context.Context, id, title, artist string, duration int) (domain.LyricsResp, error) {
	startedAt := time.Now()
	params := url.Values{"id": {id}}
	if title != "" {
		params.Set("title", title)
	}
	if artist != "" {
		params.Set("artist", artist)
	}
	if duration > 0 {
		params.Set("duration", fmt.Sprintf("%d", duration))
	}

	resp, err := c.getShort(ctx, "/api/v1/lyrics", params)
	if err != nil {
		log.Printf("[lyrics] remote request failed id=%s title=%q error=%v", id, title, err)
		return domain.LyricsResp{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Printf("[lyrics] remote response failed id=%s status=%s elapsed=%s", id, resp.Status, time.Since(startedAt).Round(time.Millisecond))
		return domain.LyricsResp{}, fmt.Errorf("server %s", resp.Status)
	}

	var lr domain.LyricsResp
	bodyStr := bytes.TrimSpace(body)
	if len(bodyStr) > 0 && bodyStr[0] == '[' {
		var tracks []domain.ServerLyricTrack
		if err := json.Unmarshal(body, &tracks); err != nil {
			log.Printf("[lyrics] remote response decode failed id=%s error=%v", id, err)
			return lr, err
		}
		var sources []string
		lineCount := 0
		for _, t := range tracks {
			source := t.Source
			if source == "" {
				source = t.Language
			}
			sources = append(sources, source)
			lineCount += len(t.Lines)
			lt := domain.LyricsTrack{Language: t.Language, TrackName: t.TrackName, ArtistName: t.ArtistName}
			for _, line := range t.Lines {
				lt.Synced = append(lt.Synced, domain.LyricLine{Time: line.Start, End: line.End, Text: line.Text})
			}
			lr.AllTracks = append(lr.AllTracks, lt)
		}
		if len(lr.AllTracks) > 0 {
			lr.SelectLanguage(0)
		}
		log.Printf("[lyrics] remote response id=%s tracks=%d lines=%d sources=%v elapsed=%s", id, len(tracks), lineCount, sources, time.Since(startedAt).Round(time.Millisecond))
		return lr, nil
	}
	if err := json.Unmarshal(body, &lr); err != nil {
		log.Printf("[lyrics] remote response decode failed id=%s error=%v", id, err)
		return lr, err
	}
	lr.Normalize()
	log.Printf("[lyrics] remote response id=%s tracks=%d elapsed=%s", id, len(lr.AllTracks), time.Since(startedAt).Round(time.Millisecond))
	return lr, nil
}

func (c *HTTPProvider) DownloadOPUS(ctx context.Context, id, title, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	safe := sanitize(title)
	if safe == "" {
		safe = id
	}
	resp, err := c.getStream(ctx, "/api/v1/stream", url.Values{"id": {id}})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server %s", resp.Status)
	}
	dest, err := fileutil.WriteUnique(destDir, safe, id, ".opus", func(dst io.Writer) error {
		_, err := io.Copy(dst, resp.Body)
		return err
	})
	if err != nil {
		return "", err
	}
	return dest, nil
}

func sanitize(s string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", `"`, "-", "<", "-", ">", "-", "|", "-")
	return strings.TrimSpace(r.Replace(s))
}
func truncateMsg(b []byte) string {
	if len(b) > 200 {
		return string(b[:200]) + "..."
	}
	return string(b)
}
