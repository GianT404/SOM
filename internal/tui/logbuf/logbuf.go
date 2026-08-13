package logbuf

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const DefaultCapacity = 2000

type Ring struct {
	mu   sync.Mutex
	buf  []string
	head int
	size int

	version int
	snapVer int
	snap    atomic.Pointer[[]string]
}

func New(max int) *Ring {
	if max < 1 {
		max = 1
	}
	r := &Ring{buf: make([]string, max)}
	empty := make([]string, 0)
	r.snap.Store(&empty)
	return r
}

func (r *Ring) Capacity() int { return len(r.buf) }

func (r *Ring) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.size
}

func (r *Ring) Write(p []byte) (int, error) {
	s := strings.TrimRight(string(p), "\n")
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		r.push(line)
	}
	return len(p), nil
}

func (r *Ring) Push(line string) { r.push(line) }

func (r *Ring) push(line string) {
	r.mu.Lock()
	r.buf[r.head] = line
	r.head = (r.head + 1) % len(r.buf)
	if r.size < len(r.buf) {
		r.size++
	}
	r.version++
	r.mu.Unlock()
}

func (r *Ring) Replace(lines []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	capn := len(r.buf)
	if len(lines) > capn {
		lines = lines[len(lines)-capn:]
	}
	for i, l := range lines {
		r.buf[i] = l
	}
	r.head = len(lines) % capn
	r.size = len(lines)
	r.version++
	r.publishLocked()
}

func (r *Ring) Lines() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.snapVer != r.version {
		r.publishLocked()
	}
	return *r.snap.Load()
}

func (r *Ring) Snapshot() []string {
	if r.mu.TryLock() {
		if r.snapVer != r.version {
			r.publishLocked()
		}
		out := *r.snap.Load()
		r.mu.Unlock()
		return out
	}
	return *r.snap.Load()
}

func (r *Ring) publishLocked() {
	out := make([]string, r.size)
	capn := len(r.buf)
	start := (r.head - r.size + capn) % capn
	for i := 0; i < r.size; i++ {
		out[i] = r.buf[(start+i)%capn]
	}
	r.snapVer = r.version
	r.snap.Store(&out)
}

func CrashDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return home
}

func crashFilename() string {
	return fmt.Sprintf("crash_som_tui_%s.log", time.Now().Format("20060102_150405"))
}

func (r *Ring) DumpCrash(note string) string {
	path := filepath.Join(CrashDir(), crashFilename())
	if err := r.DumpCrashTo(path, note); err != nil {
		fmt.Fprintf(os.Stderr, "logbuf: crash dump FAILED (%s): %v\n", path, err)
	}
	return path
}

func (r *Ring) DumpCrashTo(path, note string) error {
	lines := r.Snapshot()

	stack := make([]byte, 1<<20)
	n := runtime.Stack(stack, true)

	var b strings.Builder
	b.WriteString("SOM TUI crash dump\n")
	b.WriteString("time: " + time.Now().Format(time.RFC3339) + "\n")
	if note != "" {
		b.WriteString("reason: " + note + "\n")
	}
	b.WriteString(fmt.Sprintf("ring capacity: %d  retained: %d\n", r.Capacity(), len(lines)))
	b.WriteString("\n--- ring buffer (oldest -> newest) ---\n")
	for _, l := range lines {
		b.WriteString(l)
		b.WriteString("\n")
	}
	b.WriteString("\n--- goroutine stack trace ---\n")
	b.Write(stack[:n])
	if stack[n-1] != '\n' {
		b.WriteString("\n")
	}

	return os.WriteFile(path, []byte(b.String()), 0o644)
}
