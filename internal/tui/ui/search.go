package ui

import (
	"som/internal/domain"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
)

func (p LeftPanel) ViewSearchContent(w, h int) string {
	innerW := w - 4

	inputFocused := p.input.Focused()
	searchBorder := themeCol("#7c7986")
	contentBorder := themeCol("#7c7986")
	if inputFocused {
		searchBorder = themeCol("#e8593c")
	} else {
		contentBorder = themeCol("#e8593c")
	}

	inputRow := " " + p.input.View()
	if p.loading {
		inputRow += " " + p.spinner.View()
	}

	inputContent := lipgloss.NewStyle().Width(innerW).Render(inputRow)
	searchBox := renderBox(w, "Search", inputContent, searchBorder)

	var suggestionBox string
	if inputFocused && len(p.suggestions) > 0 {
		suggestionBox = renderBox(w, "Suggestions", p.renderSuggestions(innerW), themeCol("#7c7986"))
	}

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

		// Gọi hàm dùng chung
		list := renderSharedTrackList(
			innerW, p.tracks, p.searchCursor, p.searchOffset, p.visibleRows(),
			func(t domain.Track) (string, string, string, int, bool) {
				// Check xem bài này đã down chưa để trả về true/false cho showCheck
				return t.Title, t.Artist, t.ID, t.Duration, p.isDownloaded(t)
			},
			false, nil, nil,
		)
		resultContent = statusLine + list
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

	if suggestionBox != "" {
		return searchBox + "\n" + suggestionBox + "\n" + resultsBox
	}
	return searchBox + "\n" + resultsBox
}

func (p LeftPanel) renderSuggestions(innerW int) string {
	show := suggestMaxShow
	if len(p.suggestions)-p.suggestOffset < show {
		show = len(p.suggestions) - p.suggestOffset
	}
	if show < 0 {
		show = 0
	}
	titleW := innerW - 2
	if titleW < 10 {
		titleW = 10
	}
	var b strings.Builder
	for idx := p.suggestOffset; idx < p.suggestOffset+show; idx++ {
		mark := " "
		text := truncate(strings.TrimSpace(p.suggestions[idx]), titleW)
		line := mark + " " + text
		pad := innerW - runewidth.StringWidth(line)
		if pad < 0 {
			pad = 0
		}
		if p.suggestFocus && idx == p.suggestCursor {
			b.WriteString(SelectedItemStyle.Render(line + strings.Repeat(" ", pad)))
		} else {
			b.WriteString(NormalItemStyle.Render(line + strings.Repeat(" ", pad)))
		}
		b.WriteString("\n")
	}
	return b.String()
}
