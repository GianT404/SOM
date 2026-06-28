package ui

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	"som-tui/internal/tui/api"
	"som-tui/internal/tui/player"

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
		}

	case PlayStartedMsg:
		t := msg.Track
		a.nowPlay = &t
		streamURL := a.client.StreamURL(t.ID)
		if err := a.player.Play(streamURL); err != nil {
			a.setStatus(StatusErrStyle.Render("✗ " + err.Error()))
		} else {
			a.playedAt = time.Now()
			a.right.SetTrack(&t, a.playedAt)
			a.setStatus(StatusOKStyle.Render("▶  " + t.Title))
		}
		c, id, title, dur := a.client, t.ID, t.Title, t.Duration
		cmds = append(cmds, func() tea.Msg {
			lr, err := c.Lyrics(id, title, dur)
			return LyricsLoadedMsg{Lyrics: lr, Err: err}
		})
	case PlayLocalMsg:
		t := api.Track{
			ID:       "local",
			Title:    msg.Title,
			Duration: 0,
		}
		a.nowPlay = &t

		if err := a.player.Play(msg.Path); err != nil {
			a.setStatus(StatusErrStyle.Render("✗ " + err.Error()))
		} else {
			a.playedAt = time.Now()
			a.right.SetTrack(&t, a.playedAt)
			a.setStatus(StatusOKStyle.Render("▶  " + t.Title))
		}
		jsonPath := strings.TrimSuffix(msg.Path, ".m4a") + ".json"
		data, err := os.ReadFile(jsonPath)
		if err == nil {
			var lr api.LyricsResp
			if json.Unmarshal(data, &lr) == nil {
				a.right.SetLyrics(lr, a.playedAt)
			}
		} else {
			a.right.SetLyrics(api.LyricsResp{Plain: "(No lyrics available)"}, a.playedAt)
		}
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
		return "Loading…"
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

	// Help bar
	help := HelpStyle.Render(
		"  tab:switch pane shift+tab: switch tab  enter:search/play  ↑↓/jk:nav  d:download  space:pause  /:search  q:quit",
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

// ─── Layout helpers ───────────────────────────────────────────────────────────

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
