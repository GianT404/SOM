package ui

import (
	"fmt"

	"som/internal/tui/layout"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
)

func renderSharedTrackList[T any](
	innerW int,
	items []T,
	cursor int,
	offset int,
	visibleRows int,
	extract func(i int, item T) (title, artist, path string, duration int, showCheck bool, isPlaying bool, displayIdx int),
	selectMode bool,
	selected map[string]bool,
	alreadyIn map[string]bool,
) string {
	var b strings.Builder

	end := offset + visibleRows
	if end > len(items) {
		end = len(items)
	}

	layout := layout.NewTracklistLayout(innerW, selectMode)
	headerTick := "  "
	if selectMode {
		headerTick = "      "
	}

	header := fmt.Sprintf("%s%-*s  %-*s  %-*s  %*s", headerTick, layout.IndexWidth, "#", layout.TitleWidth, "Title", layout.ArtistWidth, "Artist", layout.CheckWidth+layout.TimeWidth, "Time")

	b.WriteString(DimItemStyle.Render(header))
	b.WriteString("\n")

	lineStyle := lipgloss.NewStyle().Foreground(themeCol("#7c7986"))
	b.WriteString(lineStyle.Render(strings.Repeat("─", innerW)))

	for i := offset; i < end; i++ {
		title, artist, path, durSec, showCheck, isPlaying, displayIdx := extract(i, items[i])
		prefix := "  "
		if i == cursor || isPlaying {
			prefix = " "
		}

		tick := ""
		if selectMode {
			if alreadyIn[path] != selected[path] {
				tick = "[+] "
			} else {
				tick = "[ ] "
			}
		}

		checkStr := ""
		if showCheck {
			checkStr = IconCheck + " "
		}
		checkStr = runewidth.FillRight(checkStr, layout.CheckWidth)

		idx := fmt.Sprintf("%-*d", layout.IndexWidth, displayIdx)
		safeTitle := runewidth.FillRight(truncate(title, layout.TitleWidth), layout.TitleWidth)
		safeArtist := runewidth.FillRight(truncate(artist, layout.ArtistWidth), layout.ArtistWidth)
		dur := fmt.Sprintf("%*s", layout.TimeWidth, FormatDuration(durSec))

		line := prefix + tick + idx + "  " + safeTitle + "  " + safeArtist + "  " + checkStr + dur

		b.WriteString("\n")

		style := NormalItemStyle
		if i == cursor {
			style = SelectedItemStyle
		}

		if isPlaying {
			style = style.Foreground(themeCol("#fff")).Bold(true)
		}

		b.WriteString(style.Width(innerW).Render(line))
	}

	return b.String()
}
