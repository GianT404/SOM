package scraper

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kkdai/youtube/v2"

	"som/internal/cache"
)

type YtdlpScraper struct {
	// BinPath is the absolute path to the yt-dlp binary.
	BinPath string
	// fastPath dùng lib youtube/v2 resolve URL (nhanh hơn) thay vì spawn
	// yt-dlp. An toàn cho server vì URL đó không được dùng để phát trực tiếp.
	fastPath       bool
	fastpathMisses *cache.TTLCache[struct{}]
}

const (
	fastpathMissTTL   = 6 * time.Hour
	maxFastpathMisses = 5000
)

func (y *YtdlpScraper) markFastpathMiss(videoID string) {
	y.fastpathMisses.Put(videoID, struct{}{})
}

func (y *YtdlpScraper) isFastpathMiss(videoID string) bool {
	_, ok := y.fastpathMisses.Get(videoID)
	return ok
}

// defaultUA là UA trình duyệt dùng cho fastpath (lib youtube/v2 không trả
const defaultUA = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

var (
	clientsList     []string
	cookiesPath     string
	extraYtArgs     []string
	clientChainMu   sync.Mutex
	preferredClient string
	ytConfigOnce    sync.Once
)

func loadYtConfig() {
	ytConfigOnce.Do(func() {
		extraYtArgs = strings.Fields(os.Getenv("SOM_YTDLP_ARGS"))
		cookiesPath = strings.TrimSpace(os.Getenv("SOM_YTDLP_COOKIES"))
		if v := strings.TrimSpace(os.Getenv("SOM_YTDLP_CLIENTS")); v != "" {
			for _, c := range strings.Split(v, ",") {
				if c = strings.TrimSpace(c); c != "" {
					clientsList = append(clientsList, c)
				}
			}
		}
		if len(clientsList) == 0 {
			clientsList = []string{"web_embedded", "web", "tv_embedded", "android", "mweb", "ios"}
		}
	})
}

func nextClient(attempt int) string {
	loadYtConfig()
	clientChainMu.Lock()
	defer clientChainMu.Unlock()
	cands := make([]string, 0, len(clientsList)+1)
	if preferredClient != "" {
		cands = append(cands, preferredClient)
	}
	for _, c := range clientsList {
		if c != preferredClient {
			cands = append(cands, c)
		}
	}
	if attempt < len(cands) {
		return cands[attempt]
	}
	return ""
}

// markClientSuccess nhớ client vừa chạy thành công để lần sau thử trước.
func markClientSuccess(c string) {
	if c == "" {
		return
	}
	clientChainMu.Lock()
	defer clientChainMu.Unlock()
	preferredClient = c
}

// ytBaseArgs trả args nền tảng cho mọi lệnh yt-dlp: flags ổn định + cookies + extra args từ SOM_YTDLP_ARGS.
func ytBaseArgs() []string {
	loadYtConfig()
	args := []string{"--no-warnings", "--force-ipv4"}
	if cookiesPath != "" {
		args = append(args, "--cookies", cookiesPath)
	}
	args = append(args, extraYtArgs...)
	return args
}

// ytPreferredFlag trả --extractor-args của client đang chạy được (nếu có) để yt-dlp ưu tiên client đó, tránh bị CDN chặn 403.
func ytPreferredFlag() []string {
	loadYtConfig()
	clientChainMu.Lock()
	defer clientChainMu.Unlock()
	if preferredClient == "" {
		return nil
	}
	return []string{"--extractor-args", "youtube:player_client=" + preferredClient}
}

