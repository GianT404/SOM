package ui

import (
	"strings"
	"time"

	"som/internal/domain"
	"som/internal/tui/player"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type RightPanel struct {
	lyrics  domain.LyricsResp
	loaded  bool
	curLine int
	offset  int
	width   int
	height  int

	player  *player.Player
	nowPlay *domain.Track
	elapsed time.Duration

	playlistPos   int
	playlistTotal int
	random        bool

	loadingLyrics bool
	spinner       spinner.Model

	// showLangPopup and langCursor drive the "l"-triggered lyrics language
	// picker popup.
	showLangPopup bool
	langCursor    int
}

func NewRightPanel(p *player.Player) RightPanel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = StatusMsgStyle
	return RightPanel{player: p, spinner: sp}
}

func (r *RightPanel) SetSize(w, h int) { r.width = w; r.height = h }

func (r *RightPanel) SetTrack(t *domain.Track) {
	r.nowPlay = t
	r.elapsed = 0
	r.curLine = 0
	r.offset = 0
	r.loaded = false
	r.loadingLyrics = true
	r.showLangPopup = false
	r.langCursor = 0
}

func (r *RightPanel) SetLyrics(lr domain.LyricsResp) {
	r.lyrics = lr
	r.loaded = true
	r.loadingLyrics = false
	r.curLine = 0
	r.offset = 0
	r.langCursor = lr.LanguageIndex()
}

func (r *RightPanel) SetPlaylistState(pos, total int, random bool) {
	r.playlistPos = pos
	r.playlistTotal = total
	r.random = random
}

// TickAt đồng bộ vị trí lyrics theo đúng vị trí player
func (r *RightPanel) TickAt() {
	r.elapsed = r.player.Position()

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
		if focused && r.showLangPopup {
			switch msg.String() {
			case "up", "k":
				if r.langCursor > 0 {
					r.langCursor--
				}
			case "down", "j":
				if r.langCursor < len(r.lyrics.AllTracks)-1 {
					r.langCursor++
				}
			case "enter":
				r.lyrics.SelectLanguage(r.langCursor)
				r.curLine = 0
				r.offset = 0
				r.showLangPopup = false
			case "l", "esc":
				r.showLangPopup = false
			}
			return r, nil
		}

		switch msg.String() {
		case "l":
			if focused && len(r.lyrics.AllTracks) > 0 {
				r.langCursor = r.lyrics.LanguageIndex()
				r.showLangPopup = true
			}
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
