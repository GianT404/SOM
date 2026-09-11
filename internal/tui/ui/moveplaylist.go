package ui

import (
	"fmt"
	"strings"

	"som/internal/storage"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-runewidth"
)

func (a *App) startMoveToPlaylist() tea.Cmd {
	cmd := a.switchSidebar(SideDownloads)
	a.left.input.Blur()
	a.moveSelectActive = true
	a.moveSelected = map[string]bool{}
	return cmd
}

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

func (a *App) selectedMoveCount() int {
	n := 0
	for _, v := range a.moveSelected {
		if v {
			n++
		}
	}
	return n
}

func (a *App) finishMoveSelection() {
	if a.selectedMoveCount() == 0 {
		a.setStatus(StatusErrStyle.Render("X No tracks selected (press . to select)"))
		return
	}
	a.moveSelectActive = false

	if len(a.left.playlists) == 0 {
		a.moveCreateActive = true
		a.moveCreateInput.SetValue("")
		a.moveCreateInput.Focus()
		a.moveCreateInput.CursorEnd()
		return
	}

	a.movePickActive = true
	a.cmdCursor = 0
}

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
			a.moveConfirmActive = true
			a.moveConfirmPlIdx = len(a.left.playlists) - 1
			a.cmdCursor = 0
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
				a.moveConfirmActive = true
				a.moveConfirmPlIdx = a.cmdCursor
				a.cmdCursor = 0
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

// applyMoveToPlaylist them cac track da chon vao playlist duoc xac nhan.
func (a *App) applyMoveToPlaylist() {
	if a.moveConfirmPlIdx < 0 || a.moveConfirmPlIdx >= len(a.left.playlists) {
		return
	}
	if a.left.plStore == nil {
		a.setStatus(StatusErrStyle.Render("X Database not initialized"))
		return
	}
	pl := &a.left.playlists[a.moveConfirmPlIdx]

	existing := make(map[string]bool, len(pl.Tracks))
	for _, t := range pl.Tracks {
		existing[t.ID] = true
	}

	moved := 0
	for _, lf := range a.left.locals {
		if !a.moveSelected[lf.Path] {
			continue
		}
		id := "local:" + lf.Path
		if existing[id] {
			continue
		}
		track := storage.PlaylistTrack{
			ID:       id,
			Title:    lf.Name,
			Artist:   lf.Artist,
			Duration: lf.Duration,
			IsLocal:  true,
		}
		if err := a.left.plStore.AddTrackToPlaylist(pl.ID, track); err != nil {
			continue
		}
		pl.Tracks = append(pl.Tracks, track)
		existing[id] = true
		moved++
	}

	a.setStatus(StatusOKStyle.Render(fmt.Sprintf("> Moved %d track to \"%s\"", moved, pl.Name)))
}

func (a *App) renderMoveCreatePopup() string {
	var b strings.Builder
	b.WriteString("\n ")
	b.WriteString(NormalItemStyle.Render(fmt.Sprintf("No playlist %d track:", a.selectedMoveCount())))
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
	b.WriteString(NormalItemStyle.Render(fmt.Sprintf("Move %d track to:", a.selectedMoveCount())))
	b.WriteString("\n\n ")
	for i, pl := range a.left.playlists {
		line := "  " + pl.Name
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
	if a.moveConfirmPlIdx >= 0 && a.moveConfirmPlIdx < len(a.left.playlists) {
		name = a.left.playlists[a.moveConfirmPlIdx].Name
	}
	b.WriteString("\n ")
	b.WriteString(DimItemStyle.Render(fmt.Sprintf("Move %d track in \"%s\"?", a.selectedMoveCount(), name)))
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
