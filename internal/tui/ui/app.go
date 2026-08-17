package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"som/internal/domain"
	"som/internal/tui/player"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
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

type App struct {
	provider      domain.MusicProvider
	player        *player.Player
	nowPlay       *domain.Track
	songStarted   bool
	width         int
	height        int
	left          LeftPanel
	right         RightPanel
	statusMsg     string
	statusAt      time.Time
	showHelpPopup bool
	playlist      []domain.Track
	currentIdx    int
	random        bool
	shuffleHist   []int

	sidebarActive SidebarItem
	sidebarAnim   sidebarAnimState
	logOffset     int
	activeContext SidebarItem
	palette       CommandPalette
	booting       bool
	splashFrame   int
	pendingKeys   []tea.KeyMsg
}

const maxPendingKeys = 64

func NewApp(provider domain.MusicProvider) *App {
	return &App{
		provider:      provider,
		sidebarActive: SideDownloads,
		activeContext: SideDownloads,
		palette:       NewCommandPalette(),
		booting:       true,
	}
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(splashTick(), bootCmd(a.provider))
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if a.booting {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			a.width = msg.Width
			a.height = msg.Height
			// Truyền size xuống palette (visualizer) ngay cả khi đang boot,
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
			a.resizePanels()

			// Replay phím bấm trong lúc boot để không bị nuốt.
			cmds := []tea.Cmd{a.left.Init(), tick(), animTick(), logoTick()}
			for _, k := range a.pendingKeys {
				_, c := a.Update(k)
				if c != nil {
					cmds = append(cmds, c)
				}
			}
			a.pendingKeys = nil
			return a, tea.Batch(cmds...)

		case tea.KeyMsg:
			if msg.String() == "ctrl+c" {
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

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.resizePanels()

	case tickMsg:
		a.left.animTick++
		a.right.TickAt()
		// resolve stream thất bại trước khi phát thì không tự trỏ qua bài khác.
		if !a.left.loadingStream && a.songStarted && a.player.State() == player.Stopped && a.nowPlay != nil {
			playErr := a.player.PlaybackError()
			a.nowPlay = nil
			if playErr != nil {
				// Stream/file lỗi giữa chừng: dừng, không tự nhảy bài tiếp.
				a.setStatus(StatusErrStyle.Render("X playback failed: " + playErr.Error()))
			} else {
				cmds = append(cmds, a.playNext())
			}
		}

		cmds = append(cmds, tick())
	case PlayPlaylistMsg:
		a.playlist = msg.Tracks
		a.shuffleHist = nil
		a.activeContext = SidePlaylists
		a.left.loadingStream = true
		cmds = append(cmds, a.left.spinner.Tick, a.playTrackAt(msg.Index, msg.Tracks[msg.Index]))
	case tea.KeyMsg:
		if a.showHelpPopup {
			switch msg.String() {
			case "?", "esc", "q":
				a.showHelpPopup = false
			}
			return a, nil
		}

		switch msg.String() {

		case "ctrl+c":
			a.player.Stop()
			return a, tea.Quit

		case "alt+q":
			a.player.Stop()
			return a, tea.Quit

		case "esc":
			if a.palette.Visible() {
				a.palette.Close()
			}
		case "1", "2", "3", "4", "5":
			if a.left.input.Focused() {
				break
			}
			targetTab := SidebarItem(msg.String()[0] - '1')
			return a, a.switchSidebar(targetTab)
		case ":":
			if a.left.input.Focused() {
				break
			}
			if a.palette.Visible() {
				a.palette.Close()
			} else {
				cmds = append(cmds, a.palette.Open())
			}

		case "tab":
			if a.left.input.Focused() {
				a.left.input.Blur()
			} else {
				next := (a.sidebarActive + 1) % sideCount
				cmds = append(cmds, a.switchSidebar(next))
			}

		case " ":
			if a.left.input.Focused() {
				break
			}
			a.player.TogglePause()

		case "right":
			if a.left.input.Focused() {
				break
			}
			a.player.SeekBy(5)
			a.right.TickAt()

		case "left":
			if a.left.input.Focused() {
				break
			}
			a.player.SeekBy(-5)
			a.right.TickAt()

		case "]", "}":
			if a.left.input.Focused() {
				break
			}
			cmds = append(cmds, a.playNext())

		case "[", "{":
			if a.left.input.Focused() {
				break
			}
			cmds = append(cmds, a.playPrev())

		case "r", "R":
			if a.left.input.Focused() {
				break
			}
			a.random = !a.random
			a.shuffleHist = nil
			a.syncPlaylistState()

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
			if a.left.input.Focused() {
				break
			}
			v := a.player.Volume() + 0.05
			a.player.SetVolume(v)
			a.setStatus(StatusMsgStyle.Render(fmt.Sprintf("Volume: %d%%", int(math.Round(v*100)))))

		case "-", "_":
			if a.left.input.Focused() {
				break
			}
			v := a.player.Volume() - 0.05
			a.player.SetVolume(v)
			if v <= 0.01 {
				a.setStatus(StatusMsgStyle.Render("Volume: MUTE"))
			} else {
				a.setStatus(StatusMsgStyle.Render(fmt.Sprintf("Volume: %d%%", int(math.Round(v*100)))))
			}
		case "?":
			if a.left.input.Focused() {
				break
			}
			a.showHelpPopup = true
			return a, nil

		case "enter":

		}

	case SearchResultMsg:
		if msg.Err != nil {
			a.setStatus(StatusErrStyle.Render("X " + msg.Err.Error()))
		}

	case PlayStartedMsg:
		t := msg.Track
		a.playlist = a.left.tracks
		a.shuffleHist = nil
		a.activeContext = SideSearch

		idx := -1
		for i, tr := range a.playlist {
			if tr.ID == t.ID {
				idx = i
				break
			}
		}
		a.left.loadingStream = true
		cmds = append(cmds, a.left.spinner.Tick, a.playTrackAt(idx, t))
	case animTickMsg:
		// Duy trì ticker ghost khi animation sidebar còn chạy.
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
	case PlayLocalMsg:
		locals := a.left.getFilteredLocals()
		if len(locals) == 0 {
			locals = a.left.locals
		}
		if len(locals) == 0 {
			a.setStatus(StatusErrStyle.Render("X No local files found"))
			break
		}
		a.playlist = make([]domain.Track, len(locals))
		a.shuffleHist = nil
		idx := -1
		for i, lf := range locals {
			a.playlist[i] = domain.Track{
				ID:       "local:" + lf.Path,
				Title:    lf.Name,
				Artist:   lf.Artist,
				Duration: lf.Duration,
			}
			if lf.Name == msg.Title {
				idx = i
			}
		}
		if idx < 0 {
			idx = len(a.playlist) - 1
		}
		a.activeContext = SideDownloads
		cmds = append(cmds, a.playTrackAt(idx, a.playlist[idx]))

	case StreamStartedMsg:
		a.left.loadingStream = false
		if msg.Err != nil {
			a.setStatus(StatusErrStyle.Render("X " + msg.Err.Error()))
			break
		}
		a.nowPlay = &msg.Track
		a.songStarted = true
		a.right.SetTrack(&msg.Track)
		a.setStatus(StatusOKStyle.Render(">  " + msg.Track.Title))
		if msg.LyricsErr != nil {
			a.right.SetLyrics(domain.LyricsResp{Plain: "(no lyrics available)"})
		} else {
			a.right.SetLyrics(msg.Lyrics)
		}
		// start lyrics spinner
		cmds = append(cmds, a.right.spinner.Tick)

	case DownloadDoneMsg:
		if msg.Err != nil {
			a.setStatus(StatusErrStyle.Render(msg.Err.Error()))
		} else {
			a.setStatus(StatusOKStyle.Render("Saved " + msg.Path))
		}

	}

	focusedContent := a.sidebarActive == SideSearch || a.sidebarActive == SideDownloads || a.sidebarActive == SidePlaylists
	var leftCmd tea.Cmd
	a.left, leftCmd = a.left.Update(msg, focusedContent)
	cmds = append(cmds, leftCmd)

	var rightCmd tea.Cmd
	a.right, rightCmd = a.right.Update(msg, a.sidebarActive == SideLyrics)
	cmds = append(cmds, rightCmd)

	var paletteCmd tea.Cmd
	a.palette, paletteCmd = a.palette.Update(msg)
	cmds = append(cmds, paletteCmd)

	return a, tea.Batch(cmds...)
}

func (a *App) mainContentHeight() int {
	statusH := 0
	if a.statusMsg != "" && time.Since(a.statusAt) < 5*time.Second {
		statusH = 1
	}
	// somRow(6) + sep(1) + progressBar(4) + help(1) + status
	overhead := 6 + 1 + 4 + 1 + statusH
	contentH := a.height - overhead
	if contentH < 5 {
		contentH = 5
	}
	return contentH
}

func (a *App) View() string {
	if a.booting || a.width == 0 {
		return renderSplash(a.width, a.height, a.splashFrame)
	}

	contentH := a.mainContentHeight()
	sideH := contentH
	mainW := a.width - sidebarWidth
	frame := a.splashFrame
	if mainW < 10 {
		mainW = 10
	}

	somRow := renderSOMLogo()

	sep := lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", a.width))

	sideView := renderSidebar(a.sidebarActive, a.sidebarAnim, sideH)

	inputNotFocused := !a.left.input.Focused()
	var mainView string
	switch a.sidebarActive {
	case SideSearch:
		mainView = a.left.ViewSearchContent(mainW, contentH)
	case SideDownloads:
		mainView = a.left.ViewDownloadsContent(mainW, contentH)
	case SidePlaylists:
		mainView = a.left.ViewPlaylistsContent(mainW, contentH)
	case SideLogs:
		mainView = renderLogsView(a.logOffset, mainW, contentH, inputNotFocused)
	default:
		mainView = a.renderLyricsView(mainW, contentH, inputNotFocused, frame)
	}
	contentRow := lipgloss.JoinHorizontal(lipgloss.Top, sideView, mainView)

	status := ""
	if a.statusMsg != "" && time.Since(a.statusAt) < 5*time.Second {
		status = "  " + a.statusMsg
	}

	rStyle := HelpStyle
	if a.random {
		rStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	}
	help := HelpStyle.Render("  tab:nav  enter:play  ]:next  [:prev ") +
		rStyle.Render("r") +
		HelpStyle.Render(":random  d:download  space:pause  /:search  ?:help alt + q:quit")

	progressBar := a.renderProgressBar(a.width)

	var b strings.Builder
	b.WriteString(somRow + "\n")
	b.WriteString(sep + "\n")
	b.WriteString(contentRow + "\n")
	if status != "" {
		b.WriteString(status + "\n")
	}
	b.WriteString(progressBar + "\n")
	b.WriteString(help)

	view := b.String()

	if a.showHelpPopup {
		popup := a.renderHelpPopup()
		view = lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, popup)
	} else if a.left.showPlInput {
		popup := a.left.renderPlInputPopup()
		view = lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, popup)
	} else if a.left.showAddPopup {
		popup := a.left.renderAddPopup()
		view = lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, popup)
	} else if a.left.showDeletePopup {
		popup := a.left.renderDeletePopup()
		view = lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, popup)
	} else if a.palette.Visible() {
		popup := a.palette.View()
		view = lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, popup)
	}

	return view
}

