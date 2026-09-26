package ui

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"som/internal/domain"
	"som/internal/storage"
	"som/internal/tui/avrcp"
	"som/internal/tui/player"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type animTickMsg time.Time

func animTick() tea.Cmd {
	return tea.Tick(270*time.Millisecond, func(t time.Time) tea.Msg {
		return animTickMsg(t)
	})
}

func sidebarAnimTick() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return animTickMsg(t)
	})
}

type logoTickMsg time.Time

func logoTick() tea.Cmd {
	return tea.Tick(63*time.Millisecond, func(t time.Time) tea.Msg {
		return logoTickMsg(t)
	})
}

type MoveSession struct {
	TargetPlIdx int
	Selected    map[string]bool
}

type App struct {
	provider      domain.MusicProvider
	downloadDir   string
	player        *player.Player
	playback      *PlaybackManager
	width         int
	height        int
	left          LeftPanel
	right         RightPanel
	statusMsg     string
	statusAt      time.Time
	sessionStart  time.Time
	sidebarActive SidebarItem
	sidebarAnim   sidebarAnimState
	logOffset     int
	activeContext SidebarItem
	palette       CommandPalette
	booting       bool
	splashFrame   int
	pendingKeys   []tea.KeyPressMsg

	modals       []Overlay
	hideHint     bool
	hideLogo     bool
	mouseEnabled bool
	skipSilence  bool

	mouseLastClickAt  time.Time
	mouseLastClickY   int
	mouseLastClickTab SidebarItem
	avrcp             *avrcp.Server
	activePreset      int
	activeSpeed       int
	activeSort        string
	importPanel       ImportPanel
	moveSession       *MoveSession
	prevLyric         string
	currLyric         string
	lyricAnimStart    time.Time
}
type Overlay interface {
	Init() tea.Cmd
	Update(tea.Msg) (Overlay, tea.Cmd)
	View() string
}

const maxPendingKeys = 64

func NewApp(provider domain.MusicProvider, downloadDir string) *App {
	mi := textinput.New()
	mi.CharLimit = 50
	mi.Prompt = ""
	return &App{
		provider:      provider,
		downloadDir:   downloadDir,
		sidebarActive: SideDownloads,
		activeContext: SideDownloads,
		sessionStart:  time.Now(),
		palette:       NewCommandPalette(),
		booting:       true,
		activeSpeed:   3,
		mouseEnabled:  false,
		importPanel:   NewImportPanel(),
		playback:      NewPlaybackManager(),
	}
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(splashTick(), bootCmd(a.provider, a.downloadDir))
}

func (a *App) selectedMoveCount() int {
	n := 0
	if a.moveSession != nil {
		for _, v := range a.moveSession.Selected {
			if v {
				n++
			}
		}
	}
	return n
}

func (a *App) toggleMoveSelection() {
	locals := a.left.getFilteredLocals()
	if a.left.dlCursor < 0 || a.left.dlCursor >= len(locals) {
		return
	}
	if a.moveSession != nil {
		path := locals[a.left.dlCursor].Path
		a.moveSession.Selected[path] = !a.moveSession.Selected[path]
	}
}

