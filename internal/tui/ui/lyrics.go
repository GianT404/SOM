package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
	prefixBorderL := borderChar.Render("\u256d\u2500")
	prefixWL := lipgloss.Width(prefixBorderL)
	rightLineCount := r.width - prefixWL - titleW - 1
	if rightLineCount < 0 {
		rightLineCount = 0
	}
	topBorder := prefixBorderL +
		titleRendered +
		borderChar.Render(strings.Repeat("\u2500", rightLineCount)+"\u256e")

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
		placeholder := "Play a track to load lyrics..."
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
					rendered = LyricHighlightStyle.Render(prefix + "\u25b8 " + seg)
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

func (r RightPanel) lyricsHeight() int {
	playerTotal := playerBoxContentH + 2
	h := r.height - playerTotal - 2
	if h < 5 {
		return 5
	}
	return h
}
