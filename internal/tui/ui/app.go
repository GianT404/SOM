package ui

import (
	"fmt"
	"log"
	"time"

	"som/internal/domain"
	"som/internal/storage"
	"som/internal/tui/avrcp"
	"som/internal/tui/layout"
	"som/internal/tui/player"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type animeTickMsg time.Time

func animeTick() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return animeTickMsg(t)
	})
}

type sessionTickMsg time.Time

func sessionTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return sessionTickMsg(t)
	})
}

type MoveSession struct {
	TargetPlIdx int
	Selected    map[string]bool
}

type App struct {
	provider           domain.MusicProvider
	downloadDir        string
	player             *player.Player
	playback           *PlaybackManager
	width              int
	height             int
	left               LeftPanel
	right              RightPanel
	statusMsg          string
	statusAt           time.Time
	sessionStart       time.Time
	sidebarActive      SidebarItem
	logOffset          int
	activeContext      SidebarItem
	palette            CommandPalette
	booting            bool
	splashFrame        int
	animeFrame         int
	animeActive        bool
	pendingKeys        []tea.KeyPressMsg
	playbackTickActive bool
	modals             []Overlay
	hideHint           bool
	hideLogo           bool
	mouseEnabled       bool
	skipSilence        bool

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
		id := lf.Path
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
			if err := a.left.plStore.AddTrackToPlaylist(pl.ID, id); err == nil {
				pl.Tracks = append(pl.Tracks, storage.PlaylistTrack{
					ID: id, Title: lf.Name, Artist: lf.Artist, Duration: lf.Duration, Thumbnail: lf.Thumbnail, Path: lf.Path,
				})
				existing[id] = true
				added++
			}
		}
	}
	a.setStatus(StatusOKStyle.Render(fmt.Sprintf("> Changed: %d added, %d removed in \"%s\"", added, removed, pl.Name)))
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if a.booting {
		return a.updateBoot(msg)
	}

	modalCmd, stop := a.updateModal(msg)
	if stop {
		return a, modalCmd
	}

	var cmds []tea.Cmd
	if modalCmd != nil {
		cmds = append(cmds, modalCmd)
	}

	systemCmds, handled, stop := a.updateSystem(msg)
	cmds = append(cmds, systemCmds...)
	if !handled {
		cmds = append(cmds, a.routeEvents(msg)...)
	}
	if stop {
		return a, tea.Batch(cmds...)
	}

	cmds = append(cmds, a.updateComponents(msg)...)

	return a, tea.Batch(cmds...)
}

func (a *App) updateBoot(msg tea.Msg) (tea.Model, tea.Cmd) {
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

		cmds := []tea.Cmd{a.left.Init(), sessionTick()}
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

func (a *App) updateModal(msg tea.Msg) (tea.Cmd, bool) {
	if len(a.modals) == 0 {
		return nil, false
	}

	if _, ok := msg.(CloseAllModalsMsg); ok {
		a.modals = nil
		return nil, true
	}
	if _, ok := msg.(CloseModalMsg); ok {
		a.modals = a.modals[:len(a.modals)-1]
		return nil, true
	}

	top := len(a.modals) - 1
	var modalCmd tea.Cmd
	a.modals[top], modalCmd = a.modals[top].Update(msg)

	switch msg.(type) {
	case tea.KeyPressMsg, tea.MouseClickMsg, tea.MouseWheelMsg:
		return modalCmd, true
	default:
		return modalCmd, false
	}
}

func (a *App) updateSystem(msg tea.Msg) ([]tea.Cmd, bool, bool) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.resizePanels()
		return cmds, true, false

	case tea.MouseClickMsg:
		if a.mouseEnabled {
			if c := a.handleMouseClick(msg); c != nil {
				cmds = append(cmds, c)
			}
		}
		return cmds, true, false

	case tea.MouseWheelMsg:
		if a.mouseEnabled {
			a.handleMouseWheel(msg.Button == tea.MouseWheelUp)
		}
		return cmds, true, false

	case animeTickMsg:
		if !a.animeActive {
			return cmds, true, false
		}
		a.animeFrame++
		cmds = append(cmds, animeTick())
		return cmds, true, false

	case sessionTickMsg:
		cmds = append(cmds, sessionTick())
		return cmds, true, false

	case tickMsg:
		if !a.playbackTickActive {
			return cmds, true, false
		}

		if a.player == nil ||
			a.playback == nil ||
			!a.playback.SongStarted {
			a.playbackTickActive = false
			return cmds, true, false
		}

		cmds = append(cmds, a.handleTick())
		return cmds, true, false

	case spinner.TickMsg:
		if a.importPanel.importing {
			var c tea.Cmd
			a.importPanel.spinner, c = a.importPanel.spinner.Update(msg)
			cmds = append(cmds, c)
		}
		return cmds, true, false

	case tea.KeyPressMsg:
		oldTab := a.sidebarActive
		if c := a.handleKeys(msg); c != nil {
			cmds = append(cmds, c)
		}
		return cmds, true, oldTab != a.sidebarActive
	default:
		return cmds, false, false
	}
}

func (a *App) routeEvents(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd

	if c := a.handleAudioEvents(msg); c != nil {
		cmds = append(cmds, c)
	}
	if c := a.handleDataEvents(msg); c != nil {
		cmds = append(cmds, c)
	}

	return cmds
}

