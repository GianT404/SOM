package ui

import (
	"fmt"
	"strings"

	"som/internal/tui/player"

	"github.com/charmbracelet/lipgloss"
)

const playerBoxContentH = 5

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
	prefixBorder := borderChar.Render("\u256d\u2500")
	prefixW := lipgloss.Width(prefixBorder)
	rightLineCount := r.width - prefixW - titleW - 1
	if rightLineCount < 0 {
		rightLineCount = 0
	}
	topBorder := prefixBorder +
		titleRendered +
		borderChar.Render(strings.Repeat("\u2500", rightLineCount)+"\u256e")

	return lipgloss.JoinVertical(lipgloss.Left, topBorder, contentBox)
}

func (r RightPanel) playerBorderTitle(maxW int) string {
	if r.nowPlay == nil {
		return DimItemStyle.Render(" Nothing playing ")
	}

	titleMax := maxW - 2
	if titleMax < 4 {
		titleMax = 4
	}
	title := truncate(r.nowPlay.Title, titleMax)

	return fmt.Sprintf(" %s ", title)
}

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

	elapsedStr := FormatDuration(elapsedSec)
	totalStr := FormatDuration(totalSec)
	if totalSec <= 0 {
		totalStr = "--:--"
	}

	b.WriteString("\n")

	if totalSec > 0 {
		barW := innerW - 2
		if barW < 4 {
			barW = 4
		}

		pct := float64(elapsedSec) / float64(totalSec)
		if pct > 1 {
			pct = 1
		}
		filled := int(float64(barW) * pct)
		if filled > barW {
			filled = barW
		}

		timeStr := elapsedStr + "  " + totalStr
		timeRunes := []rune(timeStr)
		timeStart := (barW - len(timeRunes)) / 2
		if timeStart < 0 {
			timeStart = 0
		}
		timeEnd := timeStart + len(timeRunes)

		var row strings.Builder

		prefixLen := timeStart
		if prefixLen > 0 {
			pf := min(filled, prefixLen)
			if pf > 0 {
				row.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#E8593C")).Render(strings.Repeat("█", pf)))
			}
			pe := prefixLen - pf
			if pe > 0 {
				row.WriteString(strings.Repeat(" ", pe))
			}
		}

		row.WriteString(ProgressTimeStyle.Render(timeStr))

		suffixLen := barW - timeEnd
		if suffixLen > 0 {
			sf := max(0, min(filled-timeEnd, suffixLen))
			if sf > 0 {
				row.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#E8593C")).Render(strings.Repeat("█", sf)))
			}
			se := suffixLen - sf
			if se > 0 {
				row.WriteString(strings.Repeat(" ", se))
			}
		}

		b.WriteString(" " + row.String() + "\n")
	} else {
		timeStr := elapsedStr + "  " + totalStr
		timeStart := (innerW - 2 - len([]rune(timeStr))) / 2
		if timeStart < 1 {
			timeStart = 1
		}
		b.WriteString(strings.Repeat(" ", timeStart) + ProgressTimeStyle.Render(timeStr) + "\n")
	}

	ctrlW := innerW - 2
	if ctrlW < 10 {
		ctrlW = 10
	}
	var ctrl strings.Builder
	ctrl.WriteString(ProgressTimeStyle.Render("\uf048"))
	ctrl.WriteString(strings.Repeat(" ", 1))
	if r.player.State() == player.Playing {
		ctrl.WriteString(ProgressTimeStyle.Render("\uf04d"))
	} else {
		ctrl.WriteString(ProgressTimeStyle.Render("\uf04b"))
	}
	ctrl.WriteString(strings.Repeat(" ", 1))
	ctrl.WriteString(ProgressTimeStyle.Render("\uf051"))
	if r.random {
		ctrl.WriteString(strings.Repeat(" ", 2))
		ctrl.WriteString(StatusOKStyle.Render("\uf074"))
	}
	ctrlStr := ctrl.String()
	padLeft := (innerW - lipgloss.Width(ctrlStr)) / 2
	if padLeft < 0 {
		padLeft = 0
	}
	b.WriteString(strings.Repeat(" ", padLeft) + ctrlStr + "\n")

	return b.String()
}
