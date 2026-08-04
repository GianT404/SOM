package api

import (
	"bytes"
	"context" // Thay đổi quan trọng: Thêm Context để theo sát chữ ký Interface
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"som/internal/domain"
)

type HTTPProvider struct {
	base   string
	short  *http.Client
	stream *http.Client
}

func NewHTTPProvider(baseURL string) *HTTPProvider {
	return &HTTPProvider{
		base:   strings.TrimRight(baseURL, "/"),
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

func (c *HTTPProvider) ResolveStream(ctx context.Context, id string) (string, error) {
	resp, err := c.getShort(ctx, "/api/v1/resolve", url.Values{"id": {id}})
	if err != nil {
		return "", fmt.Errorf("resolve request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("resolve: server %s - %s", resp.Status, truncateMsg(body))
	}
	var rr struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &rr); err != nil {
		return "", fmt.Errorf("resolve decode: %w", err)
	}
	return rr.URL, nil
}

func (c *HTTPProvider) Lyrics(ctx context.Context, id, title, artist string, duration int) (domain.LyricsResp, error) {
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
		return domain.LyricsResp{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return domain.LyricsResp{}, fmt.Errorf("server %s", resp.Status)
	}

	var lr domain.LyricsResp
	bodyStr := bytes.TrimSpace(body)
	if len(bodyStr) > 0 && bodyStr[0] == '[' {
		var tracks []domain.ServerLyricTrack
		if err := json.Unmarshal(body, &tracks); err != nil {
			return lr, err
		}
		for _, t := range tracks {
			lt := domain.LyricsTrack{Language: t.Language, TrackName: t.TrackName, ArtistName: t.ArtistName}
			for _, line := range t.Lines {
				lt.Synced = append(lt.Synced, domain.LyricLine{Time: line.Start, End: line.End, Text: line.Text})
			}
			lr.AllTracks = append(lr.AllTracks, lt)
		}
		if len(lr.AllTracks) > 0 {
			lr.SelectLanguage(0)
		}
		return lr, nil
	}
	json.Unmarshal(body, &lr)
	lr.Normalize()
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
	dest := filepath.Join(destDir, safe+".opus")

	resp, err := c.getStream(ctx, "/api/v1/stream", url.Values{"id": {id}})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("server %s", resp.Status)
	}
	f, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(dest)
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
