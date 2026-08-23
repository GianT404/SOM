package player

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

type State int

const (
	Stopped State = iota
	Playing
	Paused
)

// syncBuffer là bytes.Buffer có lock riêng, an toàn khi ffmpeg (chạy nền)
// ghi log lỗi trong lúc UI gọi Stderr() đọc cùng lúc.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func (s *syncBuffer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf.Reset()
}

type pcmTap struct {
	mu   sync.Mutex
	subs map[chan []byte]struct{}
}

func newPCMTap() *pcmTap {
	return &pcmTap{subs: make(map[chan []byte]struct{})}
}

func (t *pcmTap) Write(p []byte) (int, error) {
	t.mu.Lock()
	if len(t.subs) == 0 {
		t.mu.Unlock()
		return len(p), nil
	}
	subs := make([]chan []byte, 0, len(t.subs))
	for c := range t.subs {
		subs = append(subs, c)
	}
	t.mu.Unlock()

	cp := make([]byte, len(p))
	copy(cp, p)
	for _, c := range subs {
		select {
		case c <- cp:
		default:
		}
	}
	return len(p), nil
}

func (t *pcmTap) subscribe() chan []byte {
	c := make(chan []byte, 8)
	t.mu.Lock()
	t.subs[c] = struct{}{}
	t.mu.Unlock()
	return c
}

func (t *pcmTap) unsubscribe(c chan []byte) {
	t.mu.Lock()
	delete(t.subs, c)
	t.mu.Unlock()
	close(c)
}

// flush xóa dữ liệu PCM cũ trong tất cả subscriber channels,
// tránh visualizer hiển thị data từ bài trước khi seek/next.
func (t *pcmTap) flush() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for c := range t.subs {
		drain := true
		for drain {
			select {
			case <-c:
			default:
				drain = false
			}
		}
	}
}

const pcmFrameBytes = 48000 * 2 * 2 / 30

func relayPCM(src io.Reader, tap *pcmTap) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		buf := make([]byte, pcmFrameBytes)
		for {
			n, err := io.ReadFull(src, buf)
			if n > 0 {
				tap.Write(buf[:n])
				if _, werr := pw.Write(buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				pw.CloseWithError(err)
				return
			}
		}
	}()

	return pr
}

type Player struct {
	mu          sync.Mutex
	state       State
	otoCtx      *oto.Context
	player      *oto.Player
	decoder     *exec.Cmd
	currentPath string
	headers     map[string]string
	startTime   time.Time
	pauseOffset time.Duration
	lastErr     error      // lỗi của lần phát vừa kết thúc; nil = phát hoàn tất bình thường
	stopped     bool       // đánh dấu user chủ động stop/kill, không tính là lỗi
	stderrBuf   syncBuffer // stderr thật của ffmpeg, để debug khi decode lỗi
	volume      float64
	tap         *pcmTap
}

func New() *Player {
	op := &oto.NewContextOptions{
		SampleRate:   48000,
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
	}
	ctx, ready, err := oto.NewContext(op)
	if err != nil {
		fmt.Printf("oto init error: %v\n", err)
		return &Player{state: Stopped, tap: newPCMTap()}
	}

	<-ready

	return &Player{
		otoCtx: ctx,
		state:  Stopped,
		volume: 1.0,
		tap:    newPCMTap(),
	}
}

// Subscribe trả về 1 channel nhận bản sao PCM (s16le, 48kHz, stereo) đang
// thực sự được phát — dùng cho visualizer đọc FFT trực tiếp từ chính âm
// thanh SOM đang decode, không cần capture loopback ở tầng hệ điều hành.
// Luôn phải Unsubscribe khi không dùng nữa để tránh leak channel.
func (p *Player) Subscribe() chan []byte {
	return p.tap.subscribe()
}

func (p *Player) Unsubscribe(c chan []byte) {
	p.tap.unsubscribe(c)
}

func (p *Player) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

// Position trả về vị trí phát hiện tại (tính cả trạng thái pause).
func (p *Player) Position() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	switch p.state {
	case Playing:
		return time.Since(p.startTime)
	case Paused:
		return p.pauseOffset
	default:
		return 0
	}
}

func (p *Player) Play(filePath string) error {
	return p.playFrom(filePath, 0, nil)
}

// PlayWithHeaders chơi URL (hoặc file) kèm HTTP headers truyền cho ffmpeg.

func (p *Player) PlayWithHeaders(filePath string, headers map[string]string) error {
	return p.playFrom(filePath, 0, headers)
}

