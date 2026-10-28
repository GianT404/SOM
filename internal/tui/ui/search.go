package ui

import (
	"som/internal/domain"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
)

const searchPageSize = 7

func (p LeftPanel) ViewSearchContent(w, h int, playingID string) string {
	innerW := w - 4

	inputFocused := p.input.Focused()
	searchBorder := themeCol("#7c7986")
	if inputFocused {
		searchBorder = themeCol("#e8593c")
	}

	inputRow := " " + p.input.View()

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
		visibleRows := p.visibleRows()
		showLoadMore := p.hasMore && len(p.tracks) < p.searchOffset+visibleRows
		listRows := visibleRows
		if showLoadMore && listRows > 1 {
			listRows--
		}
		list := renderSharedTrackList(
			innerW,
			p.tracks,
			p.searchCursor,
			p.searchOffset,
			listRows,
			func(i int, t domain.Track) (string, string, string, int, bool, bool, int) {
				return t.Title, t.Artist, t.ID, t.Duration, p.isDownloaded(t), t.ID == playingID && playingID != "", i + 1
			},
			false, nil, nil,
		)
		if showLoadMore {
			label := "  Load more..."
			style := DimItemStyle
			if p.searchCursor == len(p.tracks) {
				style = SelectedItemStyle
			}
			list += "\n" + style.Width(innerW).Render(label)
		}
		resultContent = list
	} else if p.loading || !p.searched {
		resultContent = "\n"
	} else {
		resultContent = DimItemStyle.Render(" No results.") + "\n"
	}

	resultsPadded := lipgloss.NewStyle().PaddingLeft(2).Render(resultContent)

	if suggestionBox != "" {
		return searchBox + "\n" + suggestionBox + "\n" + resultsPadded
	}
	return searchBox + "\n" + resultsPadded
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
	return strings.TrimSuffix(b.String(), "\n")
}
