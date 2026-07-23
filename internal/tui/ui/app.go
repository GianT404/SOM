package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"som/internal/backend"
	"som/internal/tui/api"
	"som/internal/tui/player"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type App struct {
	client    *api.Client
	player    *player.Player
	width     int
	height    int
	left      LeftPanel
	right     RightPanel
	nowPlay   *api.Track
	playedAt  time.Time
	statusMsg string
	statusAt  time.Time

	playlist   []api.Track
	currentIdx int
	random     bool

	sidebarActive SidebarItem
	logOffset     int
}

func NewApp(serverURL string) *App {
	// Find a free port and start the backend server embedded in this process.
	port := freePort()
	go func() {
		ytdlpPath := os.Getenv("YTDLP_PATH")
		if ytdlpPath == "" {
			ytdlpPath = "yt-dlp"
		}
		ctx := context.Background()
		if err := backend.StartServer(ctx, port, ytdlpPath); err != nil {
			log.Printf("embedded backend error: %v", err)
		}
	}()

	localURL := fmt.Sprintf("http://localhost:%s", port)
	c := api.New(localURL)
	p := player.New()
	return &App{
		client:        c,
		player:        p,
		left:          NewLeftPanel(c),
		right:         NewRightPanel(p),
		sidebarActive: SideDownloads,
	}
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(a.left.Init(), tick())
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		log.Printf("WindowSizeMsg: width=%d height=%d", msg.Width, msg.Height)
		a.resizePanels()

	case tickMsg:
		a.right.TickAt(a.playedAt)
		if a.player.State() == player.Stopped && a.nowPlay != nil {
			a.nowPlay = nil
			cmds = append(cmds, a.playNext())
		}
		cmds = append(cmds, tick())

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			a.player.Stop()
			return a, tea.Quit

		case "q":
			if !a.left.input.Focused() {
				a.player.Stop()
				return a, tea.Quit
			}

		case "tab":
			if a.left.input.Focused() {
				a.left.input.Blur()
			} else {
				a.sidebarActive = (a.sidebarActive + 1) % sideCount
				a.switchSidebar(a.sidebarActive)
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
			a.right.SeekBy(5 * time.Second)

		case "left":
			if a.left.input.Focused() {
				break
			}
			a.player.SeekBy(-5)
			a.right.SeekBy(-5 * time.Second)

		case "n":
			if a.left.input.Focused() {
				break
			}
			cmds = append(cmds, a.playNext())

		case "p":
			if a.left.input.Focused() {
				break
			}
			cmds = append(cmds, a.playPrev())

		case "r":
			if a.left.input.Focused() {
				break
			}
			a.random = !a.random
			a.syncPlaylistState()

		case "up", "k":
			if a.sidebarActive == SideLogs {
				if a.logOffset < LogBuf.Len()-1 {
					a.logOffset++
				}
			}
		case "down", "j":
			if a.sidebarActive == SideLogs {
				if a.logOffset > 0 {
					a.logOffset--
				}
			}

		case "enter":

		}

	case SearchResultMsg:
		if msg.Err == nil {
			a.playlist = msg.Tracks
		}

	case PlayStartedMsg:
		t := msg.Track
		if len(a.left.tracks) > 0 {
			a.playlist = a.left.tracks
		}
		idx := -1
		for i, tr := range a.playlist {
			if tr.ID == t.ID {
				idx = i
				break
			}
		}
		a.left.loadingStream = true
		cmds = append(cmds, a.left.spinner.Tick, a.playTrackAt(idx, t))

	case PlayLocalMsg:
		locals := a.left.getFilteredLocals()
		if len(locals) == 0 {
			locals = a.left.locals
		}
		if len(locals) == 0 {
			a.setStatus(StatusErrStyle.Render("X No local files found"))
			break
		}
		a.playlist = make([]api.Track, len(locals))
		idx := -1
		for i, lf := range locals {
			a.playlist[i] = api.Track{
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
		cmds = append(cmds, a.playTrackAt(idx, a.playlist[idx]))

	case StreamStartedMsg:
		a.left.loadingStream = false
		if msg.Err != nil {
			a.setStatus(StatusErrStyle.Render("X " + msg.Err.Error()))
			break
		}
		a.playedAt = msg.PlayedAt
		a.nowPlay = &msg.Track
		a.right.SetTrack(&msg.Track, a.playedAt)
		a.setStatus(StatusOKStyle.Render(">  " + msg.Track.Title))
		if msg.LyricsErr != nil {
			a.right.SetLyrics(api.LyricsResp{Plain: "(no lyrics available)"}, a.playedAt)
		} else {
			a.right.SetLyrics(msg.Lyrics, a.playedAt)
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

	focusedContent := a.sidebarActive == SideSearch || a.sidebarActive == SideDownloads
	var leftCmd tea.Cmd
	a.left, leftCmd = a.left.Update(msg, focusedContent)
	cmds = append(cmds, leftCmd)

	var rightCmd tea.Cmd
	a.right, rightCmd = a.right.Update(msg, a.sidebarActive == SideLyrics)
	cmds = append(cmds, rightCmd)

	return a, tea.Batch(cmds...)
}

func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	// Layout: SOM + separator + sidebar|content + status + help
	statusH := 0
	if a.statusMsg != "" && time.Since(a.statusAt) < 5*time.Second {
		statusH = 1
	}
	overhead := 7 + 1 + statusH + 4 + 1 // SOM(6) + sep + status + progress + help
	contentH := a.height - overhead
	if contentH < 5 {
		contentH = 5
	}
	sideH := contentH
	mainW := a.width - sidebarWidth
	if mainW < 10 {
		mainW = 10
	}

	// SOM logo
	somRow := renderSOMLogo()

	// Separator
	sep := lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", a.width))

	// Sidebar
	sideView := renderSidebar(a.sidebarActive, sideH)

	// Main content
	inputNotFocused := !a.left.input.Focused()
	var mainView string
	switch a.sidebarActive {
	case SideSearch:
		mainView = a.left.ViewSearchContent(mainW, contentH)
	case SideDownloads:
		mainView = a.left.ViewDownloadsContent(mainW, contentH)
	case SideLogs:
		mainView = renderLogsView(a.logOffset, mainW, contentH, inputNotFocused)
	default:
		mainView = a.renderLyricsView(mainW, contentH, inputNotFocused)
	}
	contentRow := lipgloss.JoinHorizontal(lipgloss.Top, sideView, mainView)

	// Status
	status := ""
	if a.statusMsg != "" && time.Since(a.statusAt) < 5*time.Second {
		status = "  " + a.statusMsg
	}

	// Help
	help := HelpStyle.Render(
		"  tab:nav  enter:play up/down/jk:nav  n:next  p:prev  r:random  d:download  space:pause  /:search  l:lyrics lang  q:quit",
	)

	// Progress bar
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

	return b.String()
}

func (a *App) renderLyricsView(w, h int, focused bool) string {
	if a.nowPlay == nil {
		return lipgloss.NewStyle().
			Width(w - 2).
			Height(h - 2).
			Render(DimItemStyle.Render(" Play a track to see lyrics..."))
	}

	borderColor := lipgloss.Color("#7c7986")
	lyricsBox := a.right.renderLyricsBox(focused, borderColor)
	return lipgloss.NewStyle().Width(w).Render(lyricsBox)
}

func (a *App) renderProgressBar(w int) string {
	state := a.player.State()

	// Controls row (all dim #7c7986)
	playIcon := IconPlay
	if state == player.Playing {
		playIcon = IconPause
	}

	dim := ProgressDimStyle

	shuffleIcon := dim.Render(IconShuffle)
	if a.random {
		shuffleIcon = ProgressFilledStyle.Render(IconShuffle)
	}

	controls := fmt.Sprintf("  %s  %s  %s  %s ",
		dim.Render(IconPrev),
		dim.Render(playIcon),
		dim.Render(IconNext),
		shuffleIcon,
	)

	// Progress bar row
	innerW := w - 4

	elapsedSec := 0
	totalSec := 0
	if a.nowPlay != nil {
		elapsedSec = int(a.right.elapsed.Seconds())
		totalSec = a.nowPlay.Duration
		if totalSec > 0 && elapsedSec > totalSec {
			elapsedSec = totalSec
		}
	}

	timeStr := FormatDuration(elapsedSec)
	timeW := len([]rune(timeStr))

	// Bar: ██ fillW + time + ░░ emptyW (time overlaid on ░)
	restW := innerW - timeW
	if restW < 0 {
		restW = 0
	}

	fillW := 0
	if totalSec > 0 {
		fillW = restW * elapsedSec / totalSec
	}
	if fillW > restW {
		fillW = restW
	}
	emptyW := restW - fillW

	var bar strings.Builder
	bar.WriteString(ProgressFilledStyle.Render(strings.Repeat("█", fillW)))
	bar.WriteString(ProgressTimeStyle.Render(timeStr))
	bar.WriteString(ProgressEmptyStyle.Render(strings.Repeat("░", emptyW)))

	progress := bar.String()

	// Manually render box to preserve ANSI styles
	borderColor := lipgloss.Color("#7c7986")
	borderChar := lipgloss.NewStyle().Foreground(borderColor)

	// Top border with track title
	title := ""
	if a.nowPlay != nil {
		title = a.nowPlay.Title
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

func (a *App) playTrackAt(idx int, t api.Track) tea.Cmd {
	a.currentIdx = idx
	a.nowPlay = &t
	a.syncPlaylistState()

	if idx >= 0 {
		a.left.cursor = idx
		vis := a.left.visibleRows()
		if a.left.cursor < a.left.offset {
			a.left.offset = a.left.cursor
		}
		if a.left.cursor >= a.left.offset+vis {
			a.left.offset = a.left.cursor - vis + 1
		}
	}

	if strings.HasPrefix(t.ID, "local:") {
		path := strings.TrimPrefix(t.ID, "local:")
		if err := a.player.Play(path); err != nil {
			a.setStatus(StatusErrStyle.Render("X " + err.Error()))
			return nil
		}
		a.playedAt = time.Now()
		a.right.SetTrack(&t, a.playedAt)
		a.setStatus(StatusOKStyle.Render(">  " + t.Title))
		jsonPath := strings.TrimSuffix(path, ".m4a") + ".json"
		data, err := os.ReadFile(jsonPath)
		if err == nil {
			var lr api.LyricsResp
			if json.Unmarshal(data, &lr) == nil {
				a.right.SetLyrics(lr, a.playedAt)
			}
		} else {
			a.right.SetLyrics(api.LyricsResp{Plain: "(No lyrics available)"}, a.playedAt)
		}
		return nil
	}

	return func() tea.Msg {
		streamURL := a.client.StreamURL(t.ID)
		if err := a.player.Play(streamURL); err != nil {
			return StreamStartedMsg{Err: err}
		}
		// lr, lyricsErr := a.client.Lyrics(t.ID, t.Title, t.Artist, t.Duration)
		lr, lyricsErr := getCachedLyrics(a.client, t.ID, t.Title, t.Artist, t.Duration)
		return StreamStartedMsg{
			Track:     t,
			PlayedAt:  time.Now(),
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
		next = rand.Intn(len(a.playlist))
		for next == a.currentIdx && len(a.playlist) > 1 {
			next = rand.Intn(len(a.playlist))
		}
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
		prev = rand.Intn(len(a.playlist))
		for prev == a.currentIdx && len(a.playlist) > 1 {
			prev = rand.Intn(len(a.playlist))
		}
	}
	if prev < 0 {
		return nil
	}
	return a.playTrackAt(prev, a.playlist[prev])
}

func (a *App) syncPlaylistState() {
	if a.playlist != nil {
		a.right.SetPlaylistState(a.currentIdx, len(a.playlist), a.random)
	}
}

func (a *App) switchSidebar(item SidebarItem) {
	a.sidebarActive = item
	if item == SideSearch {
		a.left.searchOnEnter = true
	} else {
		a.left.searchOnEnter = false
	}
}

func (a *App) resizePanels() {
	mainW := a.width - sidebarWidth
	if mainW < 10 {
		mainW = 10
	}
	overhead := 6 + 1 + 1 + 1 // SOM(6) + sep + status + help
	contentH := a.height - overhead
	if contentH < 5 {
		contentH = 5
	}
	a.left.SetSize(mainW, contentH)
	a.right.SetSize(mainW, contentH)
}

func (a *App) setStatus(s string) {
	a.statusMsg = s
	a.statusAt = time.Now()
}

func freePort() string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "0"
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return strconv.Itoa(port)
}

func init() {
	home, _ := os.UserHomeDir()
	logPath := home + "/som_debug.log"
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic("CANNOT OPEN LOG FILE: " + err.Error())
	}
	log.SetOutput(io.MultiWriter(f, LogBuf))
}
