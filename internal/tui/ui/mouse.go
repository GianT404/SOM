package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

const mouseSettingKey = "mouse"

func mouseOption() settingOpt {
	return settingOpt{
		title: "Mouse support",
		desc:  "Click a sidebar tab to switch, click a row to select it (double-click to play), and scroll the wheel to move through lists / logs.",
		on:    func(a *App) bool { return a.mouseEnabled },
		apply: func(a *App, on bool) {
			a.mouseEnabled = on
			if a.left.plStore != nil {
				v := "0"
				if on {
					v = "1"
				}
				a.left.plStore.SetSetting(mouseSettingKey, v)
			}
		},
	}
}

func loadMouseSetting(a *App) {
	// Mặc định tắt
	a.mouseEnabled = false
	if a.left.plStore == nil {
		return
	}
	if v := a.left.plStore.GetSetting(mouseSettingKey); v == "1" {
		a.mouseEnabled = true
	}
}

func (a *App) listRowOrigin() (int, bool) {
	ct := a.somRowHeight() + 1
	switch a.sidebarActive {
	case SideSearch:
		if a.left.input.Focused() && len(a.left.suggestions) > 0 {
			return 0, false
		}
		if a.left.loadingStream || a.left.loadingDownload || a.left.errMsg != "" {
			return 0, false
		}
		if len(a.left.tracks) == 0 {
			return 0, false
		}
		return ct + 4, true
	case SideDownloads:
		if a.left.input.Focused() && len(a.left.suggestions) > 0 {
			return 0, false
		}
		if a.left.loading || a.left.errMsg != "" {
			return 0, false
		}
		if len(a.left.getFilteredLocals()) == 0 {
			return 0, false
		}
		return ct + 5, true
	case SideQueue:
		if len(a.trackQueue) == 0 {
			return 0, false
		}
		return ct + 2, true
	case SidePlaylists:
		if a.left.activePlaylist != nil {
			if len(a.left.activePlaylist.Tracks) == 0 {
				return 0, false
			}
			return ct + 3, true
		}
		if len(a.left.playlists) == 0 {
			return 0, false
		}
		return ct + 2, true
	default:
		return 0, false
	}
}

func (a *App) moveListCursorTo(idx int) bool {
	if idx < 0 {
		return false
	}
	vis := a.left.visibleRows()
	clamp := func(offset *int) {
		if idx < *offset {
			*offset = idx
		}
		if idx >= *offset+vis {
			*offset = idx - vis + 1
		}
		if *offset < 0 {
			*offset = 0
		}
	}

	switch a.sidebarActive {
	case SideSearch:
		if idx >= len(a.left.tracks) {
			return false
		}
		a.left.searchCursor = idx
		clamp(&a.left.searchOffset)
		return true
	case SideDownloads:
		n := len(a.left.getFilteredLocals())
		if idx >= n {
			return false
		}
		a.left.dlCursor = idx
		clamp(&a.left.dlOffset)
		return true
	case SideQueue:
		if idx >= len(a.trackQueue) {
			return false
		}
		a.left.qCursor = idx
		clamp(&a.left.qOffset)
		return true
	case SidePlaylists:
		if a.left.activePlaylist != nil {
			if idx >= len(a.left.activePlaylist.Tracks) {
				return false
			}
			a.left.plCursor = idx
			clamp(&a.left.plOffset)
			return true
		}
		if idx >= len(a.left.playlists) {
			return false
		}
		a.left.plCursor = idx
		clamp(&a.left.plOffset)
		return true
	}
	return false
}

func (a *App) stepListCursor(delta int) {
	var cur int
	switch a.sidebarActive {
	case SideSearch:
		if a.left.input.Focused() {
			return
		}
		if len(a.left.tracks) == 0 {
			return
		}
		cur = a.left.searchCursor
	case SideDownloads:
		if a.left.input.Focused() {
			return
		}
		if len(a.left.getFilteredLocals()) == 0 {
			return
		}
		cur = a.left.dlCursor
	case SideQueue:
		cur = a.left.qCursor
		if len(a.trackQueue) == 0 {
			return
		}
	case SidePlaylists:
		cur = a.left.plCursor
		if a.left.activePlaylist != nil {
			if len(a.left.activePlaylist.Tracks) == 0 {
				return
			}
		} else if len(a.left.playlists) == 0 {
			return
		}
	default:
		return
	}
	a.moveListCursorTo(cur + delta)
}

