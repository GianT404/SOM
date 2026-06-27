// internal/tui/ui/right.go
package ui

import (
	"fmt"
	"strings"
	"time"

	"som-tui/internal/tui/api"
	"som-tui/internal/tui/player"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const progressBarHeight = 4

type RightPanel struct {
	lyrics  api.LyricsResp
	loaded  bool
	curLine int
	offset  int
	width   int
	height  int

	player   *player.Player
	nowPlay  *api.Track
	playedAt time.Time
	elapsed  time.Duration
}

func NewRightPanel(p *player.Player) RightPanel {
	return RightPanel{player: p}
}

func (r *RightPanel) SetSize(w, h int) { r.width = w; r.height = h }

func (r *RightPanel) SetTrack(t *api.Track, playedAt time.Time) {
	r.nowPlay = t
	r.playedAt = time.Now()
	r.elapsed = 0
	r.curLine = 0
	r.offset = 0
}

func (r *RightPanel) SetLyrics(lr api.LyricsResp, playedAt time.Time) {
	r.lyrics = lr
	r.loaded = true
	r.curLine = 0
	r.offset = 0
}

func (r *RightPanel) TickAt(now time.Time) {
	if r.player.State() == player.Playing {
		r.elapsed = time.Since(r.playedAt)
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
	if key, ok := msg.(tea.KeyMsg); ok && focused {
		switch key.String() {
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
	}
	return r, nil
}

func (r RightPanel) View(focused bool) string {
	innerW := r.width - 4
	if innerW < 10 {
		innerW = 10
	}

	var b strings.Builder

	b.WriteString(r.renderLyrics(innerW))

	b.WriteString(DimItemStyle.Render(strings.Repeat("─", innerW)))
	b.WriteString("\n")

	b.WriteString(r.renderProgress(innerW))

	style := PanelStyle.Copy().BorderTop(false)
	borderColor := colorBorder
	titleStyle := PanelTitleStyle

	if focused {
		style = PanelFocusedStyle.Copy().BorderTop(false)
		borderColor = colorBorderF
		titleStyle = PanelTitleFocusedStyle
	}

	contentBox := style.
		Width(r.width - 2).
		Height(r.height - 2).
		Render(b.String())

	borderChar := lipgloss.NewStyle().Foreground(borderColor)
	titleText := "Lyrics"
	titleRendered := titleStyle.Render(titleText)
	titleW := lipgloss.Width(titleRendered)

	leftLineCount := 1
	rightLineCount := r.width - titleW - leftLineCount - 2

	if rightLineCount < 0 {
		rightLineCount = 0
	}

	topBorder := borderChar.Render("╭"+strings.Repeat("─", leftLineCount)) +
		titleRendered +
		borderChar.Render(strings.Repeat("─", rightLineCount)+"╮")

	return lipgloss.JoinVertical(lipgloss.Left, topBorder, contentBox)
}

func (r RightPanel) renderLyrics(innerW int) string {
	lyrH := r.lyricsHeight()
	var b strings.Builder

	if !r.loaded {
		pad := lyrH/2 - 1
		for i := 0; i < pad; i++ {
			b.WriteString("\n")
		}
		b.WriteString(DimItemStyle.Render("Play a track to load lyrics…"))
		b.WriteString("\n")
		for i := pad + 1; i < lyrH; i++ {
			b.WriteString("\n")
		}
		return b.String()
	}

	if len(r.lyrics.Synced) > 0 {
		written := 0

		for i := r.offset; i < len(r.lyrics.Synced) && written < lyrH; i++ {
			text := r.lyrics.Synced[i].Text
			if text == "" {
				b.WriteString("\n")
				written++
				continue
			}

			var rendered string
			if i == r.curLine {
				rendered = LyricHighlightStyle.Width(innerW).Render("▸ " + text)
			} else {
				rendered = LyricNormalStyle.Width(innerW).Render("  " + text)
			}

			subLines := strings.Split(rendered, "\n")
			for _, sl := range subLines {
				if written >= lyrH {
					break
				}
				b.WriteString(sl + "\n")
				written++
			}
		}

		for written < lyrH {
			b.WriteString("\n")
			written++
		}
		return b.String()
	}

	if r.lyrics.Plain != "" {
		plainWrapped := LyricNormalStyle.Width(innerW).Render(strings.ReplaceAll(r.lyrics.Plain, "\r\n", "\n"))
		lines := strings.Split(plainWrapped, "\n")

		written := 0
		end := r.offset + lyrH
		if end > len(lines) {
			end = len(lines)
		}
		for _, line := range lines[r.offset:end] {
			b.WriteString(line + "\n")
			written++
		}
		for written < lyrH {
			b.WriteString("\n")
			written++
		}
		return b.String()
	}
	for i := 0; i < lyrH; i++ {
		if i == lyrH/2 {
			b.WriteString(DimItemStyle.Render("  (no lyrics available)"))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (r RightPanel) renderProgress(innerW int) string {
	var b strings.Builder

	if r.nowPlay == nil {
		b.WriteString(DimItemStyle.Render("Nothing playing"))
		b.WriteString("\n\n")
		return b.String()
	}

	icon := PlayingIconStyle.Render("▶")
	if r.player.State() == player.Paused {
		icon = PausedIconStyle.Render("⏸")
	}
	trackInfo := fmt.Sprintf(" %s  %s — %s",
		icon,
		truncate(r.nowPlay.Artist, 20),
		truncate(r.nowPlay.Title, innerW-32),
	)
	b.WriteString(ProgressLabelStyle.Width(innerW).Render(trackInfo))
	b.WriteString("\n")

	elapsedSec := int(r.elapsed.Seconds())
	totalSec := r.nowPlay.Duration
	if totalSec <= 0 {
		totalSec = 1
	}
	pct := float64(elapsedSec) / float64(totalSec)
	if pct > 1 {
		pct = 1
	}

	timeLeft := fmt.Sprintf(" %s / %s ", FormatDuration(elapsedSec), FormatDuration(totalSec))
	barW := innerW - len([]rune(timeLeft)) - 1
	if barW < 4 {
		barW = 4
	}

	b.WriteString(" ")
	b.WriteString(RenderProgressBar(barW, pct))
	b.WriteString(ProgressTimeStyle.Render(timeLeft))
	b.WriteString("\n")

	return b.String()
}

func (r RightPanel) lyricsHeight() int {
	h := r.height - 2 - 1 - progressBarHeight
	if h < 3 {
		return 3
	}
	return h
}
