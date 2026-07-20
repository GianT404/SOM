package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

func (p LeftPanel) ViewSearchContent(w, h int) string {
	innerW := w - 4

	inputFocused := p.input.Focused()
	searchBorder := lipgloss.Color("#7c7986")
	contentBorder := lipgloss.Color("#7c7986")
	if inputFocused {
		searchBorder = lipgloss.Color("#e8593c")
	} else {
		contentBorder = lipgloss.Color("#e8593c")
	}

	inputRow := " " + p.input.View()
	if p.loading {
		inputRow += " " + p.spinner.View()
	}
	inputContent := lipgloss.NewStyle().Width(innerW).Render(inputRow)
	searchBox := renderBox(w, "Search", inputContent, searchBorder)

	var resultContent string
	if p.errMsg != "" {
		resultContent = StatusErrStyle.Render("X " + p.errMsg)
	} else if len(p.tracks) > 0 {
		var statusLine string
		if p.loadingStream {
			statusLine = DimItemStyle.Render(" "+p.spinner.View()+" Loading...") + "\n"
		} else if p.loadingDownload {
			statusLine = DimItemStyle.Render(" "+p.spinner.View()+" Downloading...") + "\n"
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
	resultsBox := renderBox(w, "", resultContent, contentBorder)

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
		titlePlain := runewidth.FillRight(safeTitle, titleW)
		durationBlock := FormatDuration(t.Duration)
		downloaded := p.isDownloaded(t)

		checkPlaceholder := " "
		plainLine := mark + titlePlain + " " + durationBlock + " " + checkPlaceholder
		pad := innerW - runewidth.StringWidth(plainLine)
		if pad < 0 {
			pad = 0
		}

		rowStyle := NormalItemStyle
		if i == p.cursor {
			rowStyle = SelectedItemStyle
		}

		before := rowStyle.Render(mark + titlePlain + " " + durationBlock + " ")
		var checkFrag string
		if downloaded {
			checkStyle := rowStyle.Foreground(colorDark)
			checkFrag = checkStyle.Render(IconCheck)
		} else {
			checkFrag = rowStyle.Render(" ")
		}
		after := rowStyle.Render(strings.Repeat(" ", pad))

		b.WriteString(before)
		b.WriteString(checkFrag)
		b.WriteString(after)
		b.WriteString("\n")
	}
	if len(p.tracks) > vis {
		b.WriteString(DimItemStyle.Render(fmt.Sprintf(" %d/%d", p.cursor+1, len(p.tracks))))
		b.WriteString("\n")
	}
	return b.String()
}
