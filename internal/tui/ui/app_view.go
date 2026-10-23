package ui

import (
	"fmt"
	"strings"
	"time"

	tuilayout "som/internal/tui/layout"
	"som/internal/tui/player"
	"som/internal/tui/render"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (a *App) View() tea.View {
	if a.booting || a.width == 0 {
		v := tea.NewView(renderSplash(a.width, a.height, a.splashFrame))
		v.AltScreen = true
		return v
	}

	layout := a.uiLayout()
	mainView := a.renderMainContent(layout)
	sideView := renderSidebar(
		a.sidebarActive,
		layout.MainViewHeight,
		layout.MainViewHeight,
	)
	contentRow := lipgloss.JoinHorizontal(lipgloss.Top, sideView, mainView)

	view := a.renderBaseView(layout, contentRow)

	v := tea.NewView(a.renderOverlayView(view))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	a.setViewCursor(&v, layout.HeaderHeight+layout.SeparatorHeight)

	return v
}

func (a *App) renderMainContent(layout tuilayout.UILayout) string {
	contentH := layout.MainViewHeight
	mainW := layout.MainWidth
	frame := a.animeFrame

	playingID := ""
	if a.playback.NowPlay != nil {
		playingID = a.playback.NowPlay.ID
	}

	var mainView string
	switch a.sidebarActive {
	case SideSearch:
		mainView = a.left.ViewSearchContent(mainW, contentH, playingID)

	case SideDownloads:
		a.left.SetSize(layout.TracklistWidth, contentH)
		var alreadyInMove map[string]bool
		var selected map[string]bool
		selectMode := false
		if a.moveSession != nil &&
			a.moveSession.TargetPlIdx >= 0 &&
			a.moveSession.TargetPlIdx < len(a.left.playlists) {
			selectMode = true
			selected = a.moveSession.Selected
			alreadyInMove = map[string]bool{}
			for _, t := range a.left.playlists[a.moveSession.TargetPlIdx].Tracks {
				path := t.Path
				if path == "" {
					path = strings.TrimPrefix(t.ID, "local:")
				}
				alreadyInMove[path] = true
			}
		}

		tracklistView := a.left.ViewDownloadsContent(
			layout.TracklistWidth,
			contentH,
			selected,
			selectMode,
			alreadyInMove,
			playingID,
		)
		col3View := a.renderThirdColumn(layout.ThirdColumnWidth, contentH)
		mainView = lipgloss.JoinHorizontal(lipgloss.Top, tracklistView, col3View)

	case SideImport:
		a.importPanel.SetSize(mainW, contentH)
		mainView = a.importPanel.ViewImportContent(mainW, contentH)

	case SideQueue:
		mainView = a.left.ViewQueueContent(mainW, contentH, a.playback.Queue, playingID)

	case SidePlaylists:
		a.left.SetSize(layout.TracklistWidth, contentH)
		tracklistView := a.left.ViewPlaylistsContent(layout.TracklistWidth, contentH, playingID)
		col3View := a.renderThirdColumn(layout.ThirdColumnWidth, contentH)
		mainView = lipgloss.JoinHorizontal(lipgloss.Top, tracklistView, col3View)

	case SideLogs:
		mainView = renderLogsView(a.logOffset, mainW, contentH)

	default:
		mainView = a.renderLyricsView(mainW, contentH, frame)
	}

	return lipgloss.NewStyle().
		Width(layout.MainWidth).
		Height(layout.MainViewHeight).
		Render(mainView)
}

func (a *App) renderBaseView(layout tuilayout.UILayout, contentRow string) string {
	dashboard := renderDashboard(
		a.hideLogo,
		a.player.Volume(),
		a.activeSpeed,
		a.activePreset,
		a.playback.NowPlay,
	)
	somRow := a.renderSomRow(dashboard)

	borderStyle := lipgloss.NewStyle().Foreground(colorBorder)
	sepLeft := strings.Repeat("─", sidebarWidth)
	sepRight := ""
	if a.width > sidebarWidth+1 {
		sepRight = strings.Repeat("─", a.width-sidebarWidth-1)
	}
	sep := borderStyle.Render(sepLeft + "─" + sepRight)

	status := ""
	switch {
	case a.sidebarActive == SideSearch && a.left.loading:
		status = "  " + StatusMsgStyle.Render(a.left.spinner.View()+" Searching...")
	case a.sidebarActive == SideSearch && a.left.loadingStream:
		status = "  " + StatusMsgStyle.Render(a.left.spinner.View()+" Loading...")
	case a.sidebarActive == SideSearch && a.left.loadingDownload:
		status = "  " + StatusMsgStyle.Render(a.left.spinner.View()+" Downloading...")
	case a.statusMsg != "" && time.Since(a.statusAt) < 3*time.Second:
		status = "  " + a.statusMsg
	}
	if a.statusMsg != "" && time.Since(a.statusAt) < 3*time.Second {
		status = "  " + a.statusMsg
	}

	help := "  " +
		StyleHint("tab", "nav") + "  " +
		StyleHint("enter", "play") + "  " +
		StyleHint("]", "next") + "  " +
		StyleHint("[", "prev") + "  " +
		StyleHintActive("r", "random", a.playback.Random) + "  " +
		StyleHintActive(
			"space",
			"pause",
			a.player != nil && a.player.State() == player.Paused,
		) + "  " +
		StyleHintActive("/", "search", a.left.input.Focused()) + "  " +
		StyleHint("esc", "settings") + "  " +
		StyleHint("alt+q", "quit")
	progressBar := a.renderProgressBar(a.width)

	return render.BaseFrame(
		somRow,
		sep,
		contentRow,
		status,
		progressBar,
		help,
		a.hideLogo,
		a.hideHint,
	)
}

func (a *App) renderOverlayView(view string) string {
	if len(a.modals) > 0 {
		modal := a.modals[len(a.modals)-1]
		popup := modal.View()
		placed := lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, popup)
		var hints string
		switch modal.(type) {
		case *CmdMenuModal:
			hints = StyleHint("↑/↓", "navigate") + "  " + StyleHint("enter", "select") + "  " + StyleHint("esc", "close")
		case *PresetModal, *SpeedModal:
			hints = StyleHint("↑/↓", "navigate") + "  " + StyleHint("enter", "apply") + "  " + StyleHint("esc", "back")
		case *SortModal:
			hints = StyleHint("↑/↓", "navigate") + "  " + StyleHint("enter", "apply") + "  " + StyleHint("esc", "back")
		case *RenameModal, *RenamePlaylistModal:
			hints = StyleHint("enter", "rename") + "  " + StyleHint("esc", "cancel")
		case *MoveCreateModal:
			hints = StyleHint("enter", "create") + "  " + StyleHint("esc", "cancel")
		case *DeleteModal:
			hints = StyleHint("←/→", "select") + "  " + StyleHint("enter", "confirm") + "  " + StyleHint("esc", "back")
		case *MovePickModal:
			hints = StyleHint("↑/↓", "navigate") + "  " + StyleHint("enter", "select") + "  " + StyleHint("esc", "cancel")
		case *MoveConfirmModal:
			hints = StyleHint("←/→", "select") + "  " + StyleHint("enter", "confirm") + "  " + StyleHint("esc", "cancel")
		case *RemoveTrackModal:
			hints = StyleHint("↑/↓", "navigate") + "  " + StyleHint("enter", "remove") + "  " + StyleHint("esc", "back")
		}
		if hints != "" {
			lines := strings.Split(placed, "\n")
			lines[len(lines)-1] = lipgloss.NewStyle().
				Width(a.width).
				Render(" " + hints)
			return strings.Join(lines, "\n")
		}
		return placed
	}
	if a.left.showPlInput {
		popup := a.left.renderPlInputPopup()
		placed := lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, popup)
		lines := strings.Split(placed, "\n")
		lines[len(lines)-1] = lipgloss.NewStyle().
			Width(a.width).
			Render(" " + StyleHint("enter", "create") + "  " + StyleHint("esc", "cancel"))
		return strings.Join(lines, "\n")
	}
	if a.left.showDeletePopup {
		popup := a.left.renderDeletePopup()
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, popup)
	}
	if a.palette.Visible() {
		popup := a.palette.View()
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, popup)
	}
	return view
}

