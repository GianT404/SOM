package render

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type ProgressBarStyles struct {
	Border     lipgloss.Style
	Title      lipgloss.Style
	Filled     lipgloss.Style
	Time       lipgloss.Style
	TimeFilled lipgloss.Style
	Controls   lipgloss.Style
}

func ProgressBar(w, elapsedSec, totalSec int, title, timeLabel string, styles ProgressBarStyles) string {
	if w < 4 {
		w = 4
	}
	if elapsedSec < 0 {
		elapsedSec = 0
	}
	if totalSec > 0 && elapsedSec > totalSec {
		elapsedSec = totalSec
	}

	innerW := w - 4
	timeW := len([]rune(timeLabel))
	timeStart := (innerW - timeW) / 2
	if timeStart < 0 {
		timeStart = 0
	}
	leftW := timeStart
	rightW := innerW - timeStart - timeW
	if rightW < 0 {
		rightW = 0
	}

	fillW := 0
	if totalSec > 0 {
		fillW = innerW * elapsedSec / totalSec
		if fillW > innerW {
			fillW = innerW
		}
	}

	leftFill := fillW
	if leftFill > leftW {
		leftFill = leftW
	}
	rightFill := fillW - leftW - timeW
	if rightFill < 0 {
		rightFill = 0
	}
	if rightFill > rightW {
		rightFill = rightW
	}

	labelFill := fillW - leftW
	if labelFill < 0 {
		labelFill = 0
	}
	if labelFill > timeW {
		labelFill = timeW
	}

	var bar strings.Builder
	bar.WriteString(styles.Filled.Render(strings.Repeat("█", leftFill)))
	bar.WriteString(strings.Repeat(" ", leftW-leftFill))
	runes := []rune(timeLabel)
	if labelFill > 0 {
		bar.WriteString(styles.TimeFilled.Render(string(runes[:labelFill])))
	}
	if labelFill < timeW {
		bar.WriteString(styles.Time.Render(string(runes[labelFill:])))
	}
	bar.WriteString(styles.Filled.Render(strings.Repeat("█", rightFill)))

	progress := bar.String()
	borderChar := styles.Border
	var top string
	if title == "" {
		top = borderChar.Render("╭" + strings.Repeat("─", w-2) + "╮")
	} else {
		titleRendered := styles.Title.Render(title)
		titleW := lipgloss.Width(titleRendered)
		prefix := borderChar.Render("╭── ")
		prefixW := lipgloss.Width(prefix)
		remain := w - prefixW - titleW - 1
		if remain < 0 {
			remain = 0
		}
		top = prefix + titleRendered + borderChar.Render(strings.Repeat("─", remain)+"╮")
	}

	bottom := borderChar.Render("╰" + strings.Repeat("─", w-2) + "╯")
	pad := innerW - lipgloss.Width(progress)
	if pad < 0 {
		pad = 0
	}

	line := borderChar.Render("│ ") +
		styles.Controls.Render("") +
		progress +
		strings.Repeat(" ", pad) +
		borderChar.Render(" │")

	return top + "\n" + line + "\n" + bottom
}
