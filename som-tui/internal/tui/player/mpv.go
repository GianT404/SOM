package player

import (
	"bytes"
	"fmt"
	"net"
	"os"
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
	mu     sync.Mutex
	cmd    *exec.Cmd
	state  State
	stderr bytes.Buffer
}

func New() *Player { return &Player{} }

const socketPath = "/tmp/mpv-som.sock"

func (p *Player) Play(streamURL string) error {
	p.Stop()

	if _, err := exec.LookPath("mpv"); err != nil {
		return fmt.Errorf("mpv not found in PATH")
	}
	_ = os.Remove(socketPath)

	p.mu.Lock()
	p.stderr.Reset()
	p.cmd = exec.Command("mpv",
		"--no-video",
		"--really-quiet",
		"--audio-buffer=5",
		"--cache=yes",
		"--cache-secs=10",
		"--no-terminal",
		"--input-ipc-server="+socketPath,
		streamURL,
	)
	p.cmd.Stderr = &p.stderr
	p.mu.Unlock()

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("mpv start: %w", err)
	}

	p.mu.Lock()
	p.state = Playing
	currentCmd := p.cmd
	p.mu.Unlock()

	go func(c *exec.Cmd) {
		c.Wait()

		p.mu.Lock()
		defer p.mu.Unlock()
		if p.cmd == c {
			p.state = Stopped
			_ = os.Remove(socketPath)
		}
	}(currentCmd)

	return nil
}

func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	p.cmd = nil
	p.state = Stopped
	_ = os.Remove(socketPath)
}
func (p *Player) TogglePause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd == nil || p.cmd.Process == nil || p.state == Stopped {
		return
	}
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return
	}
	defer conn.Close()
	_, err = conn.Write([]byte("cycle pause\n"))
	if err != nil {
		return
	}

	if p.state == Playing {
		p.state = Paused
	} else if p.state == Paused {
		p.state = Playing
	}
}

func (p *Player) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

func (p *Player) Stderr() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stderr.String()
}
