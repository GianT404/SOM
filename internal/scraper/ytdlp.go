package scraper

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// YtdlpScraper implements Scraper using the yt-dlp CLI tool.
type YtdlpScraper struct {
	// BinPath is the absolute path to the yt-dlp binary.
	BinPath string
}

// NewYtdlpScraper creates a new scraper with the given binary path.
// It resolves the binary to an absolute path to avoid Go 1.19+ security
// restrictions on running executables found relative to the current directory.
func NewYtdlpScraper(binPath string) *YtdlpScraper {
	if binPath == "" {
		binPath = "yt-dlp"
	}

	// Resolve to absolute path to satisfy Go's exec security policy.
	// exec.LookPath returns ErrDot when the binary is in the current directory;
	// we must handle that case too, not just err == nil.
	resolved, err := exec.LookPath(binPath)
	if err == nil || errors.Is(err, exec.ErrDot) {
		if abs, err2 := filepath.Abs(resolved); err2 == nil {
			binPath = abs
		}
	}

	return &YtdlpScraper{BinPath: binPath}
}

// ytdlpSearchItem is the raw JSON shape emitted by yt-dlp --dump-json --flat-playlist.
type ytdlpSearchItem struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Thumbnail string  `json:"thumbnail"`
	Duration  float64 `json:"duration"`
	Uploader  string  `json:"uploader"`
	URL       string  `json:"url"`

	// Thumbnails array fallback (flat-playlist sometimes uses this instead).
	Thumbnails []struct {
		URL string `json:"url"`
	} `json:"thumbnails"`
}

// Search runs yt-dlp to find 7 results for the keyword.
func (y *YtdlpScraper) Search(ctx context.Context, keyword string) ([]SearchResult, error) {
	query := fmt.Sprintf("ytsearch7:%s", keyword)
	cmd := exec.CommandContext(ctx, y.BinPath,
		query,
		"--dump-json",
		"--flat-playlist",
		"--no-warnings",
		"--no-check-certificates",
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start yt-dlp: %w", err)
	}

	// Read results from stdout line by line (one JSON object per line).
	resultCh := make(chan SearchResult, 7)
	errCh := make(chan error, 1)

	go func() {
		defer close(resultCh)
		scanner := bufio.NewScanner(stdout)
		// Allow up to 1 MB per line (some JSON can be big).
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			var item ytdlpSearchItem
			if err := json.Unmarshal(scanner.Bytes(), &item); err != nil {
				continue // skip malformed lines
			}

			thumb := item.Thumbnail
			if thumb == "" && len(item.Thumbnails) > 0 {
				thumb = item.Thumbnails[len(item.Thumbnails)-1].URL
			}

			resultCh <- SearchResult{
				ID:        item.ID,
				Title:     item.Title,
				Thumbnail: thumb,
				Duration:  int(item.Duration),
				Uploader:  item.Uploader,
			}
		}
		if err := scanner.Err(); err != nil {
			errCh <- fmt.Errorf("scan stdout: %w", err)
		}
	}()

	var results []SearchResult
	for r := range resultCh {
		results = append(results, r)
	}

	if err := cmd.Wait(); err != nil {
		return results, fmt.Errorf("yt-dlp search: %w: %s", err, stderr.String())
	}

	select {
	case e := <-errCh:
		return results, e
	default:
	}

	return results, nil
}

// GetStreamInfo returns the direct audio URL and video title for the given video ID.
// Uses --print to get the title and -g to get the URL in one call.
func (y *YtdlpScraper) GetStreamInfo(ctx context.Context, videoID string) (*StreamInfo, error) {
	// First get the title.
	titleCmd := exec.CommandContext(ctx, y.BinPath,
		"--print", "%(title)s",
		"-f", "ba[ext=m4a]/ba",
		"--no-warnings",
		"--no-check-certificates",
		"--no-playlist",
		"--", "https://www.youtube.com/watch?v="+videoID,
	)

	var titleOut, titleErr bytes.Buffer
	titleCmd.Stdout = &titleOut
	titleCmd.Stderr = &titleErr

	// Get the direct URL.
	urlCmd := exec.CommandContext(ctx, y.BinPath,
		"-g",
		"-f", "ba[ext=m4a]/ba",
		"--no-warnings",
		"--no-check-certificates",
		"--no-playlist",
		"--", "https://www.youtube.com/watch?v="+videoID,
	)

	var urlOut, urlErr bytes.Buffer
	urlCmd.Stdout = &urlOut
	urlCmd.Stderr = &urlErr

	// Run both in parallel via goroutines.
	type cmdResult struct {
		output string
		err    error
	}

	titleCh := make(chan cmdResult, 1)
	urlCh := make(chan cmdResult, 1)

	go func() {
		err := titleCmd.Run()
		titleCh <- cmdResult{strings.TrimSpace(titleOut.String()), err}
	}()

	go func() {
		err := urlCmd.Run()
		urlCh <- cmdResult{strings.TrimSpace(urlOut.String()), err}
	}()

	urlResult := <-urlCh
	if urlResult.err != nil {
		return nil, fmt.Errorf("yt-dlp stream URL: %w: %s", urlResult.err, urlErr.String())
	}

	audioURL := urlResult.output
	if audioURL == "" {
		return nil, fmt.Errorf("yt-dlp returned empty URL for %s", videoID)
	}

	// yt-dlp may return multiple lines (video+audio); take the first.
	if idx := strings.Index(audioURL, "\n"); idx > 0 {
		audioURL = audioURL[:idx]
	}

	titleResult := <-titleCh
	title := titleResult.output
	if title == "" {
		title = videoID // fallback to video ID
	}

	return &StreamInfo{
		URL:   audioURL,
		Title: title,
	}, nil
}

