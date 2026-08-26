package ui

import (
	"fmt"
	"strings"

	"som/internal/tui/logbuf"

	"charm.land/lipgloss/v2"
)

var LogBuf = logbuf.New(logbuf.DefaultCapacity)

func renderLogsView(logOffset int, w, h int, focused bool) string {
	borderColor := lipgloss.Color("#7c7986")
	if focused {
		borderColor = lipgloss.Color("#e8593c")
	}
	innerW := w - 4
	innerH := h - 2

	lines := LogBuf.Lines()
	if len(lines) == 0 {
		pad := innerW/2 - 5
		if pad < 0 {
			pad = 0
		}
		content := lipgloss.NewStyle().Width(innerW).Height(innerH).
			Render(strings.Repeat(" ", pad) + "No logs yet.")
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

	for i := len(visible); i < innerH; i++ {
		b.WriteString(strings.Repeat(" ", innerW) + "\n")
	}

	content := b.String()
	return renderBox(w, fmt.Sprintf("Logs (%d)", len(lines)), content, borderColor)
}