func (a *App) setViewCursor(v *tea.View, contentTop int) {
	if len(a.modals) > 0 {
		if modal, ok := a.modals[len(a.modals)-1].(*RenameModal); ok {
			if c := modal.input.Cursor(); c != nil {
				popup := modal.View()
				c.Position.X += (a.width-lipgloss.Width(popup))/2 + 3
				c.Position.Y += (a.height-lipgloss.Height(popup))/2 + 1
				v.Cursor = c
			}
			return
		}
	}

	if a.left.showPlInput && a.left.plInput.Focused() {
		if c := a.left.plInput.Cursor(); c != nil {
			popup := a.left.renderPlInputPopup()
			c.Position.X += (a.width-lipgloss.Width(popup))/2 + 2
			c.Position.Y += (a.height-lipgloss.Height(popup))/2 + 1
			v.Cursor = c
		}
		return
	}

	if !a.left.input.Focused() {
		return
	}

	if c := a.left.input.Cursor(); c != nil {
		c.Position.X += sidebarWidth + 3
		c.Position.Y = contentTop + 1
		v.Cursor = c
	}
}

// render lyrics hoặc import
func (a *App) renderSomRow(dashboard string) string {
	var hint string
	switch a.sidebarActive {
	case SideSearch:
		if a.left.showPlInput {
			return dashboard
		}
		hint = StyleHint("d", "download") + "  "
	case SideLyrics:
		if a.playback.NowPlay == nil || !a.right.loaded || len(a.right.lyrics.Synced) == 0 {
			return dashboard
		}
		hint = StyleHint("up/down", "select") + "  " + StyleHintActive("l", "lyric language", a.right.showLangPopup) + "  "
	case SideImport:
		if a.importPanel.importing {
			return dashboard
		}
		hint = StyleHint(".", "select") + "  " + StyleHint("i", "import") + "  " + StyleHint("r", "rescan") + "  "
	case SideDownloads:
		if a.moveSession != nil {
			plName := ""
			if a.moveSession.TargetPlIdx >= 0 && a.moveSession.TargetPlIdx < len(a.left.playlists) {
				plName = a.left.playlists[a.moveSession.TargetPlIdx].Name
			}
			hint = StyleHint(".", "select") + "  " + StyleHint("i", fmt.Sprintf("move to \"%s\" (%d)", plName, a.selectedMoveCount())) + "  " + StyleHint("+", "already in playlist") + "  " + StyleHint("esc", "cancel")
		} else {
			hint = StyleHint("ctrl+p", "pin/unpin") + "  " + StyleHint("\\", "visualizer") + "  " + StyleHintActive("l", "lyric", a.right.showLangPopup) + "  " + StyleHint(":", "Command") + "  "
		}
	case SidePlaylists:
		if a.left.showPlInput {
			return dashboard
		}
		hint = StyleHint(",", "new playlist") + "  " + StyleHintActive("l", "lyric", a.right.showLangPopup) + "  " + StyleHint("delete", "its deletes :)") + "  "

	default:
		return dashboard
	}

	lines := strings.Split(dashboard, "\n")
	if len(lines) == 0 {
		return dashboard
	}

	hintLine := len(lines) - 1

	logoW := lipgloss.Width(lines[hintLine])
	hintW := lipgloss.Width(hint)
	gap := a.width - logoW - hintW
	if gap < 1 {
		gap = 1
	}

	lines[hintLine] = lines[hintLine] + strings.Repeat(" ", gap) + hint
	return strings.Join(lines, "\n")
}
func (a *App) renderLyricsView(w, h, frame int) string {
	if a.playback.NowPlay == nil {
		return lipgloss.Place(
			w,
			h,
			lipgloss.Center,
			lipgloss.Center,
			DimItemStyle.Render("Play a track to see lyrics..."),
		)
	}

	lyricsBox := a.right.renderLyricsBox(colorWhite, frame)
	return lipgloss.NewStyle().Width(w).Render(lyricsBox)
}

