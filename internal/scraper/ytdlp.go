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
	"sort"
	"strings"
	"sync"
	"time"
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
				Artist:    item.Uploader,
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

var cleanupOnce sync.Once

// cleanupStaleTempFiles removes dm4a_*.m4a temp files older than 1 hour.
func cleanupStaleTempFiles() {
	matches, _ := filepath.Glob(filepath.Join(os.TempDir(), "dm4a_*.m4a"))
	now := time.Now()
	for _, f := range matches {
		info, err := os.Stat(f)
		if err == nil && now.Sub(info.ModTime()) > 1*time.Hour {
			os.Remove(f)
		}
	}
}

func (y *YtdlpScraper) DownloadAudio(ctx context.Context, videoID string) (string, error) {
	cleanupOnce.Do(cleanupStaleTempFiles)
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("dm4a_%s.m4a", videoID))

	if isM4AFile(tempFile) {
		return tempFile, nil
	}
	_ = os.Remove(tempFile)

	// Try direct download to bypass yt-dlp
	err := GetDirectAudio(ctx, videoID, tempFile)
	if err == nil {
		return tempFile, nil
	}

	cmd := exec.CommandContext(ctx, y.BinPath,
		"-f", "ba[ext=m4a]",
		"-o", tempFile,
		"--no-warnings",
		"--no-check-certificates",
		"--no-playlist",
		"--no-part",
		"--", "https://www.youtube.com/watch?v="+videoID,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = nil

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("yt-dlp download: %w: %s", err, stderr.String())
	}
	if !isM4AFile(tempFile) {
		_ = os.Remove(tempFile)
		return "", fmt.Errorf("yt-dlp download: output is not a playable m4a file")
	}

	return tempFile, nil
}

func isM4AFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	header := make([]byte, 12)
	n, err := file.Read(header)
	if err != nil || n < len(header) {
		return false
	}

	return string(header[4:8]) == "ftyp"
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

func (y *YtdlpScraper) VideoMetadata(ctx context.Context, videoID string) (MusicMetadata, error) {
	metaCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	cmd := exec.CommandContext(metaCtx, y.BinPath,
		"--print", "%(track)s|||%(artist)s",
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
		return MusicMetadata{}, fmt.Errorf("yt-dlp metadata: %w: %s", err, stderr.String())
	}

	out := strings.TrimSpace(stdout.String())
	parts := strings.SplitN(out, "|||", 2)
	if len(parts) != 2 {
		return MusicMetadata{}, nil
	}

	track := strings.TrimSpace(parts[0])
	artist := strings.TrimSpace(parts[1])

	return MusicMetadata{Track: track, Artist: artist}, nil
}

func (y *YtdlpScraper) Lyrics(ctx context.Context, videoID string) ([]LyricsData, error) {
	origLang := detectVideoLanguage(ctx, y.BinPath, videoID)

	directCtx, directCancel := context.WithTimeout(ctx, 4*time.Second)
	directData, err := GetDirectSubtitles(directCtx, videoID, origLang)
	directCancel()
	if err == nil && len(directData) > 0 {
		return directData, nil
	}
	ytdlpCtx, ytdlpCancel := context.WithTimeout(ctx, 6*time.Second)
	defer ytdlpCancel()

	tmpDir := os.TempDir()
	outTmpl := filepath.Join(tmpDir, fmt.Sprintf("dm4a_subs_%s", videoID))

	// Clean up any old files for this video
	matches, _ := filepath.Glob(outTmpl + "*")
	for _, f := range matches {
		_ = os.Remove(f)
	}

	cmd := exec.CommandContext(ytdlpCtx, y.BinPath,
		"--write-subs",
		"--write-auto-subs",
		"--skip-download",
		"--sub-format", "vtt",
		"--sub-langs", "en,vi,ja,ko,zh-Hans,zh-Hant",
		"--socket-timeout", "5",
		"-o", outTmpl,
		"--no-warnings",
		"--no-check-certificates",
		"--no-playlist",
		"--", "https://www.youtube.com/watch?v="+videoID,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	dlErr := cmd.Run()
	if ytdlpCtx.Err() != nil {
		dlErr = fmt.Errorf("yt-dlp timed out after 6s: %w", ytdlpCtx.Err())
	}

	// Find all downloaded VTT files
	vttFiles, _ := filepath.Glob(outTmpl + "*.vtt")
	if len(vttFiles) == 0 {
		return nil, fmt.Errorf("no subtitles available for %s (err: %v): %s", videoID, dlErr, stderr.String())
	}

	// Sort to guarantee deterministic ordering
	sort.Strings(vttFiles)

	type scoredFile struct {
		path  string
		lang  string
		score int
	}
	var scoredFiles []scoredFile
	seenLang := map[string]bool{}
	for _, f := range vttFiles {
		base := filepath.Base(f)
		parts := strings.Split(base, ".")
		if len(parts) < 2 {
			continue
		}
		code := strings.TrimSuffix(parts[len(parts)-2], "-orig")
		isOrig := strings.HasSuffix(parts[len(parts)-2], "-orig")

		if seenLang[code] {
			// Already have a track for this language (e.g. both a manual
			// and an auto-generated file for the same code) — keep only
			// the first (higher-priority) one encountered.
			continue
		}

		score := 99
		if origLang != "" && code == origLang && isOrig {
			score = 0
		} else if origLang != "" && code == origLang {
			score = 1
		} else if isOrig {
			score = 2
		} else {
			score = 3
		}

		scoredFiles = append(scoredFiles, scoredFile{path: f, lang: code, score: score})
		seenLang[code] = true
	}

	sort.SliceStable(scoredFiles, func(i, j int) bool {
		return scoredFiles[i].score < scoredFiles[j].score
	})

	var result []LyricsData
	for _, sf := range scoredFiles {
		b, err := os.ReadFile(sf.path)
		if err != nil {
			continue
		}
		lines := ParseVTT(string(b))
		if len(lines) == 0 {
			continue
		}
		result = append(result, LyricsData{Language: sf.lang, Lines: lines})
	}

	// Cleanup all generated subtitle files
	for _, f := range vttFiles {
		_ = os.Remove(f)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid subtitle lines found for %s", videoID)
	}

	return result, nil
}

// detectVideoLanguage runs a fast yt-dlp --print to detect the video's original language.
func detectVideoLanguage(ctx context.Context, binPath, videoID string) string {
	langCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(langCtx, binPath,
		"--print", "%(original_language)s",
		"--skip-download",
		"--no-warnings",
		"--no-check-certificates",
		"--no-playlist",
		"--", "https://www.youtube.com/watch?v="+videoID,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	if cmd.Run() == nil {
		return strings.TrimSpace(out.String())
	}
	return ""
}
