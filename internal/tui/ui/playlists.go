package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (p LeftPanel) ViewPlaylistsContent(w, h int) string {
	innerW := w - 4
	inputFocused := p.input.Focused()
	searchBorder := lipgloss.Color("#7c7986")
	contentBorder := lipgloss.Color("#7c7986")

	if inputFocused {
		searchBorder = lipgloss.Color("#e8593c")
	} else {
		contentBorder = lipgloss.Color("#e8593c")
	}

	var searchContent strings.Builder
	inputRow := " " + p.input.View()
	if p.loading {
		inputRow += " " + p.spinner.View()
	}
	searchContent.WriteString(lipgloss.NewStyle().Width(innerW).Render(inputRow))

	if p.showAddPopup {
		searchContent.WriteString(p.renderAddPopup())
	} else if p.showPlInput {
		searchContent.WriteString(p.plInput.View())
	}

	searchBox := renderBox(w, "Playlists", searchContent.String(), searchBorder)

	// --- Playlist List / Detail Box ---
	var listContent string
	if p.activePlaylist != nil {
		listContent = p.renderPlaylistDetail(innerW)
	} else {
		listContent = p.renderPlaylistList(innerW)
	}

	listBox := renderBox(w, "Playlist List", listContent, contentBorder)

	return searchBox + "\n" + listBox
}

func (p LeftPanel) renderPlaylistList(innerW int) string {
	if len(p.playlists) == 0 {
		return DimItemStyle.Render(" No playlists available. Press 'c' to create a new playlist.") + "\n"
	}

	var b strings.Builder
	vis := p.visibleRows()
	end := p.plOffset + vis
	if end > len(p.playlists) {
		end = len(p.playlists)
	}

	for i := p.plOffset; i < end; i++ {
		pl := p.playlists[i]
		mark := "  "
		if i == p.plCursor {
			mark = ""
		}

		name := truncate(pl.Name, innerW-10)
		count := fmt.Sprintf("%d songs", len(pl.Tracks))
		line := fmt.Sprintf("%s%s %s", mark, name, count)

		pad := innerW - lipgloss.Width(line)
		if pad < 0 {
			pad = 0
		}

		if i == p.plCursor {
			b.WriteString(SelectedItemStyle.Width(innerW).Render(line + strings.Repeat(" ", pad)))
		} else {
			b.WriteString(NormalItemStyle.Width(innerW).Render(line + strings.Repeat(" ", pad)))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (p LeftPanel) renderPlaylistDetail(innerW int) string {
	if p.activePlaylist == nil || len(p.activePlaylist.Tracks) == 0 {
		return DimItemStyle.Render("This playlist is empty. Press 'a' in the Search/Downloads tab to add songs.")
	}

	var b strings.Builder
	vis := p.visibleRows()
	tracks := p.activePlaylist.Tracks
	end := p.plOffset + vis
	if end > len(tracks) {
		end = len(tracks)
	}

	idxW := 3
	if len(tracks) >= 1000 {
		idxW = 4
	}
	artistW := 27
	titleW := innerW - idxW - artistW - 6
	if titleW < 10 {
		titleW = 10
		artistW = innerW - idxW - titleW - 6
		if artistW < 0 {
			artistW = 0
		}
	}
	header := fmt.Sprintf("  %*s  %-*s  %s", idxW, "#", titleW, "Title", "Artist")
	b.WriteString(DimItemStyle.Width(innerW).Render(header))
	b.WriteString("\n")

	for i := p.plOffset; i < end; i++ {
		t := tracks[i]
		mark := "  "
		if i == p.plCursor {
			mark = " "
		}

		idx := fmt.Sprintf("%*d", idxW, i+1)
		title := truncate(t.Title, titleW)
		artist := truncate(t.Artist, artistW)
		line := mark + idx + "  " + fmt.Sprintf("%-*s", titleW, title) + "  " + fmt.Sprintf("%-*s", artistW, artist)

		if i == p.plCursor {
			b.WriteString(LocalFileSelectedStyle.Width(innerW).Render(line))
		} else {
			b.WriteString(LocalFileStyle.Width(innerW).Render(line))
		}
		b.WriteString("\n")
	}

	if len(tracks) > vis {
		b.WriteString(DimItemStyle.Render(fmt.Sprintf(" %d/%d", p.plCursor+1, len(tracks))))
		b.WriteString("\n")
	}

	return b.String()
}

func (p LeftPanel) renderAddPopup() string {
	var b strings.Builder
	b.WriteString("\n\n")

	if len(p.playlists) == 0 {
		b.WriteString(DimItemStyle.Render(" (No playlists available. Create one first.)"))
		b.WriteString("\n")
	} else {
		for i, pl := range p.playlists {
			marker := "  "
			if i == p.popupCursor {
				marker = "▸ "
			}
			line := marker + pl.Name
			if i == p.popupCursor {
				b.WriteString(SelectedItemStyle.Render(line))
			} else {
				b.WriteString(NormalItemStyle.Render(line))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(DimItemStyle.Render(" (enter: select  | esc: cancel)"))
	return renderBox(40, "Add to Playlist", b.String(), lipgloss.Color("#e8593c"))
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

	return renderBox(45, "Confirm", b.String(), lipgloss.Color("#E24B4A"))
}
