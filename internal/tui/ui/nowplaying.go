package ui

import (
	"strings"
	"time"

	"som/internal/tui/api"
	"som/internal/tui/player"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type RightPanel struct {
	lyrics  api.LyricsResp
	loaded  bool
	curLine int
	offset  int
	width   int
	height  int

	player    *player.Player
	nowPlay   *api.Track
	playedAt  time.Time
	elapsed   time.Duration
	pausedAt  time.Time
	pausedDur time.Duration

	playlistPos   int
	playlistTotal int
	random        bool

	loadingLyrics bool
	spinner       spinner.Model
}

func NewRightPanel(p *player.Player) RightPanel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = StatusMsgStyle
	return RightPanel{player: p, spinner: sp}
}

func (r *RightPanel) SetSize(w, h int) { r.width = w; r.height = h }

func (r *RightPanel) SetTrack(t *api.Track, playedAt time.Time) {
	r.nowPlay = t
	r.playedAt = time.Now()
	r.elapsed = 0
	r.pausedAt = time.Time{}
	r.pausedDur = 0
	r.curLine = 0
	r.offset = 0
	r.loaded = false
	r.loadingLyrics = true
}

func (r *RightPanel) SetLyrics(lr api.LyricsResp, playedAt time.Time) {
	r.lyrics = lr
	r.loaded = true
	r.loadingLyrics = false
	r.curLine = 0
	r.offset = 0
}

func (r *RightPanel) SetPlaylistState(pos, total int, random bool) {
	r.playlistPos = pos
	r.playlistTotal = total
	r.random = random
}

func (r *RightPanel) SeekBy(d time.Duration) {
	r.elapsed += d
	if r.elapsed < 0 {
		r.elapsed = 0
	}
	r.playedAt = r.playedAt.Add(-d)
}

func (r *RightPanel) TickAt(now time.Time) {
	state := r.player.State()
	if state == player.Playing {
		if !r.pausedAt.IsZero() {
			r.pausedDur += time.Since(r.pausedAt)
			r.pausedAt = time.Time{}
		}
		r.elapsed = time.Since(r.playedAt) - r.pausedDur
	} else if state == player.Paused {
		if r.pausedAt.IsZero() {
			r.pausedAt = time.Now()
		}
	}

	if !r.loaded || len(r.lyrics.Synced) == 0 {
		return
	}

	elapsed := r.elapsed.Seconds()
	best := 0
	for i, line := range r.lyrics.Synced {
		if line.Time <= elapsed {
			best = i
		}
	}

	if best != r.curLine {
		r.curLine = best
		lyrH := r.lyricsHeight()
		target := r.curLine - lyrH/2
		if target < 0 {
			target = 0
		}
		maxOff := len(r.lyrics.Synced) - lyrH
		if maxOff < 0 {
			maxOff = 0
		}
		if target > maxOff {
			target = maxOff
		}
		r.offset = target
	}
}

func (r RightPanel) Update(msg tea.Msg, focused bool) (RightPanel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "pgup", "ctrl+u":
			if r.offset > 0 {
				r.offset--
			}
		case "pgdown", "ctrl+d":
			maxOff := 0
			lyrH := r.lyricsHeight()
			if len(r.lyrics.Synced) > 0 {
				maxOff = len(r.lyrics.Synced) - lyrH
			} else if r.lyrics.Plain != "" {
				lines := strings.Split(strings.ReplaceAll(r.lyrics.Plain, "\r\n", "\n"), "\n")
				maxOff = len(lines) - lyrH
			}
			if maxOff < 0 {
				maxOff = 0
			}
			if r.offset < maxOff {
				r.offset++
			}
		}

	case spinner.TickMsg:
		if r.loadingLyrics {
			var cmd tea.Cmd
			r.spinner, cmd = r.spinner.Update(msg)
			return r, cmd
		}
	}
	return r, nil
}
