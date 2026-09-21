package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
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

func (p LeftPanel) ViewDownloadsContent(w, h int, selected map[string]bool, selectMode bool, alreadyIn map[string]bool) string {
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
	listContent := p.renderLocalList(innerW, selected, selectMode, alreadyIn)
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

func (p LeftPanel) renderLocalList(innerW int, selected map[string]bool, selectMode bool, alreadyIn map[string]bool) string {
	locals := p.getFilteredLocals()
	if len(locals) == 0 {
		if p.input.Focused() && strings.TrimSpace(p.input.Value()) != "" {
			return DimItemStyle.Render(" No matching downloaded files found.") + "\n"
		}
		return DimItemStyle.Render(" No downloaded files in "+p.downloadDir+"/") + "\n"
	}

	return renderSharedTrackList(
		innerW, locals, p.dlCursor, p.dlOffset, p.visibleRows()+1,
		func(f LocalFile) (string, string, string, int, bool) {
			return f.Name, f.Artist, f.Path, f.Duration, false
		},
		selectMode, selected, alreadyIn,
	)
}

func localFileSidecar(path string) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + ".json"
}
