// internal/api/client.go
// Wraps the SOM Go HTTP backend:
//
//	GET /api/v1/search?q=
//	GET /api/v1/stream?id=
//	GET /api/v1/lyrics?id=
//	GET /api/v1/resolve?id=
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ─── Types matching SOM backend JSON ────────────────────────────────────────

type Track struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Duration  int    `json:"duration"` // seconds
	Thumbnail string `json:"thumbnail"`
}

type LyricLine struct {
	Time float64 `json:"time"` // seconds
	Text string  `json:"text"`
}

type LyricsResp struct {
	Synced []LyricLine `json:"synced"`
	Plain  string      `json:"plain"`
}

type ResolveResp struct {
	URL      string `json:"url"`
	MimeType string `json:"mimeType"`
}

// ─── Client ─────────────────────────────────────────────────────────────────

type Client struct {
	base string
	http *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		base: strings.TrimRight(baseURL, "/"),
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) get(path string, params url.Values) (*http.Response, error) {
	u := c.base + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	return c.http.Get(u)
}

// Search queries YouTube via SOM backend and returns a list of tracks.
func (c *Client) Search(q string) ([]Track, error) {
	resp, err := c.get("/api/v1/search", url.Values{"q": {q}})
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search: server returned %s", resp.Status)
	}
	var tracks []Track
	if err := json.NewDecoder(resp.Body).Decode(&tracks); err != nil {
		return nil, fmt.Errorf("search decode: %w", err)
	}
	return tracks, nil
}

// StreamURL returns the direct audio stream URL for a YouTube video ID.
// The TUI passes this to mpv / ffplay.
func (c *Client) StreamURL(id string) string {
	return fmt.Sprintf("%s/api/v1/stream?id=%s", c.base, url.QueryEscape(id))
}

// Resolve fetches the underlying audio URL (for download).
func (c *Client) Resolve(id string) (ResolveResp, error) {
	resp, err := c.get("/api/v1/resolve", url.Values{"id": {id}})
	if err != nil {
		return ResolveResp{}, fmt.Errorf("resolve request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ResolveResp{}, fmt.Errorf("resolve: server returned %s", resp.Status)
	}
	var r ResolveResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return ResolveResp{}, fmt.Errorf("resolve decode: %w", err)
	}
	return r, nil
}

// Lyrics fetches synced (LRC) lyrics for a track ID.
func (c *Client) Lyrics(id string) (LyricsResp, error) {
	resp, err := c.get("/api/v1/lyrics", url.Values{"id": {id}})
	if err != nil {
		return LyricsResp{}, fmt.Errorf("lyrics request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return LyricsResp{}, fmt.Errorf("lyrics: server returned %s", resp.Status)
	}
	var lr LyricsResp
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return LyricsResp{}, fmt.Errorf("lyrics decode: %w", err)
	}
	return lr, nil
}

// DownloadM4A streams audio from /api/v1/stream and saves it as a .m4a file.
// destDir is the folder to save into; filename is auto-generated from title.
// Returns the saved path and a progress channel (bytes written so far).
func (c *Client) DownloadM4A(id, title, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}

	// Sanitize filename
	safe := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-",
		"?", "-", "\"", "-", "<", "-", ">", "-", "|", "-").Replace(title)
	if safe == "" {
		safe = id
	}
	dest := filepath.Join(destDir, safe+".m4a")

	resp, err := c.get("/api/v1/stream", url.Values{"id": {id}})
	if err != nil {
		return "", fmt.Errorf("download stream: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download: server returned %s", resp.Status)
	}

	f, err := os.Create(dest)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(dest)
		return "", fmt.Errorf("write file: %w", err)
	}
	return dest, nil
}
