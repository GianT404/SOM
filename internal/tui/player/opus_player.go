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

type gaplessReader struct {
	buf    []byte
	offset int
	pipe   io.ReadCloser
}

func (r *gaplessReader) Read(p []byte) (n int, err error) {
	if r.offset < len(r.buf) {
		n = copy(p, r.buf[r.offset:])
		r.offset += n
		return n, nil
	}
	return r.pipe.Read(p)
}

func (r *gaplessReader) Close() error {
	if r.pipe != nil {
		return r.pipe.Close()
	}
	return nil
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
	lastErr     error
	stopped     bool
	stderrBuf   syncBuffer
	volume      float64
	generation  uint64
	audioFilter string
	speed       float64

	// Gapless playback: pre-decoded next track.
	nextCmd   *exec.Cmd
	nextPipe  io.ReadCloser
	nextBuf   []byte
	nextPath  string
	nextGen   uint64
	nextReady bool
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
		return &Player{state: Stopped}
	}

	<-ready

	return &Player{
		otoCtx:      ctx,
		state:       Stopped,
		volume:      1.0,
		speed:       1.0,
		audioFilter: "dynaudnorm=f=250:g=11:p=0.9:m=10",
	}
}

func (p *Player) SetSpeed(s float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if s <= 0 {
		s = 1.0
	}
	p.speed = s
}

func (p *Player) SetAudioFilter(filter string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if filter == "" {
		filter = "dynaudnorm=f=250:g=11:p=0.9:m=10"
	}
	p.audioFilter = filter
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
	speed := p.speed
	if speed <= 0 {
		speed = 1.0
	}
	switch p.state {
	case Playing:
		return time.Duration(float64(time.Since(p.startTime)) * speed)
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

func (p *Player) playFrom(filePath string, startSec float64, headers map[string]string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stopLocked()

	if p.otoCtx == nil {
		return fmt.Errorf("oto context is not initialized")
	}

	speed := p.speed
	if speed <= 0 {
		speed = 1.0
	}

	p.currentPath = filePath
	p.headers = headers
	p.startTime = time.Now().Add(-time.Duration((startSec / speed) * float64(time.Second)))
	p.pauseOffset = 0
	p.lastErr = nil
	p.stopped = false
	p.generation++

	args := p.buildFFmpegArgs(filePath, headers, startSec, speed)

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

	p.player = p.otoCtx.NewPlayer(pcmOut)
	p.player.SetVolume(p.volume)

	p.player.Play()
	p.state = Playing

	go func(cmd *exec.Cmd, gen uint64, optr *oto.Player) {
		err := cmd.Wait()

		if optr != nil {
			for optr.IsPlaying() {
				time.Sleep(50 * time.Millisecond)

				p.mu.Lock()
				isStopped := p.stopped
				p.mu.Unlock()

				if isStopped {
					break // Bỏ qua buffer và thoát ngay nếu user chủ động ngắt/chuyển bài
				}
			}
		}

		p.mu.Lock()
		defer p.mu.Unlock()

		if p.decoder == cmd && p.generation == gen {
			if err != nil && !p.stopped {
				p.lastErr = err
				p.state = Stopped
				p.stopped = false
			} else if err == nil && !p.stopped {
				p.lastErr = nil
				p.state = Stopped
				p.stopped = false
			}
		}
	}(p.decoder, p.generation, p.player)

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
	p.cancelPreDecodeLocked()
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
	speed := p.speed
	if speed <= 0 {
		speed = 1.0
	}
	if p.state == Playing {
		p.player.Pause()
		p.state = Paused
		p.pauseOffset = time.Duration(float64(time.Since(p.startTime)) * speed)
	} else if p.state == Paused {
		p.player.Play()
		p.state = Playing
		p.startTime = time.Now().Add(-time.Duration(float64(p.pauseOffset) / speed))
	}
}

// SeekBy hỗ trợ tua tới / tua lùi nhạc
func (p *Player) SeekBy(seconds float64) {
	p.mu.Lock()
	path := p.currentPath
	if path == "" || p.state == Stopped {
		p.mu.Unlock()
		return
	}
	speed := p.speed
	if speed <= 0 {
		speed = 1.0
	}
	var elapsed float64
	if p.state == Paused {
		elapsed = p.pauseOffset.Seconds()
	} else {
		elapsed = time.Since(p.startTime).Seconds() * speed
	}
	newSec := elapsed + seconds
	if newSec < 0 {
		newSec = 0
	}
	p.mu.Unlock()
	_ = p.playFrom(path, newSec, p.currentHeaders())
}

func (p *Player) SeekTo(seconds float64) {
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

func (p *Player) Generation() uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.generation
}

func (p *Player) CurrentPath() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.currentPath
}

func (p *Player) PreDecodeNext(filePath string, headers map[string]string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.cancelPreDecodeLocked()

	speed := p.speed
	if speed <= 0 {
		speed = 1.0
	}

	args := p.buildFFmpegArgs(filePath, headers, 0, speed)

	cmd := exec.Command("ffmpeg", args...)
	var stderr syncBuffer
	cmd.Stderr = &stderr

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return
	}

	if err := cmd.Start(); err != nil {
		return
	}

	p.nextCmd = cmd
	p.nextPipe = pipe
	p.nextPath = filePath
	p.nextGen = p.generation
	p.nextReady = false

	const maxBuf = 960000
	go func() {
		buf := make([]byte, 0, maxBuf)
		tmp := make([]byte, 4096)
		for len(buf) < maxBuf {
			n, err := pipe.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[:n]...)
			}
			if err != nil {
				break
			}
		}

		p.mu.Lock()
		defer p.mu.Unlock()

		if p.nextCmd != cmd || p.generation != p.nextGen {
			_ = pipe.Close()
			if p.nextCmd == cmd {
				// tự kill và Wait để không để lại zombie.
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
					go func() { _ = cmd.Wait() }()
				}
			}
			return
		}

		p.nextBuf = buf
		p.nextReady = true
	}()
}