func (a *App) tryFocusSearchInput(m tea.MouseClickMsg) bool {
	ct := a.somRowHeight() + 1
	if m.Y != ct+1 {
		return false
	}
	switch a.sidebarActive {
	case SideSearch, SideDownloads:
		if m.X < sidebarWidth {
			return false
		}
		if !a.left.input.Focused() {
			a.left.input.Focus()
		}
		return true
	}
	return false
}

func (a *App) statusRowVisible() bool {
	return a.statusMsg != "" && time.Since(a.statusAt) < 5*time.Second
}

func (a *App) progressBarTop() int {
	statusH := 0
	if a.statusRowVisible() {
		statusH = 1
	}
	return a.somRowHeight() + 1 + a.mainContentHeight() + statusH
}

func (a *App) seekFromProgressClick(m tea.MouseClickMsg) bool {
	if a.player == nil || a.nowPlay == nil || a.nowPlay.Duration <= 0 {
		return false
	}
	if m.Button != tea.MouseLeft {
		return false
	}
	top := a.progressBarTop()
	if m.Y < top || m.Y > top+2 {
		return false
	}
	innerW := a.width - 4
	if m.X < 2 || m.X > 2+innerW-1 {
		return false
	}
	frac := float64(m.X-1) / float64(innerW)
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	target := frac * float64(a.nowPlay.Duration)
	a.player.SeekBy(target - a.player.Position().Seconds())
	a.right.TickAt()
	return true
}

func (a *App) handleMouseWheel(up bool) {
	if !a.mouseEnabled {
		return
	}
	if a.sidebarActive == SideLogs {
		if up {
			if a.logOffset < LogBuf.Len()-1 {
				a.logOffset++
			}
		} else if a.logOffset > 0 {
			a.logOffset--
		}
		return
	}
	if up {
		a.stepListCursor(-1)
	} else {
		a.stepListCursor(1)
	}
}

func (a *App) handleMouseClick(m tea.MouseClickMsg) tea.Cmd {
	if !a.mouseEnabled {
		return nil
	}
	if a.showSettings || a.showHelpPopup || a.showCmdPopup || a.left.showDeletePopup || a.left.showPlInput || a.palette.Visible() {
		return nil
	}

	// Click lên thanh progress → tua.
	if a.seekFromProgressClick(m) {
		return nil
	}

	contentTop := a.somRowHeight() + 1

	// Sidebar: đổi tab.
	if m.X < sidebarWidth {
		tabRow := m.Y - contentTop
		if tabRow >= 0 && SidebarItem(tabRow) < sideCount {
			return a.switchSidebar(SidebarItem(tabRow))
		}
		return nil
	}

	if a.tryFocusSearchInput(m) {
		return nil
	}

	origin, ok := a.listRowOrigin()
	if !ok || m.Y < origin {
		return nil
	}
	local := m.Y - origin

	cur := -1
	switch a.sidebarActive {
	case SideSearch:
		cur = a.left.searchCursor
	case SideDownloads:
		cur = a.left.dlCursor
	case SideQueue:
		cur = a.left.qCursor
	case SidePlaylists:
		cur = a.left.plCursor
	}
	if cur < 0 {
		cur = 0
	}
	vis := a.left.visibleRows()
	if local >= vis {
		return nil
	}
	idx := cur
	switch a.sidebarActive {
	case SideSearch:
		idx = a.left.searchOffset + local
	case SideDownloads:
		idx = a.left.dlOffset + local
	case SideQueue:
		idx = a.left.qOffset + local
	case SidePlaylists:
		idx = a.left.plOffset + local
	default:
		return nil
	}
	if !a.moveListCursorTo(idx) {
		return nil
	}
	if a.sidebarActive == SideSearch || a.sidebarActive == SideDownloads {
		if a.left.input.Focused() {
			a.left.input.Blur()
		}
		a.left.suggestions = nil
		a.left.suggestFocus = false
		a.left.suggestCursor = 0
		a.left.suggestOffset = 0
	}

	// Double-click cùng dòng → như phím Enter (chọn/play).
	now := time.Now()
	dl := now.Sub(a.mouseLastClickAt)
	a.mouseLastClickAt = now
	if dl < 350*time.Millisecond && a.mouseLastClickY == m.Y && a.mouseLastClickTab == a.sidebarActive {
		_, cmd := a.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		return cmd
	}
	a.mouseLastClickY = m.Y
	a.mouseLastClickTab = a.sidebarActive
	return nil
}
