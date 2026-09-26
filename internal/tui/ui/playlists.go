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

func (p LeftPanel) ViewPlaylistsContent(w, h int, playingID string) string {
	innerW := w - 4

	var listContent string
	var title string

	if p.activePlaylist != nil {
		filtered := p.getFilteredPlaylistTracks()
		title = fmt.Sprintf("Playlist: %s (%d)", p.activePlaylist.Name, len(filtered))
		listContent = lipgloss.NewStyle().
			PaddingLeft(1).
			Render(p.renderPlaylistDetail(innerW, filtered, playingID))
	} else {
		filtered := p.getFilteredPlaylists()
		title = fmt.Sprintf("Playlists (%d)", len(filtered))
		listContent = lipgloss.NewStyle().
			PaddingLeft(1).
			Render(p.renderPlaylistList(innerW, filtered, playingID))
	}

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

		searchBox := lipgloss.NewStyle().
			PaddingLeft(0).
			Render(
				renderBox(w-3, title, searchContent.String(), searchBorder),
			)

		return lipgloss.NewStyle().
			Width(w).
			Render(searchBox + "\n" + listContent)
	}

	return lipgloss.NewStyle().
		Width(w).
		Render(listContent)
}
func (p LeftPanel) renderPlaylistList(innerW int, filtered []storage.Playlist, playingId string) string {
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

	// spacing cố định: prefix(2) + "  "(2) + "  "(2) = 6
	spacing := 6
	nameW := innerW - idxW - countW - spacing
	if nameW < 10 {
		nameW = 10
	}

	// Header
	header := fmt.Sprintf("  %-*s  %-*s  %*s", idxW, "#", nameW, "Name", countW, "Tracks")

	b.WriteString(DimItemStyle.Render(header))
	b.WriteString("\n")

	// Vẽ line
	lineStyle := lipgloss.NewStyle().Foreground(themeCol("#7c7986"))
	b.WriteString(lineStyle.Render(strings.Repeat("─", innerW)))

	for i := p.plOffset; i < end; i++ {
		pl := filtered[i]

		prefix := "  "
		if i == p.plCursor {
			prefix = " "
		}

		idx := fmt.Sprintf("%-*d", idxW, i+1)
		name := runewidth.FillRight(truncate(pl.Name, nameW), nameW)
		countStr := fmt.Sprintf("%d songs", len(pl.Tracks))
		count := fmt.Sprintf("%*s", countW, countStr) // Ép lề phải

		line := prefix + idx + "  " + name + "  " + count

		b.WriteString("\n")
		if i == p.plCursor {
			b.WriteString(SelectedItemStyle.Width(innerW).Render(line))
		} else {
			b.WriteString(NormalItemStyle.Width(innerW).Render(line))
		}
	}

	return b.String()
}
func (p LeftPanel) renderPlaylistDetail(innerW int, filtered []storage.PlaylistTrack, playingID string) string {
	if len(filtered) == 0 {
		if len(p.activePlaylist.Tracks) == 0 {
			return DimItemStyle.Render(" This playlist is empty. Press ':' on a track to move it here.")
		}
		return DimItemStyle.Render(" No matching tracks.")
	}

	return renderSharedTrackList(
		innerW,
		filtered,
		p.plCursor,
		p.plOffset,
		p.visibleRows()+1,
		func(i int, t storage.PlaylistTrack) (string, string, string, int, bool, bool, int) {
			isPlaying := playingID != "" && (playingID == t.ID || playingID == "local:"+t.Path)

			origIdx := i + 1
			for j, pt := range p.activePlaylist.Tracks {
				if pt.ID == t.ID {
					origIdx = j + 1
					break
				}
			}

			return t.Title, t.Artist, t.Path, t.Duration, false, isPlaying, origIdx
		},
		false, nil, nil,
	)
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
