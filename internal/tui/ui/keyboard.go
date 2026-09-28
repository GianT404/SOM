package ui

import (
	"som/internal/tui/player"

	tea "charm.land/bubbletea/v2"
)

// XỬ LÝ TOÀN BỘ PHÍM BẤM
func (a *App) textInputFocused() bool {
	return a.left.input.Focused() || a.left.plInput.Focused()
}

func (a *App) handlePlaybackKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	if a.textInputFocused() {
		return nil, false
	}

	switch msg.String() {
	default:
		if cmd, handled := a.handlePlaybackKey(msg); handled && cmd != nil {
			cmds = append(cmds, cmd)
		}
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
