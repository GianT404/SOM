package player

import (
	"bytes"
	"fmt"
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
		otoCtx: ctx,
		state:  Stopped,
		volume: 1.0,
	}
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
	p.generation++

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
		"-af", "dynaudnorm=f=250:g=11:p=0.9:m=10",
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
		p.pauseOffset = time.Since(p.startTime)
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

func (p *Player) Generation() uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.generation
}
