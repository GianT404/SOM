package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
)

func (p *LeftPanel) scanLocalFiles() {
	p.locals = nil
	if p.plStore == nil {
		return
	}
	files, err := p.plStore.ListAllLocalFilesSorted(p.sortPref)
	if err != nil {
		p.errMsg = "DB error: " + err.Error()
		return
	}
	p.locals = make([]LocalFile, len(files))
	for i, f := range files {
		p.locals[i] = LocalFile{
			Name:      f.Name,
			Path:      f.Path,
			Artist:    f.Artist,
			Duration:  f.Duration,
			VideoID:   f.VideoID,
			Thumbnail: f.Thumbnail,
			FileSize:  f.FileSize,
			FileMTime: f.FileMTime,
			CreatedAt: f.CreatedAt,
		}
	}
}

func (p LeftPanel) ViewDownloadsContent(w, h int, selected map[string]bool, selectMode bool) string {
	innerW := w - 4

	inputFocused := p.input.Focused()
	searchBorder := themeCol("#7c7986")
	contentBorder := themeCol("#7c7986")
	if inputFocused {
		searchBorder = themeCol("#e8593c")
	} else {
		contentBorder = themeCol("#e8593c")
	}

	// ─── Search box ───────────────────────────
	var searchContent strings.Builder
	inputRow := " " + p.input.View()
	if p.loading {
		inputRow += " " + p.spinner.View()
	}
	searchContent.WriteString(lipgloss.NewStyle().Width(innerW).Render(inputRow))

	if p.errMsg != "" {
		searchContent.WriteString(StatusErrStyle.Render("X " + p.errMsg))
	}

	searchBox := renderBox(w, "Search", searchContent.String(), searchBorder)

	// ─── Playlist box ─────────────────────────
	count := len(p.getFilteredLocals())
	listContent := p.renderLocalList(innerW, selected, selectMode)
	title := fmt.Sprintf("Playlist (%d)", count)
	if selectMode {
		title = fmt.Sprintf("Move to playlist (%d) -  %d", count, countSelected(selected))
	}
	playlistBox := renderBox(w, title, listContent, contentBorder)

	return searchBox + "\n" + playlistBox
}

func countSelected(m map[string]bool) int {
	n := 0
	for _, v := range m {
		if v {
			n++
		}
	}
	return n
}

func (p LeftPanel) renderLocalList(innerW int, selected map[string]bool, selectMode bool) string {
	locals := p.getFilteredLocals()
	if len(locals) == 0 {
		if p.input.Focused() && strings.TrimSpace(p.input.Value()) != "" {
			return DimItemStyle.Render(" No matching downloaded files found.") + "\n"
		}
		return DimItemStyle.Render(" No downloaded files in "+p.downloadDir+"/") + "\n"
	}
	var b strings.Builder
	// +1: reclaim the row previously wasted by the trailing blank line below
	vis := p.visibleRows() + 1
	end := p.dlOffset + vis
	if end > len(locals) {
		end = len(locals)
	}
	idxW := 3
	if len(locals) >= 1000 {
		idxW = 4
	}
	durW := 6
	artistW := 27
	// selectMode
	tickW := 0
	if selectMode {
		tickW = 4
	}
	titleW := innerW - tickW - idxW - artistW - durW - 8
	if titleW < 10 {
		titleW = 10
		artistW = innerW - tickW - idxW - titleW - durW - 8
		if artistW < 0 {
			artistW = 0
		}
	}
	headerTick := ""
	if selectMode {
		headerTick = "    "
	}
	header := fmt.Sprintf("%s  %*s  %-*s  %-*s  %*s", headerTick, idxW, "#", titleW, "Title", artistW, "Artist", durW-1, "Time")
	b.WriteString(DimItemStyle.Width(innerW).Render(header))
	for i := p.dlOffset; i < end; i++ {
		f := locals[i]
		mark := "  "
		if i == p.dlCursor {
			mark = " "
		}
		tick := ""
		if selectMode {
			if selected[f.Path] {
				tick = "[+] "
			} else {
				tick = "[ ] "
			}
		}
		idx := fmt.Sprintf("%*d", idxW, i+1)
		title := runewidth.FillRight(truncate(f.Name, titleW), titleW)
		safeArtist := truncate(f.Artist, artistW)
		artistPlain := runewidth.FillRight(safeArtist, artistW)
		dur := fmt.Sprintf("%*s", durW, FormatDuration(f.Duration))
		line := mark + tick + idx + "  " + title + "  " + artistPlain + "  " + dur
		b.WriteString("\n")
		if i == p.dlCursor {
			b.WriteString(LocalFileSelectedStyle.Width(innerW).Render(line))
		} else {
			b.WriteString(LocalFileStyle.Width(innerW).Render(line))
		}
	}

	return b.String()
}

func localFileSidecar(path string) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + ".json"
}
