package ui

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
)

func renderSharedTrackList[T any](
	innerW int,
	items []T,
	cursor int,
	offset int,
	visibleRows int,
	extract func(T) (title, artist, path string, duration int),
	selectMode bool,
	selected map[string]bool,
	alreadyIn map[string]bool,
) string {
	var b strings.Builder
	end := offset + visibleRows
	if end > len(items) {
		end = len(items)
	}

	idxW := 3
	if len(items) >= 1000 {
		idxW = 4
	}
	durW := 6
	artistW := 27

	tickW := 0
	if selectMode {
		tickW = 8
	}

	titleW := innerW - tickW - idxW - artistW - durW - 8
	if titleW < 10 {
		titleW = 10
	}
	artistW = innerW - tickW - idxW - titleW - durW - 8
	if artistW < 0 {
		artistW = 0
	}

	headerTick := ""
	if selectMode {
		headerTick = "    "
	}

	header := fmt.Sprintf("%s  %*s  %-*s  %-*s  %*s", headerTick, idxW, "#", titleW, "Title", artistW, "Artist", durW-1, "Time")
	b.WriteString(DimItemStyle.Width(innerW).Render(header))

	for i := offset; i < end; i++ {
		title, artist, path, durSec := extract(items[i])

		mark := "  "
		if i == cursor {
			mark = " "
		}

		tick := ""
		if selectMode {
			isAlreadyIn := alreadyIn[path]
			isToggled := selected[path]
			willBeIn := isAlreadyIn != isToggled

			if willBeIn {
				tick = "[+] "
			} else {
				tick = "[ ] "
			}
		}

		idx := fmt.Sprintf("%*d", idxW, i+1)
		safeTitle := runewidth.FillRight(truncate(title, titleW), titleW)
		safeArtist := runewidth.FillRight(truncate(artist, artistW), artistW)
		dur := fmt.Sprintf("%*s", durW, FormatDuration(durSec))

		line := mark + tick + idx + "  " + safeTitle + "  " + safeArtist + "  " + dur

		b.WriteString("\n")
		if i == cursor {
			b.WriteString(LocalFileSelectedStyle.Width(innerW).Render(line))
		} else {
			b.WriteString(LocalFileStyle.Width(innerW).Render(line))
		}
	}
	return b.String()
}
