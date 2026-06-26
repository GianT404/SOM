// internal/ui/app.go
// Root Bubble Tea model. Owns the current screen, player state, and
// delegates Update / View to sub-models.
package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/GianT404/SOM/tui/internal/api"
	"github.com/GianT404/SOM/tui/internal/player"
	tea "github.com/charmbracelet/bubbletea"
)

// tickMsg drives the lyrics highlight cursor.
type tickMsg time.Time

func tickEvery() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// ─── App model ───────────────────────────────────────────────────────────────

type App struct {
	client  *api.Client
	player  *player.Player
	width   int
	height  int
	screen  Screen
	search  SearchModel
	lyrics  LyricsModel
	nowPlay *api.Track // currently playing track (nil = nothing)
	status  string     // one-line status message
	statusT time.Time  // when the status was set
}

func NewApp(serverURL string) App {
	c := api.New(serverURL)
	p := player.New()
	return App{
		client: c,
		player: p,
		screen: ScreenSearch,
		search: NewSearchModel(c),
		lyrics: NewLyricsModel(),
	}
}

// ─── Init ─────────────────────────────────────────────────────────────────────

func (a App) Init() tea.Cmd {
	return tea.Batch(
		a.search.Init(),
		tickEvery(),
	)
}

// ─── Update ───────────────────────────────────────────────────────────────────

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.search.SetSize(msg.Width, msg.Height-6)
		a.lyrics.SetSize(msg.Width, msg.Height-6)

	case tickMsg:
		// advance lyric highlight if something is playing
		if a.player.State() == player.Playing {
			a.lyrics.Tick()
		}
		cmds = append(cmds, tickEvery())

	// ── Global keyboard ──────────────────────────────────────────────────────

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			a.player.Stop()
			return a, tea.Quit
		case " ":
			// Space = toggle pause/play
			a.player.TogglePause()
			st := "▶ Resumed"
			if a.player.State() == player.Paused {
				st = "⏸ Paused"
			}
			a.setStatus(st)
		case "s", "esc":
			// always go back to search
			a.screen = ScreenSearch
		case "l":
			if a.nowPlay != nil {
				a.screen = ScreenLyrics
			}
		}

	// ── Play a track ─────────────────────────────────────────────────────────

	case PlayStartedMsg:
		a.nowPlay = &msg.Track
		streamURL := a.client.StreamURL(msg.Track.ID)
		if err := a.player.Play(streamURL); err != nil {
			a.setStatus(StatusErrStyle.Render("▸ " + err.Error()))
		} else {
			a.setStatus(StatusOKStyle.Render("▶ Playing: " + msg.Track.Title))
		}
		// kick off lyrics fetch in background
		cmds = append(cmds, fetchLyricsCmd(a.client, msg.Track.ID))

	case PlayErrorMsg:
		a.setStatus(StatusErrStyle.Render("✗ " + msg.Err.Error()))

	// ── Lyrics loaded ────────────────────────────────────────────────────────

	case LyricsLoadedMsg:
		if msg.Err != nil {
			a.setStatus(StatusMsgStyle.Render("⚠ No lyrics found"))
			a.lyrics.SetLyrics(api.LyricsResp{Plain: "(no lyrics available)"})
		} else {
			a.lyrics.SetLyrics(msg.Lyrics)
			a.setStatus(StatusOKStyle.Render("♪ Lyrics loaded"))
		}

	// ── Download done ─────────────────────────────────────────────────────────

	case DownloadDoneMsg:
		if msg.Err != nil {
			a.setStatus(StatusErrStyle.Render("✗ Download failed: " + msg.Err.Error()))
		} else {
			a.setStatus(StatusOKStyle.Render("✓ Saved → " + msg.Path))
		}
	}

	// ── Delegate to sub-models ────────────────────────────────────────────────

	switch a.screen {
	case ScreenSearch:
		newSearch, cmd := a.search.Update(msg)
		a.search = newSearch
		cmds = append(cmds, cmd)
	case ScreenLyrics:
		newLyrics, cmd := a.lyrics.Update(msg)
		a.lyrics = newLyrics
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (a App) View() string {
	var b strings.Builder

	// Header
	b.WriteString(TitleStyle.Render("◈ SOM TUI") + "  ")
	b.WriteString(SubtitleStyle.Render("Stream music from YouTube"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", a.width))
	b.WriteString("\n")

	// Main content
	switch a.screen {
	case ScreenSearch:
		b.WriteString(a.search.View())
	case ScreenLyrics:
		b.WriteString(a.lyrics.View())
	}

	// Now-playing bar
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", a.width))
	b.WriteString("\n")
	b.WriteString(a.nowPlayingBar())

	// Status
	if a.status != "" && time.Since(a.statusT) < 4*time.Second {
		b.WriteString("\n" + a.status)
	}

	// Help
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render(
		"[↑↓] navigate  [enter] play  [d] download  [l] lyrics  [s] search  [space] pause  [q] quit",
	))

	return AppStyle.Render(b.String())
}

func (a App) nowPlayingBar() string {
	if a.nowPlay == nil {
		return NowPlayingBarStyle.Render("  ◇  Nothing playing")
	}
	icon := PlayingIconStyle.Render("▶")
	if a.player.State() == player.Paused {
		icon = PausedIconStyle.Render("⏸")
	}
	title := fmt.Sprintf(" %s  %s — %s  [%s]",
		icon,
		a.nowPlay.Artist,
		a.nowPlay.Title,
		FormatDuration(a.nowPlay.Duration),
	)
	bar := NowPlayingBarStyle.Width(a.width - 4).Render(title)
	return bar
}

func (a *App) setStatus(s string) {
	a.status = s
	a.statusT = time.Now()
}

// ─── Background commands ──────────────────────────────────────────────────────

func fetchLyricsCmd(c *api.Client, id string) tea.Cmd {
	return func() tea.Msg {
		lr, err := c.Lyrics(id)
		return LyricsLoadedMsg{Lyrics: lr, Err: err}
	}
}

func downloadCmd(c *api.Client, track api.Track, destDir string) tea.Cmd {
	return func() tea.Msg {
		path, err := c.DownloadM4A(track.ID, track.Title, destDir)
		return DownloadDoneMsg{Path: path, Err: err}
	}
}

// Expose downloadCmd for use from SearchModel
func DownloadCmd(c *api.Client, track api.Track, destDir string) tea.Cmd {
	return downloadCmd(c, track, destDir)
}
