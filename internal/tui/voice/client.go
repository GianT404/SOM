package voice

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	DefaultCommand      = "som-voice-wake"
	DefaultWakeWord     = "yui"
	DefaultCommandPause = 3 * time.Second
	maxEventLine        = 256 * 1024
	eventBufferSize     = 32
)

type Config struct {
	Command        string
	WakeWord       string
	WakeAliases    string
	CommandTimeout time.Duration
}

type Event struct {
	Event      string `json:"event"`
	Timestamp  string `json:"timestamp"`
	Transcript string `json:"transcript,omitempty"`
	WakeAlias  string `json:"wake_alias,omitempty"`
	Command    string `json:"command,omitempty"`
	State      string `json:"state"`
}

type Client struct {
	config Config
	mu     sync.Mutex
	cmd    *exec.Cmd
	done   chan struct{}
	stop   chan struct{}
	events chan Event
}

func NewClient(config Config) *Client {
	if strings.TrimSpace(config.Command) == "" {
		config.Command = envOr("SOM_VOICE_COMMAND", DefaultCommand)
	}
	if strings.TrimSpace(config.WakeWord) == "" {
		config.WakeWord = envOr("SOM_WAKE_WORD", DefaultWakeWord)
	}
	if strings.TrimSpace(config.WakeAliases) == "" {
		config.WakeAliases = strings.TrimSpace(os.Getenv("SOM_WAKE_ALIASES"))
	}
	if config.CommandTimeout <= 0 {
		config.CommandTimeout = DefaultCommandPause
	}
	return &Client{config: config}
}

func (c *Client) Start() (<-chan Event, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cmd != nil {
		return nil, errors.New("voice client is already running")
	}
	if _, err := exec.LookPath(c.config.Command); err != nil {
		return nil, fmt.Errorf("voice command %q not found: %w", c.config.Command, err)
	}

	cmd := exec.Command(c.config.Command, c.buildArgs()...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("voice stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start voice command: %w", err)
	}

	c.cmd = cmd
	c.events = make(chan Event, eventBufferSize)
	c.done = make(chan struct{})
	c.stop = make(chan struct{})

	stop := c.stop
	go c.readLoop(stdout, cmd, stop, &stderr)
	return c.events, nil
}

func (c *Client) buildArgs() []string {
	args := []string{"--json", "--wake-word", c.config.WakeWord}
	if aliases := strings.TrimSpace(c.config.WakeAliases); aliases != "" {
		args = append(args, "--wake-aliases", aliases)
	}
	if c.config.CommandTimeout > 0 {
		args = append(args, "--command-timeout", c.config.CommandTimeout.String())
	}
	return args
}

func (c *Client) readLoop(stdout io.ReadCloser, cmd *exec.Cmd, stop <-chan struct{}, stderr *bytes.Buffer) {
	defer stdout.Close()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4096), maxEventLine)
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		if strings.TrimSpace(event.Event) == "" {
			continue
		}
		select {
		case c.events <- event:
		case <-stop:
			goto wait
		}
	}
	if err := scanner.Err(); err != nil {
		select {
		case c.events <- Event{Event: "error", State: err.Error()}:
		case <-stop:
		}
	}

wait:
	exitErr := cmd.Wait()

	stderrText := ""
	if stderr != nil {
		stderrText = strings.TrimSpace(stderr.String())
	}
	if stderrText != "" || exitErr != nil {
		state := stderrText
		if exitErr != nil {
			if state != "" {
				state = fmt.Sprintf("%v: %s", exitErr, state)
			} else {
				state = exitErr.Error()
			}
		}
		select {
		case c.events <- Event{Event: "error", State: state}:
		case <-stop:
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cmd == cmd {
		c.cmd = nil
	}
	close(c.done)
	close(c.events)
}

func (c *Client) Close() error {
	c.mu.Lock()
	cmd, done, stop := c.cmd, c.done, c.stop
	c.stop = nil
	c.mu.Unlock()

	if cmd == nil {
		return nil
	}
	if stop != nil {
		close(stop)
	}
	if cmd.Process != nil {
		_ = cmd.Process.Signal(os.Interrupt)
	}

	select {
	case <-done:
		return nil
	case <-time.After(3 * time.Second):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-done
		return nil
	}
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