func ytRetryable(errMsg string) bool {
	lower := strings.ToLower(errMsg)
	markers := []string{
		"403",
		"requested format is not available",
		"sign in to confirm",
		"sign in to view",
		"not a bot",
		"not available on this app",
		"isn't available on this app",
		"only available to",
		"music premium",
		"premium members",
		"age-restricted",
		"age restricted",
		"in your country",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

const ytOutdatedHint = "yt-dlp có thể đã cũ — chạy `som --update-ytdlp` để cập nhật"

func withYtHint(err error) error {
	if err == nil {
		return nil
	}
	if ytRetryable(err.Error()) {
		return fmt.Errorf("%s (%s)", err, ytOutdatedHint)
	}
	return err
}

func UpdateYtdlp(binPath string) error {
	if binPath == "" {
		binPath = "yt-dlp"
	}
	cmd := exec.Command(binPath, "-U")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("yt-dlp -U: %w: %s", err, strings.TrimSpace(string(out)))
	}
	fmt.Print(string(out))
	return nil
}

var ytdlpCacheDir = resolveYtdlpCacheDir()

func resolveYtdlpCacheDir() string {
	dir := os.Getenv("YTDLP_CACHE_DIR")
	if dir == "" {
		dir = "/var/cache/som/yt-dlp"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("ytdlp: khong tao duoc cache dir %s: %v — dung cache mac dinh cua yt-dlp (khong persistent)", dir, err)
		return ""
	}
	return dir
}

// cacheDirArgs trả flag --cache-dir nếu thư mục cache khả dụng. Nếu không
// tạo được (permission, disk...), trả rỗng thay vì chặn cả app — yt-dlp
// tự lo cache mặc định, chỉ mất tính persistent chứ không mất chức năng.
func cacheDirArgs() []string {
	if ytdlpCacheDir == "" {
		return nil
	}
	return []string{"--cache-dir", ytdlpCacheDir}
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

var lyricsLocks = newKeyedLock()

// ytdlpSem giới hạn số process yt-dlp "nhẹ" (resolve/search/metadata/lyrics)
// chạy đồng thời. Tách riêng khỏi download nặng để một phiên chơi nhạc đang
// tải file không chặn đứng search/resolve/lyrics (trước đây 2 pool chung, mỗi
// download ~8s chiếm 1 slot → resolve & lyrics phải chờ, app timeout 10s).
var ytdlpSem = make(chan struct{}, 4)

// ytdlpDownloadSem giới hạn số luồng download nặng (DownloadAudio ~8s) cùng
// lúc. Giới hạn 2 slot → 2 phiên chơi song song OK, không lấn sang pool light.
var ytdlpDownloadSem = make(chan struct{}, 2)

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

func acquireYtdlpDownload(ctx context.Context) error {
	select {
	case ytdlpDownloadSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseYtdlpDownload() {
	<-ytdlpDownloadSem
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

	if err := acquireYtdlpDownload(ctx); err != nil {
		return "", err
	}
	defer releaseYtdlpDownload()

	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("dopus%s.opus", videoID))

	if isOpusFile(tempFile) {
		return tempFile, nil
	}
	// Sắp tạo file mới: trước tiên dọn bớt nếu đã vượt cap.
	cleanupStaleTempFiles()
	_ = os.Remove(tempFile)

	var lastErr error
	for attempt := 0; ; attempt++ {
		client := nextClient(attempt)
		if client == "" {
			if lastErr != nil {
				return "", withYtHint(lastErr)
			}
			return "", fmt.Errorf("yt-dlp download: no client worked for %s", videoID)
		}

		args := []string{
			"-f", "bestaudio",
			"-x", "--audio-format", "opus",
			"--embed-metadata",
			"-o", tempFile,
			"--no-playlist",
			"--no-part",
		}
		args = append(args, ytBaseArgs()...)
		args = append(args, "--extractor-args", "youtube:player_client="+client)
		args = append(args, cacheDirArgs()...)
		args = append(args, "--", "https://www.youtube.com/watch?v="+videoID)
		cmd := exec.CommandContext(ctx, y.BinPath, args...)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		cmd.Stdout = nil

		if err := cmd.Run(); err != nil {
			lastErr = fmt.Errorf("yt-dlp download (%s): %w: %s", client, err, stderr.String())
			_ = os.Remove(tempFile)
			if ytRetryable(lastErr.Error()) {
				continue
			}
			return "", lastErr
		}
		if !isOpusFile(tempFile) {
			_ = os.Remove(tempFile)
			return "", fmt.Errorf("yt-dlp download: output is not a playable opus file")
		}
		markClientSuccess(client)
		return tempFile, nil
	}
}

// NewYtdlpScraperFastPath giống NewYtdlpScraper nhưng bật cứng fastpath
// (lib youtube/v2) — dùng cho server, nơi URL resolve không bị CDN chặn vì
// client phát qua endpoint /stream chứ không fetch URL CDN trực tiếp.
func NewYtdlpScraperFastPath(binPath string) *YtdlpScraper {
	y := NewYtdlpScraper(binPath)
	y.fastPath = true
	return y
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

	return &YtdlpScraper{
		BinPath:        binPath,
		fastpathMisses: cache.NewTTL[struct{}](maxFastpathMisses, fastpathMissTTL),
	}
}

type ytdlpSearchItem struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Type         string  `json:"_type"`
	ExtractorKey string  `json:"extractor_key"`
	Thumbnail    string  `json:"thumbnail"`
	Duration     float64 `json:"duration"`
	Uploader     string  `json:"uploader"`
	URL          string  `json:"url"`
	Thumbnails   []struct {
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
	args := []string{
		query,
		"--dump-json",
		"--flat-playlist",
		// Chạy IPv4 ổn định + timeout mỗi socket để không kẹt vô hạn khi
		// host có IPv6 route lỗi (search không được nhanh cũng phải fail
		// sớm chứ không kéo dài tới timeout của app).
		"--socket-timeout", "15",
	}
	args = append(args, ytBaseArgs()...)
	args = append(args, ytPreferredFlag()...)
	args = append(args, cacheDirArgs()...)
	cmd := exec.CommandContext(ctx, y.BinPath, args...)

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

			// Bỏ kênh/playlist, chỉ giữ video (_type "url" với --flat-playlist).
			if item.Type != "" && item.Type != "url" {
				continue
			}
			if item.ExtractorKey != "" && item.ExtractorKey != "Youtube" {
				continue
			}
			// Channel handle (@...), không phải video id 11 ký tự.
			if strings.HasPrefix(item.ID, "@") {
				continue
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
	if y.fastPathEnabled() && !y.isFastpathMiss(videoID) {
		if info := fetchStreamInfoLib(ctx, videoID); info != nil {
			return info, nil
		}
		y.markFastpathMiss(videoID)
	}

	if err := acquireYtdlp(ctx); err != nil {
		return nil, err
	}
	defer releaseYtdlp()

	var lastErr error
	for attempt := 0; ; attempt++ {
		client := nextClient(attempt)
		if client == "" {
			if lastErr != nil {
				return nil, withYtHint(lastErr)
			}
			return nil, fmt.Errorf("yt-dlp stream info: no client worked for %s", videoID)
		}

		args := []string{
			"--print", "%(title)s\t%(url)s\t%(http_headers)s",
			"-f", "bestaudio",
			"--no-playlist",
		}
		args = append(args, ytBaseArgs()...)
		args = append(args, "--extractor-args", "youtube:player_client="+client)
		args = append(args, cacheDirArgs()...)
		args = append(args, "--", "https://www.youtube.com/watch?v="+videoID)
		cmd := exec.CommandContext(ctx, y.BinPath, args...)

		var out, stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			lastErr = fmt.Errorf("yt-dlp stream info (%s): %w: %s", client, err, stderr.String())
			if ytRetryable(lastErr.Error()) {
				continue
			}
			return nil, lastErr
		}

		info, err := parseStreamInfo(out.String(), videoID)
		if err != nil {
			return nil, err
		}
		// Không markClientSuccess ở đây: resolve in URL thành công với mọi
		// client, chỉ download (lấy dữ liệu) mới phân biệt được client nào
		// thực sự bị CDN chặn.
		return info, nil
	}
}

// fastPathEnabled bật khi NewYtdlpScraperFastPath được dùng (server), hoặc
// khi set SOM_YTDLP_FASTPATH=1. Lib youtube/v2 nhanh hơn nhưng URL có thể bị
// 403 IP-binding nếu client fetch trực tiếp — đó là lý do TUI local tắt.
func (y *YtdlpScraper) fastPathEnabled() bool {
	if y != nil && y.fastPath {
		return true
	}
	return strings.TrimSpace(os.Getenv("SOM_YTDLP_FASTPATH")) == "1"
}

func parseStreamInfo(raw, videoID string) (*StreamInfo, error) {
	line := strings.TrimSpace(raw)
	title, rest, found := strings.Cut(line, "\t")
	if !found {
		return nil, fmt.Errorf("yt-dlp returned empty output for %s", videoID)
	}
	audioURL, headersRaw, _ := strings.Cut(rest, "\t")
	if audioURL == "" {
		return nil, fmt.Errorf("yt-dlp returned empty URL for %s", videoID)
	}

	// yt-dlp có thể in nhiều dòng (video+audio); lấy dòng đầu.
	if idx := strings.Index(audioURL, "\n"); idx > 0 {
		audioURL = audioURL[:idx]
	}
	if title == "" || title == "NA" {
		title = videoID // fallback to video ID
	}

	headers := parsePythonDict(headersRaw)
	if headers["User-Agent"] == "" {
		headers["User-Agent"] = defaultUA
	}

	return &StreamInfo{
		URL:     audioURL,
		Title:   title,
		Headers: headers,
	}, nil
}

// parsePythonDict bóc map dạng "{'Key': 'value', ...}" mà yt-dlp in ra cho
// %(http_headers)s thành map[string]string.
func parsePythonDict(s string) map[string]string {
	m := make(map[string]string)
	re := regexp.MustCompile(`'([^']+)'\s*:\s*'([^']*)'`)
	for _, match := range re.FindAllStringSubmatch(s, -1) {
		m[match[1]] = match[2]
	}
	return m
}

// fetchStreamInfoLib resolve title + URL audio trực tiếp bằng lib youtube/v2
// (chạy trong process, không spawn yt-dlp). Trả nil nếu bất kỳ bước nào thất
// bại để caller dùng fallback yt-dlp.
func fetchStreamInfoLib(ctx context.Context, videoID string) *StreamInfo {
	libCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	client := youtube.Client{}
	v, err := client.GetVideoContext(libCtx, videoID)
	if err != nil {
		log.Printf("fastpath: %s getvideo err: %v", videoID, err)
		return nil
	}
	if v.Title == "" {
		log.Printf("fastpath: %s empty title", videoID)
		return nil
	}
	best := bestAudioLibFormat(v.Formats)
	if best == nil {
		log.Printf("fastpath: %s no audio format", videoID)
		return nil
	}
	url, err := client.GetStreamURLContext(libCtx, v, best)
	if err != nil {
		log.Printf("fastpath: %s streamurl itag=%d err: %v", videoID, best.ItagNo, err)
		return nil
	}
	if url == "" {
		log.Printf("fastpath: %s empty url", videoID)
		return nil
	}
	return &StreamInfo{URL: url, Title: v.Title, Headers: map[string]string{"User-Agent": defaultUA}}
}

// bestAudioLibFormat chọn format audio-only tốt nhất (ops: 251 > 250 >
// 249; mp4 aac: 140 > 139; còn lại theo bitrate giảm dần).
func bestAudioLibFormat(fl youtube.FormatList) *youtube.Format {
	var best *youtube.Format
	for _, f := range fl {
		if !isAudioOnlyLibFormat(f.MimeType) {
			continue
		}
		if best == nil || libAudioRank(f) > libAudioRank(*best) {
			c := f
			best = &c
		}
	}
	return best
}

func isAudioOnlyLibFormat(mime string) bool {
	return strings.Contains(mime, "audio/") && !strings.Contains(mime, "video")
}

func libAudioRank(f youtube.Format) int {
	rank := map[int]int{251: 100, 250: 90, 140: 80, 249: 70, 260: 60, 139: 50}[f.ItagNo]
	if rank != 0 {
		return rank
	}
	// Mime type chứa "opus" ưu tiên hơn aac khi itag lạ.
	if strings.Contains(f.MimeType, "opus") {
		return 40
	}
	return 10
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

	args := []string{
		"--print", "%(title)s",
		"--no-playlist",
		"--skip-download",
	}
	args = append(args, ytBaseArgs()...)
	args = append(args, ytPreferredFlag()...)
	args = append(args, cacheDirArgs()...)
	args = append(args, "--", "https://www.youtube.com/watch?v="+videoID)
	cmd := exec.CommandContext(ctx, y.BinPath, args...)

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

	args := []string{
		"--print", "%(track)s|||%(artist)s",
		"--no-playlist",
		"--skip-download",
	}
	args = append(args, ytBaseArgs()...)
	args = append(args, ytPreferredFlag()...)
	args = append(args, cacheDirArgs()...)
	args = append(args, "--", "https://www.youtube.com/watch?v="+videoID)
	cmd := exec.CommandContext(metaCtx, y.BinPath, args...)

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
	unlock := lyricsLocks.Lock(videoID)
	defer unlock()

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

	args := []string{
		"--write-subs",
		"--write-auto-subs",
		"--skip-download",
		"--sub-format", "vtt",
		"--sub-langs", "all",

		"--socket-timeout", "20",
		"-o", outTmpl,
	}
	args = append(args, ytBaseArgs()...)
	args = append(args, ytPreferredFlag()...)
	args = append(args, cacheDirArgs()...)
	args = append(args, "--", "https://www.youtube.com/watch?v="+videoID)
	cmd := exec.CommandContext(ytdlpCtx, y.BinPath, args...)

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
			Source:   "youtube",
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
	args := []string{
		"--print", "%(original_language)s",
		"--skip-download",
		"--no-warnings",
		"--no-playlist",
		"--force-ipv4",
	}
	args = append(args, cacheDirArgs()...)
	args = append(args, "--", "https://www.youtube.com/watch?v="+videoID)
	cmd := exec.CommandContext(langCtx, binPath, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if cmd.Run() == nil {
		return strings.TrimSpace(out.String())
	}
	return ""
}
