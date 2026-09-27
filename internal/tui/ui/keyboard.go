package ui

import (
	"som/internal/tui/player"

	tea "charm.land/bubbletea/v2"
)

// XỬ LÝ TOÀN BỘ PHÍM BẤM
func (a *App) handleKeys(msg tea.KeyPressMsg) tea.Cmd {
	var cmds []tea.Cmd

	if msg.String() == "ctrl+c" || msg.String() == "alt+q" {
		a.player.Stop()
		if a.avrcp != nil {
			a.avrcp.Close()
		}
		return tea.Quit
	}

	switch msg.String() {
	case "esc":
		if a.moveSession != nil {
			a.moveSession = nil
			a.setStatus(StatusMsgStyle.Render("> No changes to playlist"))
		} else if a.palette.Visible() {
			a.palette = a.palette.Close(a.sidebarActive)
		} else if !a.left.input.Focused() && !a.left.plInput.Focused() && !a.left.showDeletePopup && !a.left.showPlInput {
			if a.sidebarActive == SidePlaylists && a.left.activePlaylist != nil {
			} else {
				modal := NewEscMenuModal()
				a.modals = append(a.modals, modal)
				cmds = append(cmds, modal.Init())
			}
		}
	case ".":
		if a.moveSession != nil && a.sidebarActive == SideDownloads && !a.left.input.Focused() {
			a.toggleMoveSelection()
		}
	case "i":
		if a.moveSession != nil && a.sidebarActive == SideDownloads && !a.left.input.Focused() {
			if a.selectedMoveCount() == 0 {
				a.setStatus(StatusErrStyle.Render("X No tracks selected to move"))
			} else {
				plName := a.left.playlists[a.moveSession.TargetPlIdx].Name
				modal := NewMoveConfirmModal(plName, a.selectedMoveCount())
				a.modals = []Overlay{modal}
				cmds = append(cmds, modal.Init())
			}
		}
	case "1", "2", "3", "4", "5", "6", "7":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		targetTab := SidebarItem(msg.String()[0] - '1')
		cmds = append(cmds, a.switchSidebar(targetTab))
	case "\\":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		if a.palette.Visible() {
			a.palette = a.palette.Close(a.sidebarActive)
		} else {
			var cmd tea.Cmd
			a.palette, cmd = a.palette.Open()
			cmds = append(cmds, cmd)
		}
	case ":":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		modal := NewCmdMenuModal(a.cmdOptionList())
		a.modals = []Overlay{modal}
		cmds = append(cmds, modal.Init())
	case "tab":
		if a.left.input.Focused() {
			a.left.input.Blur()
		} else if a.left.plInput.Focused() {
			a.left.plInput.Blur()
			a.left.showPlInput = false
		} else {
			next := (a.sidebarActive + 1) % sideCount
			cmds = append(cmds, a.switchSidebar(next))
		}
	case "space":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		a.player.TogglePause()
		if a.avrcp != nil {
			if a.player.State() == player.Playing {
				a.avrcp.UpdatePlaybackStatus("Playing")
			} else if a.player.State() == player.Paused {
				a.avrcp.UpdatePlaybackStatus("Paused")
			}
		}
	case "right":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		a.player.SeekBy(5)
		cmds = append(cmds, func() tea.Msg { return PlaybackTickMsg{} })
	case "left":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		a.player.SeekBy(-5)
		cmds = append(cmds, func() tea.Msg { return PlaybackTickMsg{} })
	case "]", "}":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		cmds = append(cmds, func() tea.Msg { return PlayNextMsg{} })
	case "[", "{":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		cmds = append(cmds, func() tea.Msg { return PlayPrevMsg{} })
	case "r", "R":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		if a.sidebarActive == SideImport {
			break
		}
		cmds = append(cmds, func() tea.Msg { return ToggleRandomMsg{} })
	case "up":
		if a.sidebarActive == SideLogs {
			if a.logOffset < LogBuf.Len()-1 {
				a.logOffset++
			}
		}
	case "down":
		if a.sidebarActive == SideLogs {
			if a.logOffset > 0 {
				a.logOffset--
			}
		}
	case "+", "=":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		v := a.player.Volume() + 0.05
		a.player.SetVolume(v)
	case "-", "_":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		v := a.player.Volume() - 0.05
		a.player.SetVolume(v)

	case "?":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		modal := NewHelpModal(a.width, a.height)
		a.modals = append(a.modals, modal)
		cmds = append(cmds, modal.Init())
	}
	return tea.Batch(cmds...)
}
