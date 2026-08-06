package handler

import (
	"bytes"
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

// StreamHandler redirects to the direct audio URL resolved by yt-dlp.
// Trước đây endpoint này tải cả file về temp rồi mới serve -> first byte
// phải chờ toàn bộ quá trình download (30s+). Giờ nó 302 sang URL CDN mà
// yt-dlp đã resolve sẵn (đã qua xử lý signature/throttling) và được cache
// 50 phút trong ResilientScraper: client nhận response tức thì, backend
// không còn proxy nguyên file qua mạng.
//
// Khi query ?clean=1, endpoint proxy stream qua ffmpeg với filter
// silenceremove (Smart Silence Skip): cắt mọi khoảng lặng dưới -50dB kéo
// dài hơn 1 giây (gồm cả ở giữa và cuối bài). Chế độ này chậm hơn vì
// backend phải transcode realtime, chỉ dùng khi client yêu cầu.
type StreamHandler struct {
	Scraper scraper.Scraper
}

// NewStreamHandler creates a StreamHandler.
func NewStreamHandler(sc scraper.Scraper) *StreamHandler {
	return &StreamHandler{Scraper: sc}
}

// silenceremoveFilter cắt mọi khoảng lặng < -50dB kéo dài >= 1s.
// stop_periods=-1 nghĩa là trim TẤT CẢ các khoảng lặng (đầu, giữa, cuối).
var silenceremoveFilter = "silenceremove=start_periods=1:start_threshold=-50dB:start_duration=1:stop_periods=-1:stop_threshold=-50dB:stop_duration=1"

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
		h.proxyClean(w, r, info.URL, filename)
		return
	}

	w.Header().Set("Content-Type", "audio/ogg")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	http.Redirect(w, r, info.URL, http.StatusFound)
}

// cappedBuffer ghi stderr của ffmpeg nhưng chỉ giữ tối đa maxBytes cuối cùng
// để không tích tụ log rác, phục vụ debug khi ffmpeg lỗi.
type cappedBuffer struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	max  int
	drop bool
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.buf.Len()+len(p) > c.max {
		c.buf.Reset()
		c.drop = true
	}
	return c.buf.Write(p)
}

func (c *cappedBuffer) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.drop {
		return "(truncated) " + c.buf.String()
	}
	return c.buf.String()
}

// proxyClean tải CDN URL rồi pipe qua ffmpeg silenceremove, stream kết quả
// về client dưới dạng ogg/opus. Kết quả được ghi vào /tmp/dopus-clean-<id>.ogg
// để lần sau serve thẳng (đỡ tốn CPU transcode lại). Dùng context tách khỏi
// timeout 3 phút của router (stream dài), nhưng vẫn bị cắt khi client ngắt.
func (h *StreamHandler) proxyClean(w http.ResponseWriter, r *http.Request, srcURL, filename string) {
	ctx := context.WithoutCancel(r.Context())
	videoID := r.URL.Query().Get("id")
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

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner", "-loglevel", "error",
		"-i", srcURL,
		"-vn",
		"-af", silenceremoveFilter,
		"-c:a", "libopus", "-b:a", "160k",
		"-f", "ogg",
		"pipe:1",
	)

	stderr := &cappedBuffer{max: 4 << 10}
	cmd.Stderr = stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start audio processing")
		return
	}
	if err := cmd.Start(); err != nil {
		log.Printf("stream: ffmpeg start error: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to start audio processing")
		return
	}

	// Ghi đồng thời ra client và file cache (rename khi transcode thành công).
	partPath := cleanPath + ".part"
	cacheOut, cerr := os.Create(partPath)
	if cerr != nil {
		cacheOut = nil // không cache được thì vẫn stream như cũ
	}

	w.Header().Set("Content-Type", "audio/ogg")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)

	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 32*1024)
	for {
		n, rerr := stdout.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				_ = cmd.Process.Kill()
				break
			}
			if cacheOut != nil {
				if _, cerr := cacheOut.Write(buf[:n]); cerr != nil {
					_ = cacheOut.Close()
					_ = os.Remove(partPath)
					cacheOut = nil
				}
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if rerr != nil {
			break
		}
	}

	if err := cmd.Wait(); err != nil {
		if msg := stderr.String(); msg != "" {
			log.Printf("stream: ffmpeg clean error: %v; stderr: %s", err, msg)
		} else {
			log.Printf("stream: ffmpeg clean error: %v", err)
		}
		if cacheOut != nil {
			_ = cacheOut.Close()
		}
		_ = os.Remove(partPath)
		return
	}

	if cacheOut != nil {
		_ = cacheOut.Close()
		if err := os.Rename(partPath, cleanPath); err != nil {
			log.Printf("stream: cache rename error: %v", err)
		}
	}
}

// serveCachedClean serve thẳng file clean nếu có bản còn tươi. Trả về true
// nếu đã serve xong.
func (h *StreamHandler) serveCachedClean(w http.ResponseWriter, r *http.Request, filename, path string) bool {
	fi, err := os.Stat(path)
	if err != nil || time.Since(fi.ModTime()) > cleanCacheTTL {
		return false
	}
	log.Printf("stream: serving cached clean file for %s", r.URL.Query().Get("id"))
	w.Header().Set("Content-Type", "audio/ogg")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	http.ServeFile(w, r, path)
	return true
}