func (a *App) applyMoveToPlaylist() {
	if a.moveSession == nil || a.moveSession.TargetPlIdx < 0 || a.moveSession.TargetPlIdx >= len(a.left.playlists) {
		return
	}
	pl := &a.left.playlists[a.moveSession.TargetPlIdx]
	existing := make(map[string]bool, len(pl.Tracks))
	for _, t := range pl.Tracks {
		existing[t.ID] = true
	}

	added, removed := 0, 0
	for _, lf := range a.left.locals {
		if !a.moveSession.Selected[lf.Path] {
			continue
		}
		id := "local:" + lf.Path
		if existing[id] {
			if err := a.left.plStore.RemoveTrackFromPlaylist(pl.ID, lf.Path); err == nil {
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
			if err := a.left.plStore.AddTrackToPlaylist(pl.ID, lf.Path); err == nil {
				pl.Tracks = append(pl.Tracks, storage.PlaylistTrack{
					ID: lf.Path, Title: lf.Name, Artist: lf.Artist, Duration: lf.Duration, Thumbnail: lf.Thumbnail, Path: lf.Path,
				})
				existing[id] = true
				added++
			}
		}
	}
	a.setStatus(StatusOKStyle.Render(fmt.Sprintf("> Changed: %d added, %d removed in \"%s\"", added, removed, pl.Name)))
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	//  Booting
	if a.booting {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			a.width = msg.Width
			a.height = msg.Height
			a.palette.width = msg.Width
			a.palette.height = msg.Height
			return a, nil
		case splashTickMsg:
			a.splashFrame++
			return a, splashTick()
		case splashDoneMsg:
			a.player = msg.player
			a.right = NewRightPanel(msg.player)
			a.left = msg.left
			a.left.input.Blur()
			a.booting = false
			a.playback.SetDependencies(msg.player, a.provider, a.left.plStore)
			a.loadSettings()
			a.resizePanels()
			a.avrcp = avrcp.New()
			var resumeCmd tea.Cmd
			a.palette, resumeCmd = a.palette.Resume()
			cmds := []tea.Cmd{a.left.Init(), tick(), animTick(), logoTick(), resumeCmd}
			if a.avrcp != nil {
				cmds = append(cmds, a.avrcp.WatchCommands())
			}
			for _, k := range a.pendingKeys {
				_, c := a.Update(k)
				if c != nil {
					cmds = append(cmds, c)
				}
			}
			a.pendingKeys = nil
			return a, tea.Batch(cmds...)
		case tea.KeyPressMsg:
			if msg.String() == "ctrl+c" || msg.String() == "alt+q" {
				return a, tea.Quit
			}
			if len(a.pendingKeys) < maxPendingKeys {
				a.pendingKeys = append(a.pendingKeys, msg)
			}
			return a, nil
		default:
			return a, nil
		}
	}

	var cmds []tea.Cmd
	//  Modal Overlay (Highest Priority)
	if len(a.modals) > 0 {
		if _, ok := msg.(CloseAllModalsMsg); ok {
			a.modals = nil
			return a, nil
		}
		if _, ok := msg.(CloseModalMsg); ok {
			a.modals = a.modals[:len(a.modals)-1]
			return a, nil
		}

		var modalCmd tea.Cmd
		top := len(a.modals) - 1
		a.modals[top], modalCmd = a.modals[top].Update(msg)

		switch msg.(type) {
		case tea.KeyPressMsg, tea.MouseClickMsg, tea.MouseWheelMsg:
			return a, modalCmd
		}
		cmds = append(cmds, modalCmd)
	}

	// System & Routing
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.resizePanels()
	case tea.MouseClickMsg:
		if a.mouseEnabled {
			if c := a.handleMouseClick(msg); c != nil {
				cmds = append(cmds, c)
			}
		}
	case tea.MouseWheelMsg:
		if a.mouseEnabled {
			a.handleMouseWheel(msg.Button == tea.MouseWheelUp)
		}
	case tickMsg:
		cmds = append(cmds, a.handleTick())
	case animTickMsg:
		if a.sidebarAnim.on && time.Now().Before(a.sidebarAnim.end) {
			cmds = append(cmds, sidebarAnimTick())
		} else if a.sidebarAnim.on {
			a.sidebarAnim.on = false
		}
		if a.sidebarActive == SideDownloads {
			a.left.animTick++
			cmds = append(cmds, animTick())
		}
	case logoTickMsg:
		a.splashFrame++
		cmds = append(cmds, logoTick())
	case spinner.TickMsg:
		if a.importPanel.importing {
			var c tea.Cmd
			a.importPanel.spinner, c = a.importPanel.spinner.Update(msg)
			cmds = append(cmds, c)
		}
	case tea.KeyPressMsg:
		oldTab := a.sidebarActive
		if c := a.handleKeys(msg); c != nil {
			cmds = append(cmds, c)
		}

		if oldTab != a.sidebarActive {
			return a, tea.Batch(cmds...)
		}
	default:
		// Uỷ quyền cho 2 Sub-Routers
		if c := a.handleAudioEvents(msg); c != nil {
			cmds = append(cmds, c)
		}
		if c := a.handleDataEvents(msg); c != nil {
			cmds = append(cmds, c)
		}
	}

	//  Update Component Con
	focusedContent := a.sidebarActive != SideImport && (a.sidebarActive == SideSearch || a.sidebarActive == SideDownloads || a.sidebarActive == SideQueue || a.sidebarActive == SidePlaylists)
	var leftCmd tea.Cmd

	a.left, leftCmd = a.left.Update(msg, focusedContent, a.playback.NowPlay)
	cmds = append(cmds, leftCmd)

	oldLyric := a.right.GetCurrentLyricLine()
	var rightCmd tea.Cmd
	a.right, rightCmd = a.right.Update(msg, a.sidebarActive == SideLyrics)
	cmds = append(cmds, rightCmd)

	newLyric := a.right.GetCurrentLyricLine()
	if oldLyric != newLyric {
		a.prevLyric = oldLyric
		a.currLyric = newLyric
		a.lyricAnimStart = time.Now()
	}

	if a.sidebarActive == SideImport {
		if km, ok := msg.(tea.KeyMsg); ok {
			cmds = append(cmds, a.handleImportKeys(km)...)
		}
	}
	cmds = append(cmds, rightCmd)

	var paletteCmd tea.Cmd
	a.palette, paletteCmd = a.palette.Update(msg)
	cmds = append(cmds, paletteCmd)

	var pbCmd tea.Cmd
	a.playback, pbCmd = a.playback.Update(msg)
	cmds = append(cmds, pbCmd)

	return a, tea.Batch(cmds...)
}

