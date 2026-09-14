package ui

import (
	"fmt"
	"strings"

	"som/internal/storage"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-runewidth"
)

// startMoveToPlaylist mo popup chon playlist dich ngay (hoac popup tao moi
// neu chua co playlist nao), truoc khi vao buoc chon track o tab Download.
func (a *App) startMoveToPlaylist() tea.Cmd {
	cmd := a.switchSidebar(SideDownloads)
	a.left.input.Blur()
	if len(a.left.playlists) == 0 {
		a.moveCreateActive = true
		a.moveCreateInput.SetValue("")
		a.moveCreateInput.Focus()
		a.moveCreateInput.CursorEnd()
	} else {
		a.movePickActive = true
		a.cmdCursor = 0
	}
	return cmd
}

// toggleMoveSelection bat/tat chon track dang highlight trong tab Download.
func (a *App) toggleMoveSelection() {
	locals := a.left.getFilteredLocals()
	if a.left.dlCursor < 0 || a.left.dlCursor >= len(locals) {
		return
	}
	if a.moveSelected == nil {
		a.moveSelected = map[string]bool{}
	}
	path := locals[a.left.dlCursor].Path
	a.moveSelected[path] = !a.moveSelected[path]
}

// selectedMoveCount dem so track dang duoc chon de move.
func (a *App) selectedMoveCount() int {
	n := 0
	for _, v := range a.moveSelected {
		if v {
			n++
		}
	}
	return n
}

// finishMoveSelection duoc goi khi bam 'i': playlist dich da chon tu truoc
// (moveTargetPlIdx), chi can validate selection roi qua thang buoc confirm.
func (a *App) finishMoveSelection() {
	if a.selectedMoveCount() == 0 {
		a.setStatus(StatusErrStyle.Render("X Chua chon track nao (phim . de chon)"))
		return
	}
	a.moveSelectActive = false
	a.moveConfirmActive = true
	a.cmdCursor = 0
}

// updateMovePopup xu ly phim bam cho cac popup cua luong move: tao playlist
// moi, chon playlist dich, va xac nhan cuoi cung.
func (a *App) updateMovePopup(k tea.KeyMsg) tea.Cmd {
	if a.moveCreateActive {
		switch k.String() {
		case "enter":
			name := strings.TrimSpace(a.moveCreateInput.Value())
			if name == "" {
				return nil
			}
			if a.left.plStore == nil {
				a.setStatus(StatusErrStyle.Render("X Database not initialized"))
				return nil
			}
			pl, err := a.left.plStore.CreatePlaylist(name)
			if err != nil {
				a.setStatus(StatusErrStyle.Render("X " + err.Error()))
				return nil
			}
			a.left.playlists = append(a.left.playlists, pl)
			a.moveCreateActive = false
			a.moveCreateInput.Blur()
			a.moveCreateInput.SetValue("")
			a.moveTargetPlIdx = len(a.left.playlists) - 1
			a.moveSelectActive = true
			a.moveSelected = map[string]bool{}
			return nil
		case "esc":
			a.moveCreateActive = false
			a.moveCreateInput.Blur()
			a.moveCreateInput.SetValue("")
			a.moveSelected = nil
			return nil
		}
		var cmd tea.Cmd
		a.moveCreateInput, cmd = a.moveCreateInput.Update(k)
		return cmd
	}

	if a.movePickActive {
		switch k.String() {
		case "up", "k":
			if a.cmdCursor > 0 {
				a.cmdCursor--
			} else {
				a.cmdCursor = len(a.left.playlists) - 1
			}
		case "down", "j":
			if a.cmdCursor < len(a.left.playlists)-1 {
				a.cmdCursor++
			} else {
				a.cmdCursor = 0
			}
		case "enter":
			if a.cmdCursor < len(a.left.playlists) {
				a.movePickActive = false
				a.moveTargetPlIdx = a.cmdCursor
				a.moveSelectActive = true
				a.moveSelected = map[string]bool{}
			}
		case "esc", ":":
			a.movePickActive = false
			a.moveSelected = nil
		}
		return nil
	}

	if a.moveConfirmActive {
		switch k.String() {
		case "left", "h", "right", "l", "up", "down", "k", "j", "tab":
			a.cmdCursor = 1 - a.cmdCursor
		case "enter":
			if a.cmdCursor == 1 {
				a.applyMoveToPlaylist()
			}
			a.moveConfirmActive = false
			a.moveSelected = nil
		case "esc", ":":
			a.moveConfirmActive = false
			a.moveSelected = nil
		}
		return nil
	}
	return nil
}

