package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	tuilayout "som/internal/tui/layout"
	"som/internal/tui/render"
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
		a.sidebarAnim,
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
	frame := a.splashFrame

	inputNotFocused := !a.left.input.Focused()
	playingID := ""
	if a.playback.NowPlay != nil {
		playingID = a.playback.NowPlay.ID
	}

	var mainView string
	switch a.sidebarActive {
	case SideSearch:
		mainView = a.left.ViewSearchContent(mainW, contentH, playingID)

	case SideDownloads:
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
		tracklistView := a.left.ViewPlaylistsContent(layout.TracklistWidth, contentH, playingID)
		col3View := a.renderThirdColumn(layout.ThirdColumnWidth, contentH)
		mainView = lipgloss.JoinHorizontal(lipgloss.Top, tracklistView, col3View)

	case SideLogs:
		mainView = renderLogsView(a.logOffset, mainW, contentH, inputNotFocused)

	default:
		mainView = a.renderLyricsView(mainW, contentH, inputNotFocused, frame)
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
	if a.statusMsg != "" && time.Since(a.statusAt) < 3*time.Second {
		status = "  " + a.statusMsg
	}

	rKeyStyle := lipgloss.NewStyle().Foreground(colorDark)
	if a.playback.Random {
		rKeyStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	}

	help := "  " +
		styleHint("tab", "nav") + "  " +
		styleHint("enter", "play") + "  " +
		styleHint("]", "next") + "  " +
		styleHint("[", "prev") + "  " +
		rKeyStyle.Render("r:") + lipgloss.NewStyle().Foreground(colorWhite).Render("random") + "  " +
		styleHint("space", "pause") + "  " +
		styleHint("/", "search") + "  " +
		styleHint("esc", "settings") + "  " +
		styleHint("alt+q", "quit")

	progressBar := a.renderProgressBar(a.width)

	var b strings.Builder
	if !a.hideLogo {
		b.WriteString(somRow + "\n")
	}
	b.WriteString(sep + "\n")
	b.WriteString(contentRow + "\n")
	b.WriteString(status + "\n")
	b.WriteString(progressBar + "\n")
	if !a.hideHint {
		b.WriteString(help)
	}

	return b.String()
}

func (a *App) renderOverlayView(view string) string {
	if len(a.modals) > 0 {
		popup := a.modals[len(a.modals)-1].View()
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, popup)
	}
	if a.left.showPlInput {
		popup := a.left.renderPlInputPopup()
		return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, popup)
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
		hint = styleHint("d", "download") + "  "
	case SideLyrics:
		if a.playback.NowPlay == nil || !a.right.loaded || len(a.right.lyrics.Synced) == 0 {
			return dashboard
		}
		hint = styleHint("up/down", "select") + "  " + styleHint("l", "lyric language") + "  "
	case SideImport:
		if a.importPanel.importing {
			return dashboard
		}
		hint = styleHint(".", "select") + "  " + styleHint("i", "import") + "  " + styleHint("r", "rescan") + "  "
	case SideDownloads:
		if a.moveSession != nil {
			plName := ""
			if a.moveSession.TargetPlIdx >= 0 && a.moveSession.TargetPlIdx < len(a.left.playlists) {
				plName = a.left.playlists[a.moveSession.TargetPlIdx].Name
			}
			hint = styleHint(".", "select") + "  " + styleHint("i", fmt.Sprintf("move to \"%s\" (%d)", plName, a.selectedMoveCount())) + "  " + styleHint("+", "already in playlist") + "  " + styleHint("esc", "cancel")
		} else {
			hint = styleHint("ctrl+p", "pin/unpin") + "  " + styleHint("\\", "visualizer") + "  " + styleHint(":", "Command") + "  "
		}
	case SidePlaylists:
		if a.left.showPlInput {
			return dashboard
		}
		hint = styleHint(",", "new playlist") + "  " + styleHint("delete", "its deletes :)") + "  "

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
func (a *App) renderLyricsView(w, h int, focused bool, frame int) string {
	if a.playback.NowPlay == nil {
		return lipgloss.NewStyle().
			Width(w).
			Height(h).
			Render(DimItemStyle.Render(" Play a track to see lyrics..."))
	}

	borderColor := themeCol("#7c7986")
	lyricsBox := a.right.renderLyricsBox(focused, borderColor, frame)
	return lipgloss.NewStyle().Width(w).Render(lyricsBox)
}

