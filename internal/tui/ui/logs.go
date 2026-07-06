package ui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

// LogBuffer is a thread-safe ring buffer that captures log output.
type LogBuffer struct {
	mu    sync.Mutex
	lines []string
	max   int
}

func NewLogBuffer(max int) *LogBuffer {
	return &LogBuffer{max: max}
}

func (b *LogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = append(b.lines, strings.TrimRight(string(p), "\n"))
	if len(b.lines) > b.max {
		b.lines = b.lines[len(b.lines)-b.max:]
	}
	return len(p), nil
}

func (b *LogBuffer) Lines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]string, len(b.lines))
	copy(result, b.lines)
	return result
}

func (b *LogBuffer) Len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.lines)
}

var LogBuf = NewLogBuffer(500)

func renderLogsView(logOffset int, w, h int) string {
	borderColor := lipgloss.Color("#7c7986")
	innerW := w - 4
	innerH := h - 2

	lines := LogBuf.Lines()
	if len(lines) == 0 {
		content := lipgloss.NewStyle().Width(innerW).Height(innerH).
			Render(strings.Repeat(" ", innerW/2-5) + "No logs yet.")
		return renderBox(w, "Logs", content, borderColor)
	}

	end := len(lines) - logOffset
	start := end - innerH
	if start < 0 {
		start = 0
	}
	visible := lines[start:end]

	var b strings.Builder
	for _, line := range visible {
		trunc := line
		if len([]rune(trunc)) > innerW {
			trunc = string([]rune(trunc)[:innerW])
		}
		b.WriteString(DimItemStyle.Width(innerW).Render(trunc))
		b.WriteString("\n")
	}

	// pad remaining lines
	for i := len(visible); i < innerH; i++ {
		b.WriteString(strings.Repeat(" ", innerW) + "\n")
	}

	content := b.String()
	return renderBox(w, fmt.Sprintf("Logs (%d)", len(lines)), content, borderColor)
}