func (a *App) applyMoveToPlaylist() {
	if a.moveTargetPlIdx < 0 || a.moveTargetPlIdx >= len(a.left.playlists) {
		return
	}
	if a.left.plStore == nil {
		a.setStatus(StatusErrStyle.Render("X Database not initialized"))
		return
	}

	pl := &a.left.playlists[a.moveTargetPlIdx]
	existing := make(map[string]bool, len(pl.Tracks))
	for _, t := range pl.Tracks {
		existing[t.ID] = true
	}

	added := 0
	removed := 0

	for _, lf := range a.left.locals {
		if !a.moveSelected[lf.Path] {
			continue
		}

		id := "local:" + lf.Path

		if existing[id] {
			if err := a.left.plStore.RemoveTrackFromPlaylist(pl.ID, id); err == nil {
				var newTracks []storage.PlaylistTrack
				for _, t := range pl.Tracks {
					if t.ID != id {
						newTracks = append(newTracks, t)
					}
				}
				pl.Tracks = newTracks
				existing[id] = false
				removed++
			}
		} else {
			track := storage.PlaylistTrack{
				ID:       id,
				Title:    lf.Name,
				Artist:   lf.Artist,
				Duration: lf.Duration,
				IsLocal:  true,
			}
			if err := a.left.plStore.AddTrackToPlaylist(pl.ID, track); err == nil {
				pl.Tracks = append(pl.Tracks, track)
				existing[id] = true
				added++
			}
		}
	}

	a.setStatus(StatusOKStyle.Render(fmt.Sprintf("> Changed: %d added, %d removed in \"%s\"", added, removed, pl.Name)))
}

func (a *App) renderMoveCreatePopup() string {
	var b strings.Builder
	b.WriteString("\n ")
	b.WriteString(NormalItemStyle.Render(fmt.Sprintf("No playlist yet. Enter name for %d track:", a.selectedMoveCount())))
	b.WriteString("\n\n  ")
	b.WriteString(a.moveCreateInput.View())
	b.WriteString("\n\n ")
	b.WriteString(DimItemStyle.Render(" (enter: create  | esc: cancel)"))
	w := a.moveCreateInput.Width() + 8
	if w < 48 {
		w = 48
	}
	if a.width > 0 && w > a.width-2 {
		w = a.width - 2
	}
	return renderBox(w, "Create New Playlist", b.String(), themeCol("#e8593c"))
}

func (a *App) renderMovePickPopup() string {
	var b strings.Builder
	b.WriteString("\n ")
	b.WriteString(NormalItemStyle.Render("Select target playlist:"))
	b.WriteString("\n\n ")
	for i, pl := range a.left.playlists {
		line := fmt.Sprintf("  %s (%d)", pl.Name, len(pl.Tracks))
		if i == a.cmdCursor {
			pad := 51 - runewidth.StringWidth(line)
			if pad < 0 {
				pad = 0
			}
			b.WriteString(SelectedItemStyle.Render(line + strings.Repeat(" ", pad)))
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
		b.WriteString("\n ")
	}
	b.WriteString("\n ")
	b.WriteString(DimItemStyle.Render(" (enter: select  | esc: cancel)"))
	return renderBox(56, "Select Playlist", b.String(), themeCol("#e8593c"))
}

func (a *App) renderMoveConfirmPopup() string {
	var b strings.Builder
	name := ""
	if a.moveTargetPlIdx >= 0 && a.moveTargetPlIdx < len(a.left.playlists) {
		name = a.left.playlists[a.moveTargetPlIdx].Name
	}
	b.WriteString("\n ")
	b.WriteString(DimItemStyle.Render(fmt.Sprintf("Move %d track to \"%s\"?", a.selectedMoveCount(), name)))
	b.WriteString("\n\n ")

	cancelStyle := NormalItemStyle
	confirmStyle := NormalItemStyle
	if a.cmdCursor == 0 {
		cancelStyle = SelectedItemStyle
	} else {
		confirmStyle = SelectedItemStyle
	}
	b.WriteString(fmt.Sprintf("%s     %s", cancelStyle.Render("[ Cancel ]"), confirmStyle.Render("[ Confirm ]")))
	b.WriteString("\n\n")
	b.WriteString(DimItemStyle.Render(" (enter: confirm  | esc: cancel)"))
	return renderBox(60, "Confirm Move", b.String(), themeCol("#e8593c"))
}
