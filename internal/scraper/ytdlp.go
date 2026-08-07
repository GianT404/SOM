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

type YtdlpScraper struct {
	// BinPath is the absolute path to the yt-dlp binary.
	BinPath string
}
type lockEntry struct {
	mu  sync.Mutex
	ref int
}

type keyedLock struct {
	mu    sync.Mutex
	locks map[string]*lockEntry
}

func newKeyedLock() *keyedLock {
	return &keyedLock{locks: make(map[string]*lockEntry)}
}

func (k *keyedLock) Lock(key string) func() {
	k.mu.Lock()
	e, ok := k.locks[key]
	if !ok {
		e = &lockEntry{}
		k.locks[key] = e
	}
	e.ref++
	k.mu.Unlock()

	e.mu.Lock()

	return func() {
		e.mu.Unlock()
		k.mu.Lock()
		e.ref--
		if e.ref == 0 {
			delete(k.locks, key)
		}
		k.mu.Unlock()
	}
}

var downloadLocks = newKeyedLock()

// ytdlpSem giới hạn số process yt-dlp chạy đồng thời để bảo vệ VM 1 CPU và
// tránh vượt pids_limit. Request vượt slot sẽ chờ tới khi có slot hoặc khi
// context bị hủy/timeout.
var ytdlpSem = make(chan struct{}, 2)

func acquireYtdlp(ctx context.Context) error {
	select {
	case ytdlpSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseYtdlp() {
	<-ytdlpSem
}

// maxCachedAudioFiles giới hạn số file opus cache trong temp dir.
// Quá ngưỡng thì evict các file lâu không dùng nhất (LRU theo modtime),
// tránh bị bơm đầy disk khi ai đó spam /stream với nhiều videoID độc nhất.
const maxCachedAudioFiles = 50

// cleanup chạy định kỳ suốt vòng đời process, thay vì chỉ chạy 1 lần.
func init() {
	go func() {
		cleanupStaleTempFiles()
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			cleanupStaleTempFiles()
		}
	}()
}

func cleanupStaleTempFiles() {
	matches, _ := filepath.Glob(filepath.Join(os.TempDir(), "dopus*.opus"))
	now := time.Now()

	type fileEntry struct {
		path string
		mod  time.Time
	}
	var fresh []fileEntry
	for _, f := range matches {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		// File cũ hơn 1h: xóa hẳn (cache không còn giá trị).
		if now.Sub(info.ModTime()) > 1*time.Hour {
			os.Remove(f)
			continue
		}
		fresh = append(fresh, fileEntry{path: f, mod: info.ModTime()})
	}

	// Vẫn vượt ngưỡng: evict các file cũ nhất tới khi đủ dưới cap.
	if len(fresh) > maxCachedAudioFiles {
		sort.Slice(fresh, func(i, j int) bool {
			return fresh[i].mod.Before(fresh[j].mod)
		})
		for _, e := range fresh[:len(fresh)-maxCachedAudioFiles] {
			os.Remove(e.path)
		}
	}

	// Subtitle files (dopus<id>.*.vtt) chỉ dùng nhất thời trong lúc parse;
	// nếu process bị kill giữa chừng chúng thành rác vĩnh viễn. File đang
	// được xử lý luôn mới (< 10 phút) nên xóa bản cũ hơn là an toàn.
	vttMatches, _ := filepath.Glob(filepath.Join(os.TempDir(), "dopus*.vtt"))
	for _, f := range vttMatches {
		info, err := os.Stat(f)
		if err == nil && now.Sub(info.ModTime()) > 10*time.Minute {
			os.Remove(f)
		}
	}

	// Cache bản clean (?clean=1): file .ogg khá lớn, evict file cũ hơn 6h
	// (khớp vòng đời URL chữ ký) và giới hạn ~20 file.
	const maxCleanFiles = 20
	cleanMatches, _ := filepath.Glob(filepath.Join(os.TempDir(), "dopus-clean-*.ogg"))
	var cleanFresh []fileEntry
	for _, f := range cleanMatches {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) > 6*time.Hour {
			os.Remove(f)
			continue
		}
		cleanFresh = append(cleanFresh, fileEntry{path: f, mod: info.ModTime()})
	}
	if len(cleanFresh) > maxCleanFiles {
		sort.Slice(cleanFresh, func(i, j int) bool {
			return cleanFresh[i].mod.Before(cleanFresh[j].mod)
		})
		for _, e := range cleanFresh[:len(cleanFresh)-maxCleanFiles] {
			os.Remove(e.path)
		}
	}

	// Dọn .part còn sót nếu transcode bị kill giữa chừng.
	partMatches, _ := filepath.Glob(filepath.Join(os.TempDir(), "dopus-clean-*.part"))
	for _, f := range partMatches {
		info, err := os.Stat(f)
		if err == nil && now.Sub(info.ModTime()) > 1*time.Hour {
			os.Remove(f)
		}
	}
}