func (a *App) renderProgressBar(w int) string {
	dim := ProgressDimStyle
	controls := dim.Render("")

	innerW := w - 4

	elapsedSec := 0
	totalSec := 0
	if a.playback.NowPlay != nil {
		elapsedSec = int(a.right.elapsed.Seconds())
		if elapsedSec < 0 {
			elapsedSec = 0
		}

		totalSec = a.playback.NowPlay.Duration
		if totalSec > 0 && elapsedSec > totalSec {
			elapsedSec = totalSec
		}
	}

	timeStr := FormatDuration(elapsedSec)
	timeW := len([]rune(timeStr))

	timeStart := (innerW - timeW) / 2
	if timeStart < 0 {
		timeStart = 0
	}
	leftW := timeStart
	rightW := innerW - timeStart - timeW
	if rightW < 0 {
		rightW = 0
	}

	fillW := 0
	if totalSec > 0 {
		fillW = innerW * elapsedSec / totalSec
	}
	if fillW > innerW {
		fillW = innerW
	}

	leftFill := fillW
	if leftFill > leftW {
		leftFill = leftW
	}
	rightFill := fillW - leftW - timeW
	if rightFill < 0 {
		rightFill = 0
	}
	if rightFill > rightW {
		rightFill = rightW
	}

	labelFill := fillW - leftW
	if labelFill < 0 {
		labelFill = 0
	}
	if labelFill > timeW {
		labelFill = timeW
	}

	var bar strings.Builder
	bar.WriteString(ProgressFilledStyle.Render(strings.Repeat("█", leftFill)))
	bar.WriteString(strings.Repeat(" ", leftW-leftFill))
	if labelFill > 0 {
		bar.WriteString(ProgressTimeOnFillStyle.Render(string([]rune(timeStr)[:labelFill])))
	}
	if labelFill < timeW {
		bar.WriteString(ProgressTimeStyle.Render(string([]rune(timeStr)[labelFill:])))
	}
	bar.WriteString(ProgressFilledStyle.Render(strings.Repeat("█", rightFill)))

	progress := bar.String()
	borderColor := themeCol("#7c7986")
	borderChar := lipgloss.NewStyle().Foreground(borderColor)

	title := ""
	if a.playback.NowPlay != nil {
		title = a.playback.NowPlay.Title
	}
	borderW := w - 2
	if borderW < 0 {
		borderW = 0
	}
	var topBorder string
	if title == "" {
		topBorder = borderChar.Render("╭" + strings.Repeat("─", w-2) + "╮")
	} else {
		titleRendered := PanelTitleStyle.Foreground(borderColor).Render(title)
		titleW := lipgloss.Width(titleRendered)
		prefix := "╭── "
		prefixStyled := borderChar.Render(prefix)
		prefixW := lipgloss.Width(prefixStyled)
		remain := w - prefixW - titleW - 1
		if remain < 0 {
			remain = 0
		}
		topBorder = prefixStyled + titleRendered + borderChar.Render(strings.Repeat("─", remain)+"╮")
	}

	bottomBorder := borderChar.Render("╰" + strings.Repeat("─", w-2) + "╯")

	barPad := innerW - lipgloss.Width(progress)
	if barPad < 0 {
		barPad = 0
	}

	// Merge controls (left) + progress bar (right) into one line.
	combinedLine := borderChar.Render("│ ") +
		controls +
		progress +
		strings.Repeat(" ", barPad) +
		borderChar.Render(" │")

	return topBorder + "\n" + combinedLine + "\n" + bottomBorder
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
	)
	lyricBox := renderBox(
		layout.Width,
		"Lyrics",
		lyricContent,
		themeCol("#7c7986"),
	)

	duration := time.Since(a.sessionStart)
	hTime := int(duration.Hours())
	mTime := int(duration.Minutes()) % 60
	sTime := int(duration.Seconds()) % 60
	statsContent := fmt.Sprintf("Session: %02dh %02dm %02ds", hTime, mTime, sTime)

	statsView := lipgloss.NewStyle().
		Width(layout.Width).
		Align(lipgloss.Center).
		Render(DimItemStyle.Render(statsContent))

	return lipgloss.JoinVertical(
		lipgloss.Top,
		visView,
		lyricBox,
		statsView,
	)
}

func (a *App) renderLyricAnim(innerW, innerH int) string {
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