func (p *Player) CancelPreDecode() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cancelPreDecodeLocked()
}

func (p *Player) cancelPreDecodeLocked() {
	if p.nextCmd != nil {
		if p.nextPipe != nil {
			_ = p.nextPipe.Close()
		}
		cmd := p.nextCmd
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			// Wait nền để thu hồi process đã kill, tránh zombie; không giữ
			// mutex trong lúc chờ ffmpeg tắt.
			go func() { _ = cmd.Wait() }()
		}
		p.nextCmd = nil
		p.nextPipe = nil
		p.nextBuf = nil
		p.nextReady = false
	}
}

// PlayFromBuffer starts playing the pre-decoded next track seamlessly.
// Returns true if gapless playback was used, false if fallback to normal Play().
func (p *Player) PlayFromBuffer() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.nextReady || p.nextCmd == nil || p.nextPipe == nil {
		return false
	}

	// Save pre-decoded data before stopLocked() clears them via cancelPreDecodeLocked().
	cmd := p.nextCmd
	pipe := p.nextPipe
	buf := p.nextBuf
	nextPath := p.nextPath
	p.nextCmd = nil
	p.nextPipe = nil
	p.nextBuf = nil
	p.nextReady = false

	p.stopLocked()

	if p.otoCtx == nil {
		_ = pipe.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			go func() { _ = cmd.Wait() }()
		}
		return false
	}

	p.currentPath = nextPath
	p.headers = nil
	p.startTime = time.Now()
	p.pauseOffset = 0
	p.lastErr = nil
	p.stopped = false
	p.generation++
	p.decoder = cmd

	reader := &gaplessReader{buf: buf, pipe: pipe}

	p.player = p.otoCtx.NewPlayer(reader)
	p.player.SetVolume(p.volume)
	p.player.Play()
	p.state = Playing

	go func(c *exec.Cmd, gen uint64, optr *oto.Player) {
		err := c.Wait()

		if optr != nil {
			for optr.IsPlaying() {
				time.Sleep(50 * time.Millisecond)
				p.mu.Lock()
				isStopped := p.stopped
				p.mu.Unlock()
				if isStopped {
					break
				}
			}
		}

		p.mu.Lock()
		defer p.mu.Unlock()
		if p.decoder == c && p.generation == gen {
			if err != nil && !p.stopped {
				p.lastErr = err
			}
			p.state = Stopped
			p.stopped = false
		}
	}(cmd, p.generation, p.player)

	return true
}

func (p *Player) buildFFmpegArgs(filePath string, headers map[string]string, startSec float64, speed float64) []string {
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
		args = append(args, "-ss", fmt.Sprintf("%.3f", startSec))
	}

	af := p.audioFilter
	if speed != 1.0 {
		if speed == 0.25 {
			af += ",atempo=0.5,atempo=0.5"
		} else {
			af += fmt.Sprintf(",atempo=%.2f", speed)
		}
	}

	args = append(args,
		"-i", filePath,
		"-f", "s16le",
		"-ar", "48000",
		"-ac", "2",
		"-af", af,
		"-v", "error",
		"-",
	)
	return args
}