func (y *YtdlpScraper) DownloadAudio(ctx context.Context, videoID string) (string, error) {
	unlock := downloadLocks.Lock(videoID)
	defer unlock()

	if err := acquireYtdlp(ctx); err != nil {
		return "", err
	}
	defer releaseYtdlp()

	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("dopus%s.opus", videoID))

	if isOpusFile(tempFile) {
		return tempFile, nil
	}
	// Sắp tạo file mới: trước tiên dọn bớt nếu đã vượt cap.
	cleanupStaleTempFiles()
	_ = os.Remove(tempFile)
	cmd := exec.CommandContext(ctx, y.BinPath,
		"-f", "bestaudio",
		"-x", "--audio-format", "opus",
		"--embed-metadata",
		"-o", tempFile,
		"--no-warnings",
		"--no-playlist",
		"--no-part",
		"--force-ipv4",
		"--", "https://www.youtube.com/watch?v="+videoID,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = nil

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("yt-dlp download: %w: %s", err, stderr.String())
	}
	if !isOpusFile(tempFile) {
		_ = os.Remove(tempFile)
		return "", fmt.Errorf("yt-dlp download: output is not a playable opus file")
	}

	return tempFile, nil
}

func NewYtdlpScraper(binPath string) *YtdlpScraper {
	if binPath == "" {
		binPath = "yt-dlp"
	}

	resolved, err := exec.LookPath(binPath)
	if err == nil || errors.Is(err, exec.ErrDot) {
		if abs, err2 := filepath.Abs(resolved); err2 == nil {
			binPath = abs
		}
	}

	return &YtdlpScraper{BinPath: binPath}
}

type ytdlpSearchItem struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Thumbnail  string  `json:"thumbnail"`
	Duration   float64 `json:"duration"`
	Uploader   string  `json:"uploader"`
	URL        string  `json:"url"`
	Thumbnails []struct {
		URL string `json:"url"`
	} `json:"thumbnails"`
}

// Search runs yt-dlp to find 7 results for the keyword.
func (y *YtdlpScraper) Search(ctx context.Context, keyword string) ([]SearchResult, error) {
	if err := acquireYtdlp(ctx); err != nil {
		return nil, err
	}
	defer releaseYtdlp()

	query := fmt.Sprintf("ytsearch7:%s", keyword)
	cmd := exec.CommandContext(ctx, y.BinPath,
		query,
		"--dump-json",
		"--flat-playlist",
		"--no-warnings",
		// Chạy IPv4 ổn định + timeout mỗi socket để không kẹt vô hạn khi
		// host có IPv6 route lỗi (search không được nhanh cũng phải fail
		// sớm chứ không kéo dài tới timeout của app).
		"--force-ipv4",
		"--socket-timeout", "15",
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

func (y *YtdlpScraper) GetStreamInfo(ctx context.Context, videoID string) (*StreamInfo, error) {
	if err := acquireYtdlp(ctx); err != nil {
		return nil, err
	}
	defer releaseYtdlp()

	// Một lần gọi duy nhất lấy cả title lẫn URL trực tiếp (trước đây spawn 2
	// process song song, nhân đôi thời gian + pid trên VM 1 CPU).
	cmd := exec.CommandContext(ctx, y.BinPath,
		"--print", "%(title)s\t%(url)s",
		"-f", "bestaudio",
		"--no-warnings",
		"--no-playlist",
		"--force-ipv4",
		"--", "https://www.youtube.com/watch?v="+videoID,
	)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("yt-dlp stream info: %w: %s", err, stderr.String())
	}

	line := strings.TrimSpace(out.String())
	title, audioURL, found := strings.Cut(line, "\t")
	if !found || audioURL == "" {
		return nil, fmt.Errorf("yt-dlp returned empty URL for %s", videoID)
	}

	// yt-dlp có thể in nhiều dòng (video+audio); lấy dòng đầu.
	if idx := strings.Index(audioURL, "\n"); idx > 0 {
		audioURL = audioURL[:idx]
	}
	if title == "" || title == "NA" {
		title = videoID // fallback to video ID
	}

	return &StreamInfo{
		URL:   audioURL,
		Title: title,
	}, nil
}

func isOpusFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	header := make([]byte, 4)
	n, err := file.Read(header)
	if err != nil || n < len(header) {
		return false
	}

	return string(header) == "OggS"
}

