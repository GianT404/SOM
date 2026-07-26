package scraper

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const lrclibBase = "https://lrclib.net"

// lrclibTrack is the shape of a single result from LRCLib.
type lrclibTrack struct {
	ID           int     `json:"id"`
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	AlbumName    string  `json:"albumName"`
	Duration     float64 `json:"duration"`
	Instrumental bool    `json:"instrumental"`
	SyncedLyrics string  `json:"syncedLyrics"`
	PlainLyrics  string  `json:"plainLyrics"`
}

var lrcTimestampRe = regexp.MustCompile(`^\[(\d+):(\d+)(?:[.:](\d+))?\]\s*(.*)$`)
var lrcRegex = regexp.MustCompile(`^\[(\d{2,}):(\d{2})\.(\d{2,3})\](.*)`)

func parseLRC(syncedLyrics string) []LyricLine {
	var lines []LyricLine
	scanner := bufio.NewScanner(strings.NewReader(syncedLyrics))

	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}

		matches := lrcRegex.FindStringSubmatch(text)
		if len(matches) == 5 {
			min, _ := strconv.Atoi(matches[1])
			sec, _ := strconv.Atoi(matches[2])

			msStr := matches[3]
			ms, _ := strconv.Atoi(msStr)
			if len(msStr) == 2 {
				ms *= 10
			}

			start := float64(min)*60 + float64(sec) + float64(ms)/1000.0
			content := strings.TrimSpace(matches[4])

			lines = append(lines, LyricLine{
				Start: start,
				End:   0,
				Text:  content,
			})
		}
	}

	for i := 0; i < len(lines)-1; i++ {
		lines[i].End = lines[i+1].Start

		if lines[i].End-lines[i].Start > 10.0 {
			lines[i].End = lines[i].Start + 5.0
		}
	}

	if len(lines) > 0 {
		lastIdx := len(lines) - 1
		lines[lastIdx].End = lines[lastIdx].Start + 5.0
	}

	return lines
}

var lrclibHTTPClient = &http.Client{Timeout: 6 * time.Second}

// GetLRCLibLyrics fetches synced lyrics from lrclib.net.
// It first tries the exact /api/get endpoint (requires duration),
// then falls back to /api/search if the exact match returns 404.
//
// trackName and artistName come from video metadata (title / uploader).
// durationSec is the video duration in seconds (used to pick the best result).
func GetLRCLibLyrics(ctx context.Context, trackName, artistName string, durationSec float64) ([]LyricsData, error) {
	// Cap total time spent across both strategies (get + search) so this
	// secondary feature can't stall the parent request for too long. The
	// caller (handler) also enforces its own overall deadline; this is a
	// tighter inner bound so LRCLib alone never eats the whole budget.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// ---- Strategy 1: exact match via /api/get ----
	if durationSec > 0 {
		params := url.Values{
			"track_name":  {trackName},
			"artist_name": {artistName},
			"duration":    {fmt.Sprintf("%.0f", durationSec)},
		}
		reqURL := lrclibBase + "/api/get?" + params.Encode()
		if data, err := fetchLRCLibSingle(ctx, reqURL); err == nil && data != nil {
			return data, nil
		}
	}

	// ---- Strategy 2: search fallback ----
	query := trackName
	if artistName != "" {
		query = artistName + " " + trackName
	}
	params := url.Values{"q": {query}}
	reqURL := lrclibBase + "/api/search?" + params.Encode()

	tracks, err := fetchLRCLibSearch(ctx, reqURL)
	if err != nil || len(tracks) == 0 {
		return nil, fmt.Errorf("lrclib: no results for %q", trackName)
	}

	// Pick best result: prefer one whose duration is closest to ours.
	best := tracks[0]
	if durationSec > 0 {
		bestDiff := absDiff(best.Duration, durationSec)
		for _, t := range tracks[1:] {
			if d := absDiff(t.Duration, durationSec); d < bestDiff {
				best = t
				bestDiff = d
			}
		}
	}

	return lrclibTrackToLyricsData(best), nil
}

func absDiff(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}

// fetchLRCLibSingle does a GET and decodes a single lrclibTrack.
func fetchLRCLibSingle(ctx context.Context, reqURL string) ([]LyricsData, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	req.Header.Set("Lrclib-Client", "SOM/1.0 (https://github.com/GianT404/SOM)")

	resp, err := lrclibHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // signal "not found"
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lrclib: HTTP %d from %s", resp.StatusCode, reqURL)
	}

	var t lrclibTrack
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return nil, fmt.Errorf("lrclib: decode error: %w", err)
	}

	return lrclibTrackToLyricsData(t), nil
}

// fetchLRCLibSearch does a GET and decodes an array of lrclibTrack.
func fetchLRCLibSearch(ctx context.Context, reqURL string) ([]lrclibTrack, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	req.Header.Set("Lrclib-Client", "SOM/1.0 (https://github.com/GianT404/SOM)")

	resp, err := lrclibHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lrclib search: HTTP %d", resp.StatusCode)
	}

	var tracks []lrclibTrack
	if err := json.NewDecoder(resp.Body).Decode(&tracks); err != nil {
		return nil, fmt.Errorf("lrclib search: decode error: %w", err)
	}
	return tracks, nil
}

// lrclibTrackToLyricsData converts a lrclibTrack into our internal format.
// We return only one language "lrclib" (synced) or "" (plain).
func lrclibTrackToLyricsData(t lrclibTrack) []LyricsData {
	var result []LyricsData

	if t.SyncedLyrics != "" {
		lines := parseLRC(t.SyncedLyrics)
		if len(lines) > 0 {
			result = append(result, LyricsData{
				Language:   "lrclib",
				Lines:      lines,
				TrackName:  t.TrackName,
				ArtistName: t.ArtistName,
			})
		}
	}

	// If no synced lyrics but there are plain lyrics, serve them as a single block.
	if len(result) == 0 && t.PlainLyrics != "" {
		var lines []LyricLine
		for _, l := range strings.Split(t.PlainLyrics, "\n") {
			l = strings.TrimSpace(l)
			if l != "" {
				lines = append(lines, LyricLine{Text: l})
			}
		}
		if len(lines) > 0 {
			result = append(result, LyricsData{
				Language:   "lrclib",
				Lines:      lines,
				TrackName:  t.TrackName,
				ArtistName: t.ArtistName,
			})
		}
	}

	return result
}
