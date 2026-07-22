package player

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"som/internal/tui/bindeps"
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
	gen    uint64
}

func New() *Player { return &Player{} }

var socketPath = filepath.Join(os.TempDir(), "mpv-som.sock")

func (p *Player) Play(streamURL string) error {
	p.Stop()

	mpvPath := bindeps.Find("mpv")
	if mpvPath == "mpv" {
		if _, err := exec.LookPath("mpv"); err != nil {
			return fmt.Errorf("mpv not found in PATH")
		}
	}
	_ = os.Remove(socketPath)

	p.mu.Lock()
	p.gen++
	myGen := p.gen
	p.stderr.Reset()
	p.mu.Unlock()

	cmd := exec.Command(mpvPath,
		"--no-video",
		"--really-quiet",
		"--audio-buffer=5",
		"--cache=yes",
		"--cache-secs=10",
		"--no-terminal",
		"--input-ipc-server="+socketPath,
		streamURL,
	)

	p.mu.Lock()
	cmd.Stderr = &p.stderr
	p.mu.Unlock()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("mpv start: %w", err)
	}

	p.mu.Lock()
	if p.gen != myGen {
		p.mu.Unlock()
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return nil
	}
	p.cmd = cmd
	p.state = Playing
	p.mu.Unlock()

	go func(c *exec.Cmd, gen uint64) {
		c.Wait()

		p.mu.Lock()
		defer p.mu.Unlock()
		if p.gen == gen {
			p.state = Stopped
			_ = os.Remove(socketPath)
		}
	}(cmd, myGen)

	return nil
}

func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gen++
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	p.cmd = nil
	p.state = Stopped
	_ = os.Remove(socketPath)
}
func (p *Player) SeekBy(seconds int) {
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
	_, _ = conn.Write([]byte(fmt.Sprintf("seek %d relative\n", seconds)))
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