// VideoTitle returns the title of the video.
func (y *YtdlpScraper) VideoTitle(ctx context.Context, videoID string) (string, error) {
	if err := acquireYtdlp(ctx); err != nil {
		return "", err
	}
	defer releaseYtdlp()

	cmd := exec.CommandContext(ctx, y.BinPath,
		"--print", "%(title)s",
		"--no-warnings",
		"--no-playlist",
		"--skip-download",
		"--force-ipv4",
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
	if err := acquireYtdlp(ctx); err != nil {
		return MusicMetadata{}, err
	}
	defer releaseYtdlp()

	metaCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	cmd := exec.CommandContext(metaCtx, y.BinPath,
		"--print", "%(track)s|||%(artist)s",
		"--no-warnings",
		"--no-playlist",
		"--skip-download",
		"--force-ipv4",
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

	// yt-dlp in ra literal "NA" khi field không tồn tại, phải lọc bỏ
	track := cleanYtdlpNA(parts[0])
	artist := cleanYtdlpNA(parts[1])

	return MusicMetadata{Track: track, Artist: artist}, nil
}

// cleanYtdlpNA trả về "" nếu yt-dlp in ra placeholder "NA"
func cleanYtdlpNA(s string) string {
	s = strings.TrimSpace(s)
	if s == "NA" {
		return ""
	}
	return s
}

func (y *YtdlpScraper) Lyrics(ctx context.Context, videoID string) ([]LyricsData, error) {
	origLang := detectVideoLanguage(ctx, y.BinPath, videoID)

	directCtx, directCancel := context.WithTimeout(ctx, 4*time.Second)
	directData, err := GetDirectSubtitles(directCtx, videoID, origLang)
	directCancel()
	if err == nil && len(directData) > 0 {
		return directData, nil
	}

	// Fallback to yt-dlp if direct fetch fails.
	if err := acquireYtdlp(ctx); err != nil {
		return nil, err
	}
	defer releaseYtdlp()

	ytdlpCtx, ytdlpCancel := context.WithTimeout(ctx, 6*time.Second)
	defer ytdlpCancel()

	tmpDir := os.TempDir()
	outTmpl := filepath.Join(tmpDir, fmt.Sprintf("dopus%s", videoID))

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
		"--sub-langs", "all",

		"--socket-timeout", "20",
		"-o", outTmpl,
		"--no-warnings",
		"--no-playlist",
		"--force-ipv4",
		"--", "https://www.youtube.com/watch?v="+videoID,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	dlErr := cmd.Run()
	if ytdlpCtx.Err() != nil {
		dlErr = fmt.Errorf("yt-dlp timed out after 6s: %w", ytdlpCtx.Err())
	}

	vttFiles, _ := filepath.Glob(outTmpl + "*.vtt")
	if len(vttFiles) == 0 {
		return nil, fmt.Errorf("no subtitles available for %s (err: %v): %s", videoID, dlErr, stderr.String())
	}

	sort.Strings(vttFiles)

	type variant struct {
		path  string
		score int
	}
	bestByLang := make(map[string]variant)
	for _, f := range vttFiles {
		base := filepath.Base(f)
		parts := strings.Split(base, ".")
		if len(parts) < 2 {
			continue
		}
		code := strings.TrimSuffix(parts[len(parts)-2], "-orig")
		isOrig := strings.HasSuffix(parts[len(parts)-2], "-orig")

		score := 3
		if origLang != "" && code == origLang && isOrig {
			score = 0
		} else if origLang != "" && code == origLang {
			score = 1
		} else if isOrig {
			score = 2
		}

		if cur, ok := bestByLang[code]; !ok || score < cur.score {
			bestByLang[code] = variant{path: f, score: score}
		}
	}

	codes := make([]string, 0, len(bestByLang))
	for code := range bestByLang {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	if origLang != "" {
		for i, c := range codes {
			if c == origLang {
				codes = append(codes[:i:i], codes[i+1:]...)
				codes = append([]string{origLang}, codes...)
				break
			}
		}
	}

	var result []LyricsData
	for _, code := range codes {
		b, err := os.ReadFile(bestByLang[code].path)
		if err != nil {
			continue
		}
		lines := ParseVTT(string(b))
		if len(lines) == 0 {
			continue
		}
		result = append(result, LyricsData{
			Language: code,
			Lines:    lines,
		})
	}

	for _, f := range vttFiles {
		_ = os.Remove(f)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid subtitle lines found for %s", videoID)
	}

	return result, nil
}
func detectVideoLanguage(ctx context.Context, binPath, videoID string) string {
	if err := acquireYtdlp(ctx); err != nil {
		return ""
	}
	defer releaseYtdlp()

	langCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(langCtx, binPath,
		"--print", "%(original_language)s",
		"--skip-download",
		"--no-warnings",
		"--no-playlist",
		"--force-ipv4",
		"--", "https://www.youtube.com/watch?v="+videoID,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	if cmd.Run() == nil {
		return strings.TrimSpace(out.String())
	}
	return ""
}