func isKeyPressMsg(msg tea.Msg) bool {
	_, ok := msg.(tea.KeyPressMsg)
	return ok
}
func (a *App) updateComponents(msg tea.Msg) []tea.Cmd {
	var cmds []tea.Cmd

	focusedContent := a.sidebarActive != SideImport &&
		(a.sidebarActive == SideSearch ||
			a.sidebarActive == SideDownloads ||
			a.sidebarActive == SideQueue ||
			a.sidebarActive == SidePlaylists)

	var leftCmd tea.Cmd
	if !(a.right.showLangPopup && isKeyPressMsg(msg)) {
		a.left, leftCmd = a.left.Update(msg, focusedContent, a.playback.NowPlay)
		cmds = append(cmds, leftCmd)
	}

	oldLyric := a.right.GetCurrentLyricLine()

	var rightCmd tea.Cmd

	lyricsFocused := a.sidebarActive == SideLyrics

	allowLanguagePopup :=
		(a.sidebarActive == SideDownloads ||
			a.sidebarActive == SidePlaylists) &&
			!a.textInputFocused()

	a.right, rightCmd = a.right.Update(
		msg,
		lyricsFocused,
		allowLanguagePopup,
	)
	cmds = append(cmds, rightCmd)

	if c := a.syncAnimeAnimation(); c != nil {
		cmds = append(cmds, c)
	}

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

	var pbCmd tea.Cmd
	a.playback, pbCmd = a.playback.Update(msg)
	cmds = append(cmds, pbCmd)

	var paletteCmd tea.Cmd
	a.palette, paletteCmd = a.palette.Update(msg)
	cmds = append(cmds, paletteCmd)

	// Chỉ tạo playback timer khi bài  đang phát.
	if a.player != nil &&
		a.playback != nil &&
		a.playback.SongStarted {

		switch a.player.State() {
		case player.Playing:
			if !a.playbackTickActive {
				a.playbackTickActive = true
				cmds = append(cmds, tick())
			}

		case player.Paused:
			a.playbackTickActive = false

		case player.Stopped:

		}
	} else {
		a.playbackTickActive = false
	}
	// Audio capture chỉ chạy khi spectrum thực sự cần nó.
	captureNeeded := false

	if a.player != nil &&
		a.playback != nil &&
		a.playback.SongStarted &&
		a.player.State() == player.Playing {
		captureNeeded = a.spectrumActive()
	}

	var captureCmd tea.Cmd
	a.palette, captureCmd = a.palette.SyncCapture(captureNeeded)
	cmds = append(cmds, captureCmd)

	return cmds
}

// somRowHeight trả số dòng banner SOM cần dành chỗ (0 khi đã ẩn logo).
func (a *App) somRowHeight() int {
	if a.hideLogo {
		return 0
	}
	return 2
}

func (a *App) uiLayout() layout.UILayout {
	return layout.NewUILayout(
		a.width,
		a.height,
		sidebarWidth,
		a.somRowHeight(),
		a.hideHint,
	)
}

func (a *App) mainContentHeight() int {
	return a.uiLayout().MainViewHeight
}

func (a *App) SessionDuration() time.Duration {
	if a.sessionStart.IsZero() {
		return 0
	}
	return time.Since(a.sessionStart)
}

func (a *App) switchSidebar(item SidebarItem) tea.Cmd {
	oldTab := a.sidebarActive
	if oldTab != item {
		saveInputForTab(&a.left, oldTab)
		a.sidebarActive = item
		a.left.activeTab = item
		loadInputForTab(&a.left, item)

		var cmds []tea.Cmd

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
			cmds = append(cmds)
		}

		if item == SideImport && oldTab != SideImport {
			a.importPanel.ScanImportDirs(a.downloadDir, a.left.plStore)
			a.importPanel.cursor = 0
			a.importPanel.offset = 0
		}

		if c := a.syncAnimeAnimation(); c != nil {
			cmds = append(cmds, c)
		}

		if len(cmds) == 0 {
			return nil
		}
		return tea.Batch(cmds...)
	}
	return nil
}

func (a *App) syncAnimeAnimation() tea.Cmd {
	thirdColumnVisible :=
		a.sidebarActive == SideDownloads ||
			a.sidebarActive == SidePlaylists

	shouldAnimate :=
		a.right.loaded &&
			!a.right.loadingLyrics &&
			a.right.noLyrics &&
			(a.sidebarActive == SideLyrics || thirdColumnVisible)

	if shouldAnimate {
		if !a.animeActive {
			a.animeActive = true
			a.animeFrame = 0
			return animeTick()
		}
		return nil
	}

	a.animeActive = false
	return nil
}

func (a *App) spectrumActive() bool {
	if a.palette.Visible() {
		return true
	}

	switch a.sidebarActive {
	case SideDownloads, SidePlaylists:
		return a.playback != nil && a.playback.NowPlay != nil
	default:
		return false
	}
}

func (a *App) resizePanels() {
	layout := a.uiLayout()

	a.left.SetSize(layout.MainWidth, layout.MainViewHeight)
	a.right.SetSize(layout.MainWidth, layout.MainViewHeight)
	a.palette.width = a.width
	a.palette.height = a.height
}

func (a *App) setStatus(s string) {
	a.statusMsg = s
	a.statusAt = time.Now()
}

func init() {
	// Toàn bộ log chỉ đi vào ring buffer trong RAM, không ghi file đĩa nào cả. Khi crash, ring buffer mới được dump ra file.
	log.SetOutput(LogBuf)
}
