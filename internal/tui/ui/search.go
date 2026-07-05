package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (p LeftPanel) ViewSearchContent(w, h int) string {
	borderColor := lipgloss.Color("#7c7986")

	innerW := w - 4

	// Search input box (thin border)
	inputRow := " " + p.input.View()
	if p.loading {
		inputRow += " " + p.spinner.View()
	}
	inputContent := lipgloss.NewStyle().Width(innerW).Render(inputRow)
	searchBox := renderBox(w, "Search", inputContent, borderColor)

	// Results box (normal border, no title)
	var resultContent string
	if p.errMsg != "" {
		resultContent = StatusErrStyle.Render("X " + p.errMsg)
	} else if len(p.tracks) > 0 {
		var statusLine string
		if p.loadingStream {
			statusLine = DimItemStyle.Render(" " + p.spinner.View() + " Loading...") + "\n"
		} else if p.loadingDownload {
			statusLine = DimItemStyle.Render(" " + p.spinner.View() + " Downloading...") + "\n"
		}
		resultContent = statusLine + p.renderSearchList(innerW)
	} else if !p.searched {
		padLeft := (innerW) / 2
		if padLeft < 0 {
			padLeft = 0
		}
		resultContent = strings.Repeat(" ", padLeft) + "\n"
	} else {
		resultContent = DimItemStyle.Render(" No results.") + "\n"
	}
	resultsBox := renderBox(w, "", resultContent, borderColor)

	return searchBox + "\n" + resultsBox
}

func (p LeftPanel) renderSearchList(innerW int) string {
	if !p.searched {
		padLeft := (innerW - len([]rune(" Type to search…"))) / 2
		if padLeft < 0 {
			padLeft = 0
		}
		return strings.Repeat(" ", padLeft) + SubtitleStyle.Render(" Type to search…") + "\n"
	}
	if len(p.tracks) == 0 {
		return DimItemStyle.Render(" No results.") + "\n"
	}
	var b strings.Builder
	vis := p.visibleRows()
	end := p.offset + vis
	if end > len(p.tracks) {
		end = len(p.tracks)
	}
	titleW := innerW - 12
	if titleW < 10 {
		titleW = 10
	}

	for i := p.offset; i < end; i++ {
		t := p.tracks[i]
		mark := "  "
		if i == p.cursor {
			mark = ""
		}
		safeTitle := truncate(t.Title, titleW)
		titleBlock := lipgloss.NewStyle().Width(titleW).Render(safeTitle)
		durationBlock := FormatDuration(t.Duration)

		line := mark + titleBlock + " " + durationBlock

		if i == p.cursor {
			b.WriteString(SelectedItemStyle.Width(innerW).Render(line))
		} else {
			b.WriteString(NormalItemStyle.Width(innerW).Render(line))
		}
		b.WriteString("\n")
	}
	if len(p.tracks) > vis {
		b.WriteString(DimItemStyle.Render(fmt.Sprintf(" %d/%d", p.cursor+1, len(p.tracks))))
		b.WriteString("\n")
	}
	return b.String()
}