// somRowHeight trả số dòng banner SOM cần dành chỗ (0 khi đã ẩn logo).
func (a *App) somRowHeight() int {
	if a.hideLogo {
		return 0
	}
	return 2
}

func (a *App) mainContentHeight() int {
	statusH := 0
	if a.statusMsg != "" && time.Since(a.statusAt) < 3*time.Second {
		statusH = 1
	}
	helpH := 0
	if !a.hideHint {
		helpH = 1
	}
	// somRow(somRowHeight) + sep(1) + progressBar(3) + help(helpH) + status
	overhead := a.somRowHeight() + 1 + 3 + helpH + statusH
	contentH := a.height - overhead
	if contentH < 5 {
		contentH = 5
	}
	return contentH
}

func (a *App) View() tea.View {
	if a.booting || a.width == 0 {
		v := tea.NewView(renderSplash(a.width, a.height, a.splashFrame))
		v.AltScreen = true
		return v
	}

	contentH := a.mainContentHeight()
	sideH := contentH
	mainW := a.width - sidebarWidth - 1
	frame := a.splashFrame
	if mainW < 10 {
		mainW = 10
	}
	var mainView string
	dashboard := renderDashboard(a.hideLogo, a.player.Volume(), a.activeSpeed, a.activePreset, a.playback.NowPlay)
	somRow := a.renderSomRow(dashboard)

	borderStyle := lipgloss.NewStyle().Foreground(colorBorder)
	sepLeft := strings.Repeat("─", sidebarWidth)
	sepRight := ""
	if a.width > sidebarWidth+1 {
		sepRight = strings.Repeat("─", a.width-sidebarWidth-1)
	}
	sep := borderStyle.Render(sepLeft + "─" + sepRight)
	contentTop := a.somRowHeight() + 1

	inputNotFocused := !a.left.input.Focused()

	playingID := ""
	if a.playback.NowPlay != nil {
		playingID = a.playback.NowPlay.ID
	}
	col3W := int(float64(mainW) * 0.3)
	if col3W < 25 {
		col3W = 25
	}
	tracklistW := mainW
	if a.sidebarActive == SideDownloads || a.sidebarActive == SidePlaylists {
		tracklistW = mainW - col3W - 1
	}
	switch a.sidebarActive {

	case SideSearch:
		mainView = a.left.ViewSearchContent(mainW, contentH, playingID)
	case SideDownloads:
		var alreadyInMove map[string]bool
		var selected map[string]bool
		selectMode := false
		if a.moveSession != nil && a.moveSession.TargetPlIdx >= 0 && a.moveSession.TargetPlIdx < len(a.left.playlists) {
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
		tracklistView := a.left.ViewDownloadsContent(tracklistW, contentH, selected, selectMode, alreadyInMove, playingID)
		col3View := a.renderThirdColumn(col3W, contentH)
		sep := borderStyle.Render("│")
		mainView = lipgloss.JoinHorizontal(lipgloss.Top, tracklistView, sep, col3View)
	case SideImport:
		a.importPanel.SetSize(mainW, contentH)
		mainView = a.importPanel.ViewImportContent(mainW, contentH)
	case SideQueue:
		mainView = a.left.ViewQueueContent(mainW, contentH, a.playback.Queue, playingID)
	case SidePlaylists:
		tracklistView := a.left.ViewPlaylistsContent(tracklistW, contentH, playingID)
		col3View := a.renderThirdColumn(col3W, contentH)
		sep := borderStyle.Render("│")
		mainView = lipgloss.JoinHorizontal(lipgloss.Top, tracklistView, sep, col3View)
	case SideLogs:
		mainView = renderLogsView(a.logOffset, mainW, contentH, inputNotFocused)
	default:
		mainView = a.renderLyricsView(mainW, contentH, inputNotFocused, frame)
	}

	mainViewHeight := lipgloss.Height(mainView)
	borderH := mainViewHeight
	sideView := renderSidebar(a.sidebarActive, a.sidebarAnim, sideH-1, borderH)
	contentRow := lipgloss.JoinHorizontal(lipgloss.Top, sideView, mainView)

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
	if status != "" {
		b.WriteString(status + "\n")
	}
	b.WriteString(progressBar + "\n")
	if !a.hideHint {
		b.WriteString(help)
	}

	view := b.String()

	if len(a.modals) > 0 {
		popup := a.modals[len(a.modals)-1].View()
		view = lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, popup)
	} else if a.left.showPlInput {
		popup := a.left.renderPlInputPopup()
		view = lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, popup)
	} else if a.left.showDeletePopup {
		popup := a.left.renderDeletePopup()
		view = lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, popup)
	} else if a.palette.Visible() {
		popup := a.palette.View()
		view = lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, popup)
	}

	v := tea.NewView(view)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion

	if a.left.input.Focused() {
		if c := a.left.input.Cursor(); c != nil {
			c.Position.X += sidebarWidth + 3
			c.Position.Y = contentTop + 1
			v.Cursor = c
		}
	}

	return v
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

