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

	var helpParts []string
	for _, hint := range a.globalHelpHints() {
		helpParts = append(helpParts, hint.text)
	}
	help := "  " + strings.Join(helpParts, "  ")
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

type globalHelpHint struct {
	text string
	key  rune
	mod  tea.KeyMod
}

func (a *App) globalHelpHints() []globalHelpHint {
	return []globalHelpHint{
		{text: StyleHint("tab", "nav")},
		{text: StyleHint("enter", "play")},
		{text: StyleHint("]", "next"), key: ']'},
		{text: StyleHint("[", "prev"), key: '['},
		{text: StyleHintActive("r", "random", a.playback.Random), key: 'r'},
		{text: StyleHintActive("space", "pause", a.player != nil && a.player.State() == player.Paused), key: tea.KeySpace},
		{text: StyleHintActive("/", "search", a.left.input.Focused()), key: '/'},
		{text: StyleHint("esc", "settings"), key: tea.KeyEsc},
		{text: StyleHint("alt+q", "quit")},
	}
}

func (a *App) somRowHints() []globalHelpHint {
	switch a.sidebarActive {
	case SideSearch:
		if a.left.showPlInput {
			return nil
		}
		return []globalHelpHint{{text: StyleHint("d", "download")}}
	case SideLyrics:
		if a.playback.NowPlay == nil || !a.right.loaded || len(a.right.lyrics.Synced) == 0 {
			return nil
		}
		return []globalHelpHint{
			{text: StyleHint("up/down", "select")},
			{text: StyleHintActive("l", "lyric language", a.right.showLangPopup), key: 'l'},
		}
	case SideImport:
		if a.importPanel.importing {
			return nil
		}
		return []globalHelpHint{
			{text: StyleHint(".", "select")},
			{text: StyleHint("i", "import")},
			{text: StyleHint("r", "rescan")},
		}
	case SideDownloads:
		if a.moveSession != nil {
			plName := ""
			if a.moveSession.TargetPlIdx >= 0 && a.moveSession.TargetPlIdx < len(a.left.playlists) {
				plName = a.left.playlists[a.moveSession.TargetPlIdx].Name
			}
			return []globalHelpHint{
				{text: StyleHint(".", "select")},
				{text: StyleHint("i", fmt.Sprintf("move to \"%s\" (%d)", plName, a.selectedMoveCount()))},
				{text: StyleHint("+", "already in playlist")},
				{text: StyleHint("esc", "cancel")},
			}
		}
		return []globalHelpHint{
			{text: StyleHint("ctrl+p", "pin/unpin"), key: 'p', mod: tea.ModCtrl},
			{text: StyleHint("\\", "visualizer"), key: '\\'},
			{text: StyleHintActive("l", "lyric", a.right.showLangPopup), key: 'l'},
			{text: StyleHint(":", "Command"), key: ':'},
		}
	case SideQueue:
		return []globalHelpHint{{text: StyleHint("delete", "remove from queue")}}
	case SidePlaylists:
		if a.left.showPlInput {
			return nil
		}
		return []globalHelpHint{
			{text: StyleHint(",", "new playlist")},
			{text: StyleHintActive("l", "lyric", a.right.showLangPopup), key: 'l'},
			{text: StyleHint("delete", "its deletes :)")},
		}
	default:
		return nil
	}
}

func renderHintItems(items []globalHelpHint) string {
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = item.text
	}
	return strings.Join(parts, "  ") + " "
}

func (a *App) renderOverlayView(view string) string {
	if len(a.modals) > 0 {
		modal := a.modals[len(a.modals)-1]
		popup := modal.View()
		placed := lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, popup)
		var hints string
		switch modal.(type) {
		case *EscMenuModal:
			hints = StyleHint("↑/↓", "navigate") + "  " + StyleHint("enter", "open") + "  " + StyleHint("esc", "close")
			if a.mouseEnabled {
				hints += "  " + StyleHint("click", "select") + "  " + StyleHint("double-click", "open")
			}
		case *SettingsModal:
			hints = StyleHint("↑/↓", "select") + "  " + StyleHint("←/→", "toggle") + "  " + StyleHint("esc", "close")
			if a.mouseEnabled {
				hints += "  " + StyleHint("click", "select") + "  " + StyleHint("double-click", "toggle")
			}
		case *CmdMenuModal:
			hints = StyleHint("↑/↓", "navigate") + "  " + StyleHint("enter", "select") + "  " + StyleHint("esc", "close")
			if a.mouseEnabled {
				hints += "  " + StyleHint("click", "select") + "  " + StyleHint("double-click", "open")
			}
		case *PresetModal, *SpeedModal, *SortModal:
			hints = StyleHint("↑/↓", "navigate") + "  " + StyleHint("enter", "apply") + "  " + StyleHint("esc", "back")
			if a.mouseEnabled {
				hints += "  " + StyleHint("click", "select") + "  " + StyleHint("double-click", "apply")
			}
		case *MovePickModal:
			hints = StyleHint("↑/↓", "navigate") + "  " + StyleHint("enter", "select") + "  " + StyleHint("esc", "cancel")
			if a.mouseEnabled {
				hints += "  " + StyleHint("click", "select") + "  " + StyleHint("double-click", "open")
			}
		case *RemoveTrackModal:
			hints = StyleHint("↑/↓", "navigate") + "  " + StyleHint("enter", "remove") + "  " + StyleHint("esc", "back")
			if a.mouseEnabled {
				hints += "  " + StyleHint("click", "select") + "  " + StyleHint("double-click", "remove")
			}
		case *RenameModal, *RenamePlaylistModal:
			hints = StyleHint("enter", "rename") + "  " + StyleHint("esc", "cancel")
		case *MoveCreateModal:
			hints = StyleHint("enter", "create") + "  " + StyleHint("esc", "cancel")
		case *DeleteModal:
			hints = StyleHint("←/→", "select") + "  " + StyleHint("enter", "confirm") + "  " + StyleHint("esc", "back")
		case *MoveConfirmModal:
			hints = StyleHint("←/→", "select") + "  " + StyleHint("enter", "confirm") + "  " + StyleHint("esc", "cancel")
		}
		if a.mouseEnabled {
			switch modal.(type) {
			case *DeleteModal, *MoveConfirmModal:
				hints += "  " + StyleHint("click", "select") + "  " + StyleHint("double-click", "confirm")
			}
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
	if a.playingPlaylistName != "" && !a.hideLogo {
		lines := strings.Split(dashboard, "\n")
		if len(lines) > 0 {
			leftWidth := lipgloss.Width(lines[0])
			nameWidth := a.width - leftWidth - 1 - lipgloss.Width(StyleHint("Playlist", ""))
			if nameWidth > 0 {
				playlist := StyleHint("Playlist", truncate(a.playingPlaylistName, nameWidth)+" ")
				gap := a.width - leftWidth - lipgloss.Width(playlist)
				if gap < 1 {
					gap = 1
				}
				lines[0] += strings.Repeat(" ", gap) + playlist
			}
			dashboard = strings.Join(lines, "\n")
		}
	}

	items := a.somRowHints()
	if len(items) == 0 {
		return dashboard
	}
	hint := renderHintItems(items)

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
