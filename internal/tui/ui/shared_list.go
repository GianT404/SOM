package ui

import (
	"fmt"
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

	tickW := 0
	headerTick := "  " // 2 ô lề trái
	if selectMode {
		tickW = 4
		headerTick = "      " // 2 ô lề trái + 4 ô tick
	}

	// Đặt kích thước CỐ ĐỊNH
	checkW := 2
	idxW := 3
	timeW := 5

	// Tổng các khoảng phân cách: prefix(2) + sau idx(2) + sau title(2) + sau artist(2) = 8
	spacing := 8
	availW := innerW - tickW - checkW - idxW - timeW - spacing
	if availW < 10 {
		availW = 10
	}

	// Title 70%, Artist 30%
	titleW := int(float64(availW) * 0.75)
	artistW := availW - titleW

	header := fmt.Sprintf("%s%-*s  %-*s  %-*s  %*s", headerTick, idxW, "#", titleW, "Title", artistW, "Artist", checkW+timeW, "Time")

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
		checkStr = runewidth.FillRight(checkStr, checkW)

		idx := fmt.Sprintf("%-*d", idxW, displayIdx)
		safeTitle := runewidth.FillRight(truncate(title, titleW), titleW)
		safeArtist := runewidth.FillRight(truncate(artist, artistW), artistW)
		dur := fmt.Sprintf("%*s", timeW, FormatDuration(durSec))

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
