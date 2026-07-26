package player

import (
	"fmt"
	"os/exec"
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

type Player struct {
	mu          sync.Mutex
	state       State
	otoCtx      *oto.Context
	player      *oto.Player
	decoder     *exec.Cmd
	currentPath string
	startTime   time.Time
	pauseOffset time.Duration
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
	}
}

func (p *Player) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

func (p *Player) Play(filePath string) error {
	return p.playFrom(filePath, 0)
}

func (p *Player) playFrom(filePath string, startSec int) error {
	p.Stop()

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.otoCtx == nil {
		return fmt.Errorf("oto context is not initialized")
	}

	p.currentPath = filePath
	p.startTime = time.Now().Add(-time.Duration(startSec) * time.Second)
	p.pauseOffset = 0

	args := []string{}
	if startSec > 0 {
		args = append(args, "-ss", fmt.Sprintf("%d", startSec))
	}
	args = append(args,
		"-i", filePath,
		"-f", "s16le",
		"-ar", "48000",
		"-ac", "2",
		"-v", "quiet",
		"-",
	)

	p.decoder = exec.Command("ffmpeg", args...)

	pcmOut, err := p.decoder.StdoutPipe()
	if err != nil {
		return fmt.Errorf("ffmpeg stdout pipe error: %w", err)
	}

	if err := p.decoder.Start(); err != nil {
		return fmt.Errorf("ffmpeg start error: %w", err)
	}

	p.player = p.otoCtx.NewPlayer(pcmOut)
	p.player.Play()
	p.state = Playing

	go func(cmd *exec.Cmd) {
		_ = cmd.Wait()
		p.mu.Lock()
		if p.decoder == cmd {
			p.state = Stopped
		}
		p.mu.Unlock()
	}(p.decoder)

	return nil
}

func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state != Stopped {
		if p.player != nil {
			_ = p.player.Close()
		}
		if p.decoder != nil && p.decoder.Process != nil {
			_ = p.decoder.Process.Kill()
		}
		p.state = Stopped
		p.currentPath = ""
	}
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

	_ = p.playFrom(path, newSec)
}

func (p *Player) Stderr() string {
	return ""
}
