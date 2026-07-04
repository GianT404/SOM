package ui

import (
	"encoding/json"
	"math/rand"
	"os"
	"som/internal/tui/api"
	"som/internal/tui/player"
	"strings"
	"time"

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
}

func NewApp(serverURL string) *App {
	c := api.New(serverURL)
	p := player.New()
	return &App{
		client: c,
		player: p,
		left:   NewLeftPanel(c),
		right:  NewRightPanel(p),
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
		case "ctrl+c", "q":
			a.player.Stop()
			return a, tea.Quit

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
		}

	case SearchResultMsg:
		if msg.Err == nil {
			a.playlist = msg.Tracks
		}

	case PlayStartedMsg:
		t := msg.Track
		if len(a.left.tracks) > 0 && a.left.tab == TabSearch {
			a.playlist = a.left.tracks
		}
		idx := -1
		for i, tr := range a.playlist {
			if tr.ID == t.ID {
				idx = i
				break
			}
		}
		cmds = append(cmds, a.playTrackAt(idx, t))

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
				ID:    "local:" + lf.Path,
				Title: lf.Name,
			}
			if lf.Name == msg.Title {
				idx = i
			}
		}
		if idx < 0 {
			idx = len(a.playlist) - 1
		}
		cmds = append(cmds, a.playTrackAt(idx, a.playlist[idx]))

	case LyricsLoadedMsg:
		if msg.Err != nil {
			a.right.SetLyrics(api.LyricsResp{Plain: "(no lyrics available)"}, a.playedAt)
		} else {
			a.right.SetLyrics(msg.Lyrics, a.playedAt)
		}

	case DownloadDoneMsg:
		if msg.Err != nil {
			a.setStatus(StatusErrStyle.Render(msg.Err.Error()))
		} else {
			a.setStatus(StatusOKStyle.Render("Saved " + msg.Path))
		}
	}

	var cmd tea.Cmd
	a.left, cmd = a.left.Update(msg, true)
	cmds = append(cmds, cmd)

	a.right, _ = a.right.Update(msg, true)

	return a, tea.Batch(cmds...)
}

func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	leftW, rightW := a.panelWidths()

	leftView := a.left.View(true)
	rightView := a.right.View(true)

	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Width(leftW).Render(leftView),
		lipgloss.NewStyle().Width(rightW).Render(rightView),
	)

	status := ""
	if a.statusMsg != "" && time.Since(a.statusAt) < 5*time.Second {
		status = "  " + a.statusMsg
	}

	help := HelpStyle.Render(
		"  tab:switch pane  enter:play /:search up/down/jk:nav  n:next  p:prev  r:random  d:download  space:pause  q:quit",
	)

	var b strings.Builder
	b.WriteString(columns)
	if status != "" {
		b.WriteString("\n" + status)
	}
	b.WriteString("\n")
	b.WriteString(help)

	return b.String()
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

	streamURL := a.client.StreamURL(t.ID)
	if err := a.player.Play(streamURL); err != nil {
		a.setStatus(StatusErrStyle.Render("X " + err.Error()))
		return nil
	}
	a.playedAt = time.Now()
	a.right.SetTrack(&t, a.playedAt)
	a.setStatus(StatusOKStyle.Render(">  " + t.Title))
	return func() tea.Msg {
		lr, err := a.client.Lyrics(t.ID, t.Title, t.Artist, t.Duration)
		return LyricsLoadedMsg{Lyrics: lr, Err: err}
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

// --- Layout helpers ------------------------------------------------------------

func (a *App) panelWidths() (left, right int) {
	total := a.width
	left = int(float64(total) * 0.45)
	right = total - left
	if left < 20 {
		left = 20
	}
	if right < 20 {
		right = 20
	}
	return
}

func (a *App) resizePanels() {
	leftW, rightW := a.panelWidths()
	panelH := a.height - 3
	a.left.SetSize(leftW, panelH)
	a.right.SetSize(rightW, panelH)
}

func (a *App) setStatus(s string) {
	a.statusMsg = s
	a.statusAt = time.Now()
}