func (p *Player) playFrom(filePath string, startSec int, headers map[string]string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Flush PCM tap trước khi stop player cũ, tránh visualizer hiển thị
	// data từ bài trước (stale frames).
	p.tap.flush()
	p.stopLocked()

	if p.otoCtx == nil {
		return fmt.Errorf("oto context is not initialized")
	}

	p.currentPath = filePath
	p.headers = headers
	p.startTime = time.Now().Add(-time.Duration(startSec) * time.Second)
	p.pauseOffset = 0
	p.lastErr = nil
	p.stopped = false

	var args []string
	if strings.HasPrefix(filePath, "http://") {
		args = append(args, "-reconnect", "1", "-reconnect_streamed", "1", "-reconnect_delay_max", "5")
	}
	if len(headers) > 0 {
		var hb strings.Builder
		for k, v := range headers {
			fmt.Fprintf(&hb, "%s: %s\r\n", k, v)
		}
		args = append(args, "-headers", hb.String())
	}
	if startSec > 0 {
		args = append(args, "-ss", fmt.Sprintf("%d", startSec))
	}
	args = append(args,
		"-i", filePath,
		"-f", "s16le",
		"-ar", "48000",
		"-ac", "2",
		"-v", "error",
		"-",
	)

	p.decoder = exec.Command("ffmpeg", args...)

	p.stderrBuf.Reset()
	p.decoder.Stderr = &p.stderrBuf

	pcmOut, err := p.decoder.StdoutPipe()
	if err != nil {
		return fmt.Errorf("ffmpeg stdout pipe error: %w", err)
	}

	if err := p.decoder.Start(); err != nil {
		return fmt.Errorf("ffmpeg start error: %w", err)
	}

	p.player = p.otoCtx.NewPlayer(relayPCM(pcmOut, p.tap))
	p.player.SetVolume(p.volume)

	p.player.Play()
	p.state = Playing

	go func(cmd *exec.Cmd) {
		err := cmd.Wait()
		p.mu.Lock()
		if p.decoder == cmd {
			// Chỉ tính là lỗi khi không phải user chủ động dừng (stop/seek/next).
			if err != nil && !p.stopped {
				p.lastErr = err
			}
			p.state = Stopped
			p.stopped = false
		}
		p.mu.Unlock()
	}(p.decoder)

	return nil
}

func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopLocked()

}
func (p *Player) stopLocked() {
	if p.state != Stopped {
		p.stopped = true
		if p.player != nil {
			_ = p.player.Close()
		}
		if p.decoder != nil && p.decoder.Process != nil {
			_ = p.decoder.Process.Kill()
		}
		p.state = Stopped
		p.currentPath = ""
		p.headers = nil
	}
}

// PlaybackError trả về lỗi của lần phát vừa kết thúc nếu ffmpeg thoát do lỗi(stream chết giữa chừng, file hỏng...). Nil nếu phát hoàn tất hoặc user dừng.
func (p *Player) PlaybackError() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastErr
}
func (p *Player) TogglePause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.player == nil || p.state == Stopped {
		return
	}

	if p.state == Playing {
		p.player.Pause()
		p.state = Paused
		p.pauseOffset += time.Since(p.startTime)
	} else if p.state == Paused {
		p.player.Play()
		p.state = Playing
		p.startTime = time.Now().Add(-p.pauseOffset)
	}
}

// SeekBy hỗ trợ tua tới / tua lùi nhạc
func (p *Player) SeekBy(seconds int) {
	p.mu.Lock()
	path := p.currentPath
	if path == "" || p.state == Stopped {
		p.mu.Unlock()
		return
	}

	elapsed := time.Since(p.startTime)
	if p.state == Paused {
		elapsed = p.pauseOffset
	}
	newSec := int(elapsed.Seconds()) + seconds
	if newSec < 0 {
		newSec = 0
	}
	p.mu.Unlock()

	_ = p.playFrom(path, newSec, p.currentHeaders())
}

// SeekTo nhảy tới giây cụ thể (absolute seek) và tiếp tục phát.
func (p *Player) SeekTo(seconds int) {
	p.mu.Lock()
	path := p.currentPath
	if path == "" || p.state == Stopped {
		p.mu.Unlock()
		return
	}
	if seconds < 0 {
		seconds = 0
	}
	p.mu.Unlock()

	_ = p.playFrom(path, seconds, p.currentHeaders())
}

// currentHeaders giữ headers của lần phát hiện tại để seek lại với đúng header.
func (p *Player) currentHeaders() map[string]string {
	p.mu.Lock()
	defer p.mu.Unlock()
	h := p.headers
	if h == nil {
		return nil
	}
	copyH := make(map[string]string, len(h))
	for k, v := range h {
		copyH[k] = v
	}
	return copyH
}

func (p *Player) Stderr() string {
	return p.stderrBuf.String()
}

func (p *Player) SetVolume(v float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if v < 0.0 {
		v = 0.0
	}
	if v > 2.0 {
		v = 2.0
	}

	p.volume = v
	if p.player != nil {
		p.player.SetVolume(v)
	}
}

func (p *Player) Volume() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume
}
