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

	var pinned []LocalFile
	var unpinned []LocalFile
	pinnedMap := make(map[string]LocalFile)
	for _, f := range files {
		lf := LocalFile{
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
		if p.isPinned(f.Path) {
			pinnedMap[f.Path] = lf
		} else {
			unpinned = append(unpinned, lf)
		}
	}

	var validPinned []string
	for _, path := range p.pinned {
		if lf, ok := pinnedMap[path]; ok {
			pinned = append(pinned, lf)
			validPinned = append(validPinned, path)
		}
	}

	if len(validPinned) != len(p.pinned) {
		p.pinned = validPinned
		p.savePinned()
	}

	p.locals = append(pinned, unpinned...)
}
func (p LeftPanel) ViewDownloadsContent(w, h int, selected map[string]bool, selectMode bool, alreadyIn map[string]bool, playingID string) string {
	innerW := w - 4
	listContent := lipgloss.NewStyle().
		PaddingLeft(1).
		Render(
			p.renderLocalList(innerW, selected, selectMode, alreadyIn, playingID),
		)

	if p.isSearchVisible() {
		inputFocused := p.input.Focused()
		searchBorder := themeCol("#7c7986")
		if inputFocused {
			searchBorder = themeCol("#e8593c")
		}

		var searchContent strings.Builder
		inputRow := " " + p.input.View()
		if p.loading {
			inputRow += " " + p.spinner.View()
		}

		searchContent.WriteString(
			lipgloss.NewStyle().
				Width(w - 7).
				Render(inputRow),
		)

		if p.errMsg != "" {
			searchContent.WriteString(
				StatusErrStyle.Render("X " + p.errMsg),
			)
		}

		count := len(p.getFilteredLocals())
		title := fmt.Sprintf("Search (%d)", count)
		if selectMode {
			title = fmt.Sprintf(
				"Move to playlist (%d) -  %d",
				count,
				countSelected(selected),
			)
		}

		searchBox := lipgloss.NewStyle().
			PaddingLeft(0).
			Render(
				renderBox(w-3, title, searchContent.String(), searchBorder),
			)

		return searchBox + "\n" + listContent
	}

	return lipgloss.NewStyle().
		Width(w).
		Render(listContent)
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

func (p LeftPanel) renderLocalList(innerW int, selected map[string]bool, selectMode bool, alreadyIn map[string]bool, playingID string) string {
	locals := p.getFilteredLocals()
	if len(locals) == 0 {
		if p.input.Focused() && strings.TrimSpace(p.input.Value()) != "" {
			return DimItemStyle.Render(" No matching downloaded files found.") + "\n"
		}
		return DimItemStyle.Render(" No downloaded files in "+p.downloadDir+"/") + "\n"
	}

	return renderSharedTrackList(
		innerW,
		locals,
		p.dlCursor,
		p.dlOffset,
		p.visibleRows()+1,
		func(i int, f LocalFile) (string, string, string, int, bool, bool, int) {
			isPlaying := playingID == "local:"+f.Path

			origIdx := i + 1
			for j, lf := range p.locals {
				if lf.Path == f.Path {
					origIdx = j + 1
					break
				}
			}

			title := f.Name
			if p.isPinned(f.Path) {
				title = "[P] " + title
			}

			return title, f.Artist, f.Path, f.Duration, false, isPlaying, origIdx
		},
		selectMode, selected, alreadyIn,
	)
}

func localFileSidecar(path string) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + ".json"
}