func (a *App) renderLyricsView(w, h int, focused bool, frame int) string {
	if a.nowPlay == nil {
		return lipgloss.NewStyle().
			Width(w).
			Height(h).
			Render(DimItemStyle.Render(" Play a track to see lyrics..."))
	}

	borderColor := lipgloss.Color("#7c7986")
	lyricsBox := a.right.renderLyricsBox(focused, borderColor, frame)
	return lipgloss.NewStyle().Width(w).Render(lyricsBox)
}

func (a *App) renderProgressBar(w int) string {
	dim := ProgressDimStyle
	controls := dim.Render("")

	innerW := w - 4

	elapsedSec := 0
	totalSec := 0
	if a.nowPlay != nil {
		elapsedSec = int(a.right.elapsed.Seconds())
		if elapsedSec < 0 {
			elapsedSec = 0
		}

		totalSec = a.nowPlay.Duration
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
	borderColor := lipgloss.Color("#7c7986")
	borderChar := lipgloss.NewStyle().Foreground(borderColor)

	title := ""
	if a.nowPlay != nil {
		title = a.nowPlay.Title
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

	controlsPad := innerW - lipgloss.Width(controls)
	if controlsPad < 0 {
		controlsPad = 0
	}
	controlsLine := borderChar.Render("│ ") +
		lipgloss.NewStyle().
			Width(innerW).
			Align(lipgloss.Center).
			Render(controls) +
		borderChar.Render(" │")

	barPad := innerW - lipgloss.Width(progress)
	if barPad < 0 {
		barPad = 0
	}
	barLine := borderChar.Render("│ ") + progress + strings.Repeat(" ", barPad) + borderChar.Render(" │")

	return topBorder + "\n" + controlsLine + "\n" + barLine + "\n" + bottomBorder
}

func (a *App) playTrackAt(idx int, t domain.Track) tea.Cmd {
	a.currentIdx = idx
	a.nowPlay = &t
	a.songStarted = false
	a.syncPlaylistState()

	if idx >= 0 {
		vis := a.left.visibleRows()
		switch a.activeContext {
		case SideSearch:
			a.left.searchCursor = idx
			if a.left.searchCursor < a.left.searchOffset {
				a.left.searchOffset = a.left.searchCursor
			}
			if a.left.searchCursor >= a.left.searchOffset+vis {
				a.left.searchOffset = a.left.searchCursor - vis + 1
			}
		case SideDownloads:
			a.left.dlCursor = idx
			if a.left.dlCursor < a.left.dlOffset {
				a.left.dlOffset = a.left.dlCursor
			}
			if a.left.dlCursor >= a.left.dlOffset+vis {
				a.left.dlOffset = a.left.dlCursor - vis + 1
			}
		case SidePlaylists:
			if a.left.activePlaylist != nil {
				a.left.plCursor = idx
				if a.left.plCursor < a.left.plOffset {
					a.left.plOffset = a.left.plCursor
				}
				if a.left.plCursor >= a.left.plOffset+vis {
					a.left.plOffset = a.left.plCursor - vis + 1
				}
			}
		}
	}

	if strings.HasPrefix(t.ID, "local:") {
		path := strings.TrimPrefix(t.ID, "local:")
		if err := a.player.Play(path); err != nil {
			a.setStatus(StatusErrStyle.Render("X " + err.Error()))
			return nil
		}
		a.songStarted = true
		a.right.SetTrack(&t)
		a.setStatus(StatusOKStyle.Render(">  " + t.Title))
		jsonPath := strings.TrimSuffix(path, ".opus") + ".json"
		data, err := os.ReadFile(jsonPath)
		if err == nil {
			var lr domain.LyricsResp
			if json.Unmarshal(data, &lr) == nil {
				a.right.SetLyrics(lr)
			}
		} else {
			a.right.SetLyrics(domain.LyricsResp{Plain: "(No lyrics available)"})
		}
		return nil
	}

	return func() tea.Msg {
		directURL, err := a.provider.ResolveStream(context.Background(), t.ID)
		if err != nil || directURL == "" {
			return StreamStartedMsg{Err: fmt.Errorf("lỗi lấy link CDN: %v", err)}
		}

		if err := a.player.Play(directURL); err != nil {
			return StreamStartedMsg{Err: err}
		}

		lr, lyricsErr := getCachedLyrics(a.provider, t.ID, t.Title, t.Artist, t.Duration)
		return StreamStartedMsg{
			Track:     t,
			Lyrics:    lr,
			LyricsErr: lyricsErr,
		}
	}
}

func (a *App) playNext() tea.Cmd {
	if len(a.playlist) == 0 {
		return nil
	}
	next := a.currentIdx + 1
	if a.random {
		next = a.pickAntiClumpIndex()
	}
	if next >= len(a.playlist) {
		return nil
	}
	return a.playTrackAt(next, a.playlist[next])
}

func (a *App) playPrev() tea.Cmd {
	if len(a.playlist) == 0 {
		return nil
	}
	prev := a.currentIdx - 1
	if a.random {
		prev = a.pickAntiClumpIndex()
	}
	if prev < 0 {
		return nil
	}
	return a.playTrackAt(prev, a.playlist[prev])
}

func (a *App) pickAntiClumpIndex() int {
	n := len(a.playlist)
	if n <= 1 {
		return 0
	}
	histCap := n / 2
	if histCap > 8 {
		histCap = 8
	}

	recent := make(map[int]bool, len(a.shuffleHist))
	start := len(a.shuffleHist) - histCap
	if start < 0 {
		start = 0
	}
	for _, idx := range a.shuffleHist[start:] {
		recent[idx] = true
	}

	curArtist := ""
	if a.currentIdx >= 0 && a.currentIdx < n {
		curArtist = a.playlist[a.currentIdx].Artist
	}

	var freshDiffArtist, freshSameArtist, usedDiffArtist []int
	for i, t := range a.playlist {
		if i == a.currentIdx {
			continue
		}
		diffArtist := curArtist == "" || t.Artist != curArtist
		if recent[i] {
			if diffArtist {
				usedDiffArtist = append(usedDiffArtist, i)
			}
			continue
		}
		if diffArtist {
			freshDiffArtist = append(freshDiffArtist, i)
		} else {
			freshSameArtist = append(freshSameArtist, i)
		}
	}

	pool := freshDiffArtist
	if len(pool) == 0 {
		pool = freshSameArtist
	}
	if len(pool) == 0 {
		pool = usedDiffArtist
	}
	if len(pool) == 0 {
		for i := range a.playlist {
			if i != a.currentIdx {
				pool = append(pool, i)
			}
		}
	}

	picked := pool[rand.Intn(len(pool))]

	a.shuffleHist = append(a.shuffleHist, picked)
	if len(a.shuffleHist) > histCap*2 {
		a.shuffleHist = a.shuffleHist[len(a.shuffleHist)-histCap:]
	}

	return picked
}

func (a *App) syncPlaylistState() {
	if a.playlist != nil {
		a.right.SetPlaylistState(a.currentIdx, len(a.playlist), a.random)
	}
}

func (a *App) switchSidebar(item SidebarItem) tea.Cmd {
	oldTab := a.sidebarActive
	if oldTab != item {
		// Giữ value input riêng cho từng tab.
		saveInputForTab(&a.left, oldTab)
	}
	a.sidebarActive = item
	a.left.activeTab = item
	loadInputForTab(&a.left, item)

	var cmds []tea.Cmd

	if item == SideSearch {
		a.left.searchOnEnter = true
		cmds = append(cmds, a.left.input.Focus())
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

func (a *App) resizePanels() {
	mainW := a.width - sidebarWidth
	if mainW < 10 {
		mainW = 10
	}
	contentH := a.mainContentHeight()
	a.left.SetSize(mainW, contentH)
	a.right.SetSize(mainW, contentH)
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