func (a *App) renderProgressBar(w int) string {
	elapsedSec := 0
	totalSec := 0
	title := ""
	if a.playback.NowPlay != nil {
		elapsedSec = int(a.right.elapsed.Seconds())
		totalSec = a.playback.NowPlay.Duration
		title = a.playback.NowPlay.Title
	}

	return render.ProgressBar(
		w,
		elapsedSec,
		totalSec,
		title,
		FormatDuration(elapsedSec),
		render.ProgressBarStyles{
			Border:     lipgloss.NewStyle().Foreground(themeCol("#7c7986")),
			Title:      PanelTitleStyle.Foreground(themeCol("#7c7986")),
			Filled:     ProgressFilledStyle,
			Time:       ProgressTimeStyle,
			TimeFilled: ProgressTimeOnFillStyle,
			Controls:   ProgressDimStyle,
		},
	)
}

func (a *App) renderThirdColumn(w, h int) string {
	layout := tuilayout.NewThirdColumnLayout(w, h)

	visRaw := a.palette.RenderEQColumn(
		layout.SpectrumInnerWidth,
		layout.SpectrumInnerHeight,
	)
	visView := renderBox(
		layout.Width,
		"Spectrum",
		visRaw,
		themeCol("#7c7986"),
	)

	lyricContent := a.renderLyricAnim(
		layout.LyricsInnerWidth,
		layout.LyricsInnerHeight,
		a.animeFrame,
	)

	lyricTitle := "Lyrics"

	if a.right.showLangPopup {
		lyricTitle = "Select Language"
		lyricContent = a.right.renderLangPopupHeight(
			layout.LyricsInnerWidth,
			layout.LyricsInnerHeight,
		)
	}

	lyricBox := renderBox(
		layout.Width,
		lyricTitle,
		lyricContent,
		themeCol("#7c7986"),
	)

	duration := time.Since(a.sessionStart)
	hTime := int(duration.Hours())
	mTime := int(duration.Minutes()) % 60
	sTime := int(duration.Seconds()) % 60
	statsContent := fmt.Sprintf(
		"%02dh %02dm %02ds",
		hTime,
		mTime,
		sTime,
	)

	statsInner := lipgloss.NewStyle().
		Width(layout.Width - 4).
		Align(lipgloss.Center).
		Render(DimItemStyle.Render(statsContent))

	statsView := renderBox(
		layout.Width,
		"Session",
		statsInner,
		themeCol("#7c7986"),
	)

	return lipgloss.JoinVertical(
		lipgloss.Top,
		visView,
		lyricBox,
		statsView,
	)
}

func (a *App) renderLyricAnim(innerW, innerH, frame int) string {
	if a.right.noLyrics {
		return renderNoLyricsContent(innerW, innerH, frame)
	}

	progress := float64(time.Since(a.lyricAnimStart)) / float64(350*time.Millisecond)
	return render.LyricAnimation(
		a.prevLyric,
		a.currLyric,
		innerW,
		innerH,
		progress,
		render.LyricAnimationStyles{
			Highlight: LyricHighlightStyle,
			Dim:       DimItemStyle,
		},
		func(s string, width int) []string {
			return wordWrap(s, width)
		},
	)
}
