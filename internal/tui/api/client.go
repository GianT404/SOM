package api

import (
	"bytes"
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

// ─── Types ───────────────────────────────────────────────────────────────────

type Track struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Duration  int    `json:"duration"`
	Thumbnail string `json:"thumbnail"`
}

type LyricLine struct {
	Time float64 `json:"time"`
	End  float64 `json:"end"`
	Text string  `json:"text"`
}
type LyricsResp struct {
	Synced []LyricLine `json:"synced"`
	Plain  string      `json:"plain"`
	Lrclib *struct {
		Synced []LyricLine `json:"synced"`
	} `json:"lrclib,omitempty"`

	Artist string `json:"artist,omitempty"`
	Title  string `json:"title,omitempty"`
}

func (l *LyricsResp) Normalize() {
	if len(l.Synced) == 0 && l.Lrclib != nil && len(l.Lrclib.Synced) > 0 {
		l.Synced = l.Lrclib.Synced
	}
}

type ServerLyricTrack struct {
	Language string `json:"language"`
	Lines    []struct {
		Start float64 `json:"start"`
		End   float64 `json:"end"`
		Text  string  `json:"text"`
	} `json:"lines"`
	TrackName  string `json:"trackName"`
	ArtistName string `json:"artistName"`
}

// ─── Client ──────────────────────────────────────────────────────────────────

type Client struct {
	base   string
	short  *http.Client
	stream *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		base:   strings.TrimRight(baseURL, "/"),
		short:  &http.Client{Timeout: 30 * time.Second},
		stream: &http.Client{Timeout: 0},
	}
}

func (c *Client) getShort(path string, params url.Values) (*http.Response, error) {
	return c.doGet(c.short, path, params)
}

func (c *Client) getStream(path string, params url.Values) (*http.Response, error) {
	return c.doGet(c.stream, path, params)
}

func (c *Client) doGet(hc *http.Client, path string, params url.Values) (*http.Response, error) {
	u := c.base + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	return hc.Get(u)
}

// ─── Search ───────────────────────────────────────────────────────────────────

func (c *Client) Search(q string) ([]Track, error) {
	resp, err := c.getShort("/api/v1/search", url.Values{"q": {q}})
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search: server %s — %s", resp.Status, truncateMsg(body))
	}

	var tracks []Track
	if err := json.Unmarshal(body, &tracks); err != nil {
		return nil, fmt.Errorf("search decode: %w\nbody: %s", err, truncateMsg(body))
	}
	return tracks, nil
}

// ─── Stream ───────────────────────────────────────────────────────────────────
func (c *Client) StreamURL(id string) string {
	return fmt.Sprintf("%s/api/v1/stream?id=%s", c.base, url.QueryEscape(id))
}

// ─── Lyrics ──────────────────────────────────────────────────────────────────

func (c *Client) Lyrics(id, title, artist string, duration int) (LyricsResp, error) {
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
	resp, err := c.getShort("/api/v1/lyrics", params)
	if err != nil {
		return LyricsResp{}, fmt.Errorf("lyrics request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return LyricsResp{}, fmt.Errorf("lyrics: server %s — %s", resp.Status, truncateMsg(body))
	}

	var lr LyricsResp
	bodyStr := bytes.TrimSpace(body)
	if len(bodyStr) > 0 && bodyStr[0] == '[' {
		var tracks []ServerLyricTrack
		if err := json.Unmarshal(body, &tracks); err != nil {
			return lr, fmt.Errorf("lyrics array decode error: %w", err)
		}

		if len(tracks) > 0 {
			var selected *ServerLyricTrack
			for i := range tracks {
				if tracks[i].Language == "vi" {
					selected = &tracks[i]
					break
				}
			}
			if selected == nil {
				selected = &tracks[0]
			}
			for _, line := range selected.Lines {
				lr.Synced = append(lr.Synced, LyricLine{
					Time: line.Start,
					End:  line.End,
					Text: line.Text,
				})
			}
			if selected.TrackName != "" {
				lr.Title = selected.TrackName
			}
			if selected.ArtistName != "" {
				lr.Artist = selected.ArtistName
			}
		}
		return lr, nil
	}

	if err := json.Unmarshal(body, &lr); err != nil {
		return LyricsResp{}, fmt.Errorf("lyrics decode: %w\nbody: %s", err, truncateMsg(body))
	}
	lr.Normalize()
	return lr, nil
}

// ─── Download ─────────────────────────────────────────────────────────────────
func (c *Client) DownloadM4A(id, title, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", destDir, err)
	}
	safe := sanitize(title)
	if safe == "" {
		safe = id
	}
	dest := filepath.Join(destDir, safe+".m4a")

	resp, err := c.getStream("/api/v1/stream", url.Values{"id": {id}})
	if err != nil {
		return "", fmt.Errorf("download request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download: server %s", resp.Status)
	}

	f, err := os.Create(dest)
	if err != nil {
		return "", fmt.Errorf("create %s: %w", dest, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(dest)
		return "", fmt.Errorf("write: %w", err)
	}
	return dest, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func sanitize(s string) string {
	r := strings.NewReplacer(
		"/", "-", "\\", "-", ":", "-", "*", "-",
		"?", "-", `"`, "-", "<", "-", ">", "-", "|", "-",
	)
	return strings.TrimSpace(r.Replace(s))
}

func truncateMsg(b []byte) string {
	if len(b) > 200 {
		return string(b[:200]) + "…"
	}
	return string(b)
}
