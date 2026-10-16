package ui

import (
	"fmt"
	"strings"

	"som/internal/tui/logbuf"

	"charm.land/lipgloss/v2"
)

var LogBuf = logbuf.New(logbuf.DefaultCapacity)

func renderLogsView(logOffset int, w, h int) string {
	borderColor := colorWhite

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
	if end < 0 {
		end = 0
	} else if end > len(lines) {
		end = len(lines)
	}

	visible := make([]string, 0, innerH)
	for i := end - 1; i >= 0 && len(visible) < innerH; i-- {
		wrapped := wordWrap(lines[i], innerW)
		for j := len(wrapped) - 1; j >= 0 && len(visible) < innerH; j-- {
			visible = append(visible, wrapped[j])
		}
	}
	for i, j := 0, len(visible)-1; i < j; i, j = i+1, j-1 {
		visible[i], visible[j] = visible[j], visible[i]
	}

	rows := make([]string, innerH)
	tagStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	for i := range rows {
		rows[i] = strings.Repeat(" ", innerW)
		if i < len(visible) {
			line := visible[i]
			start := strings.IndexByte(line, '[')
			end := strings.IndexByte(line[start+1:], ']')
			if start >= 0 && end >= 0 {
				end += start + 2
				rows[i] = DimItemStyle.Render(line[:start]) +
					tagStyle.Render(line[start:end]) +
					DimItemStyle.Render(line[end:])
			} else {
				rows[i] = DimItemStyle.Render(line)
			}
		}
	}

	content := strings.Join(rows, "\n")
	return renderBox(w, fmt.Sprintf("Logs (%d)", len(lines)), content, borderColor)
}
