package ui

import (
	"fmt"
	"strings"

	"som/internal/storage"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
)

// Lấy danh sách Playlist đã qua bộ lọc tìm kiếm
func (p LeftPanel) getFilteredPlaylists() []storage.Playlist {
	q := strings.ToLower(strings.TrimSpace(p.input.Value()))
	if q == "" {
		return p.playlists
	}
	tokens := strings.Fields(q)
	var filtered []storage.Playlist
	for _, pl := range p.playlists {
		hay := strings.ToLower(pl.Name)
		match := true
		for _, tok := range tokens {
			if !strings.Contains(hay, tok) {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, pl)
		}
	}
	return filtered
}

// Lấy danh sách bài hát trong Playlist hiện tại đã qua bộ lọc tìm kiếm
func (p LeftPanel) getFilteredPlaylistTracks() []storage.PlaylistTrack {
	if p.activePlaylist == nil {
		return nil
	}
	q := strings.ToLower(strings.TrimSpace(p.input.Value()))
	if q == "" {
		return p.activePlaylist.Tracks
	}
	tokens := strings.Fields(q)
	var filtered []storage.PlaylistTrack
	for _, t := range p.activePlaylist.Tracks {
		hay := strings.ToLower(t.Title + " " + t.Artist)
		match := true
		for _, tok := range tokens {
			if !strings.Contains(hay, tok) {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func (p LeftPanel) ViewPlaylistsContent(w, h int) string {
	innerW := w - 4
	inputFocused := p.input.Focused()

	searchBorder := themeCol("#7c7986")
	contentBorder := themeCol("#7c7986")
	if inputFocused {
		searchBorder = themeCol("#e8593c")
	} else {
		contentBorder = themeCol("#e8593c")
	}

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

	var listContent string
	var title string

	if p.activePlaylist != nil {
		filtered := p.getFilteredPlaylistTracks()
		title = fmt.Sprintf("Playlist: %s (%d)", p.activePlaylist.Name, len(filtered))
		listContent = p.renderPlaylistDetail(innerW, filtered)
	} else {
		filtered := p.getFilteredPlaylists()
		title = fmt.Sprintf("Playlists (%d)", len(filtered))
		listContent = p.renderPlaylistList(innerW, filtered)
	}

	contentBox := renderBox(w, title, listContent, contentBorder)
	return searchBox + "\n" + contentBox
}

func (p LeftPanel) renderPlaylistList(innerW int, filtered []storage.Playlist) string {
	if len(filtered) == 0 {
		if p.input.Value() != "" {
			return DimItemStyle.Render(" No matching playlists.")
		}
		return DimItemStyle.Render(" No playlists available. Press '/' to create a new playlist.")
	}

	var b strings.Builder
	vis := p.visibleRows() + 1
	end := p.plOffset + vis
	if end > len(filtered) {
		end = len(filtered)
	}

	idxW := 3
	countW := 10
	nameW := innerW - idxW - countW - 6
	if nameW < 10 {
		nameW = 10
	}

	header := fmt.Sprintf("  %*s  %-*s  %*s", idxW, "#", nameW, "Name", countW, "Tracks")
	b.WriteString(DimItemStyle.Width(innerW).Render(header))
	// Đã xóa ký tự b.WriteString("\n") thừa ở vị trí này

	for i := p.plOffset; i < end; i++ {
		pl := filtered[i]
		mark := "  "
		if i == p.plCursor {
			mark = " "
		}

		idx := fmt.Sprintf("%*d", idxW, i+1)
		name := runewidth.FillRight(truncate(pl.Name, nameW), nameW)
		count := fmt.Sprintf("%*s", countW, fmt.Sprintf("%d songs", len(pl.Tracks)))

		line := mark + idx + "  " + name + "  " + count

		// Ghi ký tự xuống dòng TRƯỚC khi vẽ item, để không bị dư 1 dòng rỗng ở cuối danh sách
		b.WriteString("\n")
		if i == p.plCursor {
			b.WriteString(SelectedItemStyle.Width(innerW).Render(line))
		} else {
			b.WriteString(NormalItemStyle.Width(innerW).Render(line))
		}
	}

	return b.String()
}

func (p LeftPanel) renderPlaylistDetail(innerW int, filtered []storage.PlaylistTrack) string {
	if len(filtered) == 0 {
		if len(p.activePlaylist.Tracks) == 0 {
			return DimItemStyle.Render(" This playlist is empty. Press ':' on a track to move it here.")
		}
		return DimItemStyle.Render(" No matching tracks.")
	}

	var b strings.Builder
	vis := p.visibleRows() + 1
	end := p.plOffset + vis
	if end > len(filtered) {
		end = len(filtered)
	}

	idxW := 3
	if len(filtered) >= 1000 {
		idxW = 4
	}
	durW := 6
	artistW := 27
	titleW := innerW - idxW - artistW - durW - 8
	if titleW < 10 {
		titleW = 10
	}
	artistW = innerW - idxW - titleW - durW - 8
	if artistW < 0 {
		artistW = 0
	}

	header := fmt.Sprintf("  %*s  %-*s  %-*s  %*s", idxW, "#", titleW, "Title", artistW, "Artist", durW-1, "Time")
	b.WriteString(DimItemStyle.Width(innerW).Render(header))

	for i := p.plOffset; i < end; i++ {
		t := filtered[i]
		mark := "  "
		if i == p.plCursor {
			mark = " "
		}

		idx := fmt.Sprintf("%*d", idxW, i+1)
		title := runewidth.FillRight(truncate(t.Title, titleW), titleW)
		safeArtist := truncate(t.Artist, artistW)
		artistPlain := runewidth.FillRight(safeArtist, artistW)
		dur := fmt.Sprintf("%*s", durW, FormatDuration(t.Duration))

		line := mark + idx + "  " + title + "  " + artistPlain + "  " + dur
		b.WriteString("\n")
		if i == p.plCursor {
			b.WriteString(LocalFileSelectedStyle.Width(innerW).Render(line))
		} else {
			b.WriteString(LocalFileStyle.Width(innerW).Render(line))
		}
	}

	return b.String()
}

func (p LeftPanel) renderPlInputPopup() string {
	var b strings.Builder
	b.WriteString("\n ")
	b.WriteString(p.plInput.View())
	b.WriteString("\n\n")
	b.WriteString(DimItemStyle.Render(" (enter: create  | esc: cancel)"))
	return renderBox(40, "New Playlist", b.String(), themeCol("#e8593c"))
}

func (p LeftPanel) renderDeletePopup() string {
	var b strings.Builder
	b.WriteString(p.deleteMsg + "\n\n")
	cancelStyle := NormalItemStyle
	confirmStyle := NormalItemStyle
	if p.deletePopupCursor == 0 {
		cancelStyle = SelectedItemStyle
	} else {
		confirmStyle = SelectedItemStyle.Foreground(deleteColor)
	}
	b.WriteString(fmt.Sprintf("  %s     %s", cancelStyle.Render("[ Cancel ]"), confirmStyle.Render("[ Delete ]")))
	return renderBox(45, "Confirm", b.String(), themeCol("#E24B4A"))
}
