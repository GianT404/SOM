package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"som/internal/scraper"
)

// StreamHandler serves audio for playback.
//
// Trước đây endpoint 302 thẳng sang URL CDN googlevideo mà yt-dlp resolve.
// Cách đó nhanh (không tốn server) nhưng phụ thuộc vào client tự xử lý
// redirect + Range: CDN throttle download thường (~30KB/s), và một số client
// không seek được / bị cắt giữa chừng.
//
// Giờ endpoint tải audio về file cục bộ (DownloadAudio, cache theo videoID)
// rồi serve qua http.ServeContent: có Range/206 nên seek được, tốc độ tải
// là tốc độ đường truyền thật (không bị CDN throttle), và chơi lại bài cũ
// thì trả file cached ngay.
//
// Khi query ?clean=1, endpoint transcode file cục bộ qua ffmpeg silenceremove
// Kết quả được cache ở /tmp/dopus-clean-<id>.ogg để lần sau không transcode
// lại. Transcode từ file cục bộ nên không còn lỗi 403 khi ffmpeg fetch thẳng
// URL CDN (vấn đề IP-binding/multi-IP trước đây).
type StreamHandler struct {
	Scraper scraper.Scraper
}

// NewStreamHandler creates a StreamHandler.
func NewStreamHandler(sc scraper.Scraper) *StreamHandler {
	return &StreamHandler{Scraper: sc}
}

// cleanCacheTTL khớp vòng đời URL chữ ký YouTube (~6h). Bản clean chỉ phụ
// thuộc videoID nên có thể cache theo videoID mà không cần đối chiếu URL.
const cleanCacheTTL = 6 * time.Hour

// cleanLocks chặn 2 request ?clean=1 cùng transcode cho một video.
var cleanLocks = newKeyedLock()

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

// safeFilename strips characters that are invalid in filenames.
// \x00-\x1f loại bỏ control characters (gồm CRLF) để chống header injection
// trong Content-Disposition.
var reUnsafe = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

func safeFilename(title string) string {
	safe := reUnsafe.ReplaceAllString(title, "")
	safe = strings.TrimSpace(safe)
	if safe == "" {
		safe = "audio"
	}
	return safe
}

func (h *StreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, ok := validateVideoID(w, r)
	if !ok {
		return
	}

	// Lấy title (cached 4h trong ResilientScraper) chỉ để đặt tên file.
	info, err := h.Scraper.GetStreamInfo(r.Context(), id)
	if err != nil {
		log.Printf("stream: resolve error for %s: %v", id, err)
		if errors.Is(err, scraper.ErrCircuitOpen) {
			w.Header().Set("Retry-After", "30")
			writeError(w, http.StatusServiceUnavailable, "stream temporarily unavailable, try again later")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to resolve audio URL")
		return
	}
	filename := safeFilename(info.Title) + ".opus"

	if r.URL.Query().Get("clean") == "1" {
		h.proxyClean(w, r, id, filename)
		return
	}

	// Tải file opus về local (cache theo videoID) rồi serve có Range.
	path, err := h.Scraper.DownloadAudio(r.Context(), id)
	if err != nil {
		log.Printf("stream: download error for %s: %v", id, err)
		writeError(w, http.StatusInternalServerError, "failed to prepare audio")
		return
	}
	serveAudioFile(w, r, path, filename)
}

// serveAudioFile serve file audio cục bộ qua ServeContent để hỗ trợ
// Range/206 (seek) và If-Range.
func serveAudioFile(w http.ResponseWriter, r *http.Request, path, filename string) {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("stream: open audio %s: %v", path, err)
		writeError(w, http.StatusInternalServerError, "audio file unavailable")
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		log.Printf("stream: stat audio %s: %v", path, err)
		writeError(w, http.StatusInternalServerError, "audio file unavailable")
		return
	}

	w.Header().Set("Content-Type", "audio/ogg")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	http.ServeContent(w, r, filename, fi.ModTime(), f)
}

// proxyClean transcode file opus cục bộ qua ffmpeg silenceremove rồi serve
// bản đã cache. Cache tại /tmp/dopus-clean-<id>.ogg; lần sau serve thẳng.
// Dùng context tách khỏi timeout 3 phút của router vì transcode dài, nhưng
// vẫn bị hủy khi client ngắt kết nối.
func (h *StreamHandler) proxyClean(w http.ResponseWriter, r *http.Request, videoID, filename string) {
	ctx := context.WithoutCancel(r.Context())
	cleanPath := filepath.Join(os.TempDir(), fmt.Sprintf("dopus-clean-%s.ogg", videoID))

	if h.serveCachedClean(w, r, filename, cleanPath) {
		return
	}

	// Tránh 2 request cùng lúc cùng transcode cho 1 video.
	unlock := cleanLocks.Lock(videoID)
	defer unlock()
	if h.serveCachedClean(w, r, filename, cleanPath) {
		return
	}

	// Nguồn transcode là file cục bộ (đã download qua yt-dlp) — ổn định hơn
	// hẳn việc để ffmpeg tự fetch URL CDN.
	srcPath, err := h.Scraper.DownloadAudio(ctx, videoID)
	if err != nil {
		log.Printf("stream: clean download error for %s: %v", videoID, err)
		writeError(w, http.StatusInternalServerError, "failed to prepare audio")
		return
	}

	partPath := cleanPath + ".part"
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner", "-loglevel", "error",
		"-y",
		"-i", srcPath,
		"-vn",
		"-c:a", "libopus", "-b:a", "160k",
		"-f", "ogg",
		partPath,
	)

	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Printf("stream: ffmpeg clean error for %s: %v; stderr: %s", videoID, err, stderr.String())
		_ = os.Remove(partPath)
		writeError(w, http.StatusInternalServerError, "audio processing failed")
		return
	}

	if err := os.Rename(partPath, cleanPath); err != nil {
		log.Printf("stream: cache rename error for %s: %v", videoID, err)
		_ = os.Remove(partPath)
	}

	h.serveCachedClean(w, r, filename, cleanPath)
}

// serveCachedClean serve thẳng file clean nếu có bản còn tươi. Trả về true
// nếu đã serve xong.
func (h *StreamHandler) serveCachedClean(w http.ResponseWriter, r *http.Request, filename, path string) bool {
	fi, err := os.Stat(path)
	if err != nil || time.Since(fi.ModTime()) > cleanCacheTTL {
		return false
	}
	log.Printf("stream: serving cached clean file for %s", r.URL.Query().Get("id"))
	serveAudioFile(w, r, path, filename)
	return true
}