func (a *App) switchSidebar(item SidebarItem) tea.Cmd {
	oldTab := a.sidebarActive
	if oldTab != item {
		saveInputForTab(&a.left, oldTab)
		a.sidebarActive = item
		a.left.activeTab = item
		loadInputForTab(&a.left, item)

		var cmds []tea.Cmd

		if item == SideDownloads || item == SidePlaylists {
			var cmd tea.Cmd
			a.palette, cmd = a.palette.Resume()
			cmds = append(cmds, cmd)
		} else if !a.palette.Visible() {
			a.palette = a.palette.Pause()
		}
		if item == SideSearch {
			a.left.searchOnEnter = true
			if len(a.left.tracks) > 0 {
				a.left.input.Blur()
				a.left.suggestions = nil
				a.left.suggestCursor = 0
				a.left.suggestOffset = 0
				a.left.suggestFocus = false
			} else {
				cmds = append(cmds, a.left.input.Focus())
			}
		} else {
			a.left.searchOnEnter = false
			a.left.suggestions = nil
			a.left.suggestCursor = 0
			a.left.suggestOffset = 0
			a.left.suggestFocus = false
		}

		if item == SideDownloads && oldTab != SideDownloads {
			cmds = append(cmds, animTick())
		}

		if item == SideImport && oldTab != SideImport {
			a.importPanel.ScanImportDirs(a.downloadDir, a.left.plStore)
			a.importPanel.cursor = 0
			a.importPanel.offset = 0
		}

		if item != oldTab {
			a.sidebarAnim = sidebarAnimState{
				on:    true,
				from:  oldTab,
				to:    item,
				start: time.Now(),
				end:   time.Now().Add(sidebarGhostDuration),
			}
			cmds = append(cmds, sidebarAnimTick())
		}

		if len(cmds) == 0 {
			return nil
		}
		return tea.Batch(cmds...)
	}
	return nil
}

