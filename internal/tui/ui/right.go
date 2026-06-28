package ui

import (
	"fmt"
	"strings"
	"time"

	"som/internal/tui/api"
	"som/internal/tui/player"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const playerBoxContentH = 4

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
}

func NewRightPanel(p *player.Player) RightPanel {
	return RightPanel{player: p}
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
}

func (r *RightPanel) SetLyrics(lr api.LyricsResp, playedAt time.Time) {
	r.lyrics = lr
	r.loaded = true
	r.curLine = 0
	r.offset = 0
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
	if key, ok := msg.(tea.KeyMsg); ok {
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
	borderColor := colorBorder
	if focused {
		borderColor = colorBorderF
	}
	lyricsBox := r.renderLyricsBox(focused, borderColor)
	playerBox := r.renderPlayerBox(focused, borderColor)
	return lipgloss.JoinVertical(lipgloss.Left, lyricsBox, playerBox)
}

func (r RightPanel) renderLyricsBox(focused bool, borderColor lipgloss.TerminalColor) string {
	innerW := r.width - 4
	if innerW < 10 {
		innerW = 10
	}

	content := r.renderLyrics(innerW)

	style := PanelStyle.Copy().BorderTop(false)
	titleStyle := PanelTitleStyle

	if focused {
		style = PanelFocusedStyle.Copy().BorderTop(false)
		titleStyle = PanelTitleFocusedStyle
	}

	contentBox := style.
		Width(r.width - 2).
		Height(r.lyricsHeight() + 1).
		Render(content)
	borderChar := lipgloss.NewStyle().Foreground(borderColor)
	titleRendered := titleStyle.Render("Lyrics")
	titleW := lipgloss.Width(titleRendered)
	prefixBorderL := borderChar.Render("╭─")
	prefixWL := lipgloss.Width(prefixBorderL)
	rightLineCount := r.width - prefixWL - titleW - 1
	if rightLineCount < 0 {
		rightLineCount = 0
	}
	topBorder := prefixBorderL +
		titleRendered +
		borderChar.Render(strings.Repeat("─", rightLineCount)+"╮")

	return lipgloss.JoinVertical(lipgloss.Left, topBorder, contentBox)
}

func (r RightPanel) renderPlayerBox(focused bool, borderColor lipgloss.TerminalColor) string {
	innerW := r.width - 4
	if innerW < 10 {
		innerW = 10
	}

	content := r.renderProgress(innerW)

	var style lipgloss.Style
	if focused {
		style = PanelFocusedStyle.Copy().BorderTop(false)
	} else {
		style = PanelStyle.Copy().BorderTop(false)
	}

	contentBox := style.
		Width(r.width - 2).
		Height(playerBoxContentH).
		Render(content)

	borderChar := lipgloss.NewStyle().Foreground(borderColor)
	titleRendered := r.playerBorderTitle(r.width - 6)
	titleW := lipgloss.Width(titleRendered)
	prefixBorder := borderChar.Render("╭─")
	prefixW := lipgloss.Width(prefixBorder)
	rightLineCount := r.width - prefixW - titleW - 1
	if rightLineCount < 0 {
		rightLineCount = 0
	}
	topBorder := prefixBorder +
		titleRendered +
		borderChar.Render(strings.Repeat("─", rightLineCount)+"╮")

	return lipgloss.JoinVertical(lipgloss.Left, topBorder, contentBox)
}

func (r RightPanel) playerBorderTitle(maxW int) string {
	if r.nowPlay == nil {
		return DimItemStyle.Render(" Nothing playing ")
	}

	icon := PlayingIconStyle.Render("▶")
	if r.player.State() == player.Paused {
		icon = PausedIconStyle.Render("||")
	}

	titleMax := maxW - 4
	if titleMax < 4 {
		titleMax = 4
	}
	title := truncate(r.nowPlay.Title, titleMax)

	return fmt.Sprintf(" %s  %s ", icon, title)
}

func (r RightPanel) renderLyrics(innerW int) string {
	lyrH := r.lyricsHeight()
	var b strings.Builder

	if !r.loaded {
		pad := lyrH/2 - 1
		for i := 0; i < pad; i++ {
			b.WriteString("\n")
		}
		placeholder := "Play a track to load lyrics…"
		padLeft := (innerW - len([]rune(placeholder))) / 2
		if padLeft < 0 {
			padLeft = 0
		}
		b.WriteString(DimItemStyle.Render(strings.Repeat(" ", padLeft) + placeholder))
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
			maxTextW := innerW - 4
			if maxTextW < 10 {
				maxTextW = 10
			}
			segments := wordWrap(text, maxTextW)
			for _, seg := range segments {
				if written >= lyrH {
					break
				}
				textW := len([]rune(seg)) + 2
				padLeft := (innerW - textW) / 2
				if padLeft < 0 {
					padLeft = 0
				}
				prefix := strings.Repeat(" ", padLeft)
				var rendered string
				if i == r.curLine {
					rendered = LyricHighlightStyle.Render(prefix + "▸ " + seg)
				} else {
					rendered = LyricNormalStyle.Render(prefix + "  " + seg)
				}
				b.WriteString(rendered + "\n")
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
		plainWrapped := LyricNormalStyle.Width(innerW).Render(
			strings.ReplaceAll(r.lyrics.Plain, "\r\n", "\n"),
		)
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
			noLyr := "(no lyrics available)"
			padLeft := (innerW - len([]rune(noLyr))) / 2
			if padLeft < 0 {
				padLeft = 0
			}
			b.WriteString(DimItemStyle.Render(strings.Repeat(" ", padLeft) + noLyr))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// ─── Progress bar (redesigned) ────────────────────────────────────────────────

func (r RightPanel) renderProgress(innerW int) string {
	var b strings.Builder

	if r.nowPlay == nil {
		for i := 0; i < playerBoxContentH; i++ {
			b.WriteString("\n")
		}
		return b.String()
	}

	elapsedSec := int(r.elapsed.Seconds())
	totalSec := r.nowPlay.Duration
	if totalSec <= 0 {
		totalSec = 1
	}
	pct := float64(elapsedSec) / float64(totalSec)
	if pct > 1 {
		pct = 1
	}

	elapsedStr := FormatDuration(elapsedSec)
	totalStr := FormatDuration(totalSec)

	barW := innerW - 2
	if barW < 4 {
		barW = 4
	}
	filled := int(float64(barW) * pct)
	if filled > barW {
		filled = barW
	}
	empty := barW - filled
	bar := lipgloss.NewStyle().Foreground(colorAccent).Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#3A3A3A")).Render(strings.Repeat("█", empty))

	gap := barW - len([]rune(elapsedStr)) - len([]rune(totalStr))
	if gap < 1 {
		gap = 1
	}
	timeRow := ProgressTimeStyle.Render(elapsedStr) +
		strings.Repeat(" ", gap) +
		ProgressTimeStyle.Render(totalStr)

	b.WriteString("\n")
	b.WriteString(" " + bar + "\n")
	b.WriteString(" " + timeRow + "\n")

	return b.String()
}

func (r RightPanel) lyricsHeight() int {
	playerTotal := playerBoxContentH + 2
	h := r.height - playerTotal - 2
	if h < 5 {
		return 5
	}
	return h
}
