// internal/player/mpv.go
// Thin wrapper around mpv for audio playback.
// mpv is the natural choice because SOM already uses yt-dlp; mpv can pipe
// directly from the stream URL without re-downloading.
package player

import (
	"fmt"
	"os/exec"
	"sync"
)

type State int

const (
	Stopped State = iota
	Playing
	Paused
)

type Player struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	state   State
	current string // URL or file path
}

func New() *Player { return &Player{} }

// Play starts (or restarts) playback of the given URL / file path.
// mpv is invoked with --no-video --really-quiet so it only outputs audio.
func (p *Player) Play(streamURL string) error {
	p.Stop()
	p.mu.Lock()
	defer p.mu.Unlock()

	_, err := exec.LookPath("mpv")
	if err != nil {
		return fmt.Errorf("mpv not found in $PATH – please install mpv")
	}

	p.cmd = exec.Command("mpv",
		"--no-video",
		"--really-quiet",
		"--no-cache",
		streamURL,
	)
	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("mpv start: %w", err)
	}
	p.state = Playing
	p.current = streamURL

	// reap process in background so we don't leak zombies
	go p.cmd.Wait()
	return nil
}

// Stop kills mpv if it is running.
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
		p.cmd = nil
	}
	p.state = Stopped
}

// TogglePause sends SIGSTOP / SIGCONT to mpv (Linux/macOS).
// On Windows this is a no-op (mpv does not support SIGSTOP).
func (p *Player) TogglePause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd == nil || p.cmd.Process == nil {
		return
	}
	switch p.state {
	case Playing:
		_ = sendSignalPause(p.cmd.Process)
		p.state = Paused
	case Paused:
		_ = sendSignalResume(p.cmd.Process)
		p.state = Playing
	}
}

func (p *Player) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

func (p *Player) Current() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.current
}