func (a *App) resizePanels() {
	mainW := a.width - sidebarWidth - 1
	if mainW < 10 {
		mainW = 10
	}
	contentH := a.mainContentHeight()
	a.left.SetSize(mainW, contentH)
	a.right.SetSize(mainW, contentH)
	a.palette.width = a.width
	a.palette.height = a.height
}

func (a *App) renderThirdColumn(w, h int) string {
	if w < 10 || h < 10 {
		return ""
	}

	statsH := 1 // App Time  chiếm đúng 1 dòng
	lyricH := int(float64(h) * 0.45)
	visH := h - lyricH - statsH - 1

	if visH < 1 {
		visH = 1
	}

	// 2. Spectrum
	visRaw := a.palette.RenderEQColumn(w-2, visH)
	visView := lipgloss.NewStyle().
		Width(w).
		Height(visH).
		Padding(0, 1).
		Render(visRaw)

	lyricInnerW := w - 4
	lyricInnerH := lyricH - 2
	if lyricInnerH < 1 {
		lyricInnerH = 1
	}
	lyricContent := a.renderLyricAnim(lyricInnerW, lyricInnerH)
	lyricBox := renderBox(w, "Lyrics", lyricContent, themeCol("#7c7986"))

	// App Time
	duration := time.Since(a.sessionStart)
	hTime := int(duration.Hours())
	mTime := int(duration.Minutes()) % 60
	sTime := int(duration.Seconds()) % 60
	statsContent := fmt.Sprintf("Session: %02dh %02dm %02ds", hTime, mTime, sTime)

	// Canh giữa text và làm mờ màu
	statsView := lipgloss.NewStyle().
		Width(w).
		Align(lipgloss.Center).
		Render(DimItemStyle.Render(statsContent))
	return lipgloss.JoinVertical(lipgloss.Top, visView, "", lyricBox, statsView)
}
func (a *App) setStatus(s string) {
	a.statusMsg = s
	a.statusAt = time.Now()
}

func (a *App) renderLyricAnim(innerW, innerH int) string {
	if innerH < 1 {
		return ""
	}

	if a.playback.NowPlay == nil {
		return lipgloss.Place(innerW, innerH, lipgloss.Center, lipgloss.Center, DimItemStyle.Render("Play a track to see lyrics..."))
	}

	progress := float64(time.Since(a.lyricAnimStart)) / float64(350*time.Millisecond)
	if progress >= 1.0 || a.prevLyric == "" {
		return lipgloss.Place(innerW, innerH, lipgloss.Center, lipgloss.Center, LyricHighlightStyle.Width(innerW).Align(lipgloss.Center).Render(a.currLyric))
	}

	gap := 2
	offset := int(math.Round(progress * float64(gap)))

	cY := innerH / 2
	oldY := cY - offset
	newY := cY + gap - offset

	lines := make([]string, innerH)

	putLine := func(y int, text string, style lipgloss.Style) {
		if y >= 0 && y < innerH && text != "" {
			lines[y] = style.Width(innerW).Align(lipgloss.Center).Render(text)
		}
	}

	oldStyle := LyricHighlightStyle
	newStyle := DimItemStyle
	if progress > 0.5 {
		oldStyle = DimItemStyle
		newStyle = LyricHighlightStyle
	}

	putLine(oldY, a.prevLyric, oldStyle)
	putLine(newY, a.currLyric, newStyle)

	return strings.Join(lines, "\n")
}

func init() {
	// Toàn bộ log chỉ đi vào ring buffer trong RAM, không ghi file đĩa nào cả. Khi crash, ring buffer mới được dump ra file.
	log.SetOutput(LogBuf)
}