// DownloadAudio downloads audio to a temporary file and returns its path.
// yt-dlp handles YouTube's n-parameter throttle decryption internally,
// so this gives full-speed downloads unlike raw URL fetching.
func (y *YtdlpScraper) DownloadAudio(ctx context.Context, videoID string) (string, error) {
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("dm4a_%s.m4a", videoID))

	// If it already exists and is fully downloaded, reuse it.
	if info, err := os.Stat(tempFile); err == nil && info.Size() > 0 {
		return tempFile, nil
	}

	// Try direct download to bypass yt-dlp
	err := GetDirectAudio(ctx, videoID, tempFile)
	if err == nil {
		return tempFile, nil
	}
	// Fallback to yt-dlp below...
	cmd := exec.CommandContext(ctx, y.BinPath,
		"-f", "ba[ext=m4a]/ba",
		"-o", tempFile,
		"--no-warnings",
		"--no-check-certificates",
		"--no-playlist",
		"--no-part",
		"--", "https://www.youtube.com/watch?v="+videoID,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	// Ignore stdout for clean logs
	cmd.Stdout = nil

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("yt-dlp download: %w: %s", err, stderr.String())
	}

	return tempFile, nil
}

// VideoTitle returns the title of the video.
func (y *YtdlpScraper) VideoTitle(ctx context.Context, videoID string) (string, error) {
	cmd := exec.CommandContext(ctx, y.BinPath,
		"--print", "%(title)s",
		"--no-warnings",
		"--no-check-certificates",
		"--no-playlist",
		"--skip-download",
		"--", "https://www.youtube.com/watch?v="+videoID,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("yt-dlp title: %w: %s", err, stderr.String())
	}

	title := strings.TrimSpace(stdout.String())
	if title == "" {
		title = videoID
	}
	return title, nil
}

// Lyrics fetches subtitles and parses them into LyricLines.
// It attempts to use direct Innertube fetching first to prevent IP blocking.
// If direct fetching fails, it falls back to yt-dlp downloads.
func (y *YtdlpScraper) Lyrics(ctx context.Context, videoID string) ([]LyricsData, error) {
	// Try the new direct approach first
	directData, err := GetDirectSubtitles(ctx, videoID)
	if err == nil && len(directData) > 0 {
		return directData, nil
	}

	// Fallback to yt-dlp if direct fetch fails
	tmpDir := os.TempDir()
	outTmpl := filepath.Join(tmpDir, fmt.Sprintf("dm4a_subs_%s", videoID))

	// Clean up any old files for this video
	matches, _ := filepath.Glob(outTmpl + "*")
	for _, f := range matches {
		_ = os.Remove(f)
	}

	cmd := exec.CommandContext(ctx, y.BinPath,
		"--write-subs",
		"--write-auto-subs",
		"--skip-download",
		"--sub-format", "vtt",
		// Restrict to common languages to prevent yt-dlp 429 Too Many Requests from YT.
		"--sub-langs", "en,vi,ja,ko,zh-Hans,zh-Hant",
		"-o", outTmpl,
		"--no-warnings",
		"--no-check-certificates",
		"--no-playlist",
		"--", "https://www.youtube.com/watch?v="+videoID,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	dlErr := cmd.Run()

	// Find all downloaded VTT files
	vttFiles, _ := filepath.Glob(outTmpl + "*.vtt")
	if len(vttFiles) == 0 {
		return nil, fmt.Errorf("no subtitles available for %s (err: %v): %s", videoID, dlErr, stderr.String())
	}

	var results []LyricsData

	for _, f := range vttFiles {
		b, err := os.ReadFile(f)
		if err == nil {
			vtt := string(b)
			lines := ParseVTT(vtt)

			if len(lines) > 0 {
				// Filename format: dm4a_subs_ID.lang.vtt or dm4a_subs_ID.lang-orig.vtt
				parts := strings.Split(filepath.Base(f), ".")
				lang := "unknown"
				if len(parts) >= 3 {
					lang = parts[len(parts)-2]
				}
				// Clean up "-orig" suffix from auto-generated subs
				lang = strings.TrimSuffix(lang, "-orig")

				results = append(results, LyricsData{
					Language: lang,
					Lines:    lines,
				})
			}
		}
	}

	// Cleanup all generated subtitle files
	for _, f := range vttFiles {
		_ = os.Remove(f)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("failed to parse any subtitles for %s", videoID)
	}

	return results, nil
}
