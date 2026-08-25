package ui

import (
	"fmt"
	"strings"

	"som/internal/domain"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

func (p LeftPanel) ViewQueueContent(w, h int, queue []domain.Track) string {
	innerW := w - 4

	contentBorder := lipgloss.Color("#7c7986")

	count := len(queue)
	listContent := p.renderQueueList(innerW, queue)
	box := renderBox(w, fmt.Sprintf("Queue (%d)", count), listContent, contentBorder)

	return box
}

func (p LeftPanel) renderQueueList(innerW int, queue []domain.Track) string {
	if len(queue) == 0 {
		return DimItemStyle.Render(" No tracks in queue.") + "\n" +
			DimItemStyle.Render(" Use ':' -> 'Add to queue'") + "\n"
	}

	var b strings.Builder
	vis := p.visibleRows() + 1
	end := p.qOffset + vis
	if end > len(queue) {
		end = len(queue)
	}

	idxW := 3
	durW := 6
	artistW := 27
	titleW := innerW - idxW - artistW - durW - 8
	if titleW < 10 {
		titleW = 10
		artistW = innerW - idxW - titleW - durW - 8
		if artistW < 0 {
			artistW = 0
		}
	}
	header := fmt.Sprintf("  %*s  %-*s  %-*s  %*s", idxW, "#", titleW, "Title", artistW, "Artist", durW-1, "Time")
	b.WriteString(DimItemStyle.Width(innerW).Render(header))

	for i := p.qOffset; i < end; i++ {
		t := queue[i]
		mark := "  "
		if i == p.qCursor {
			mark = " "
		}
		idx := fmt.Sprintf("%*d", idxW, i+1)
		title := runewidth.FillRight(truncate(t.Title, titleW), titleW)
		safeArtist := truncate(t.Artist, artistW)
		artistPlain := runewidth.FillRight(safeArtist, artistW)
		dur := fmt.Sprintf("%*s", durW, FormatDuration(t.Duration))
		line := mark + idx + "  " + title + "  " + artistPlain + "  " + dur
		b.WriteString("\n")
		if i == p.qCursor {
			b.WriteString(LocalFileSelectedStyle.Width(innerW).Render(line))
		} else {
			b.WriteString(LocalFileStyle.Width(innerW).Render(line))
		}
	}

	return b.String()
}
