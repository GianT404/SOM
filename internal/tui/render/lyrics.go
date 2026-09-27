package render

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type LyricAnimationStyles struct {
	Highlight lipgloss.Style
	Dim       lipgloss.Style
}

func LyricAnimation(
	prev, curr string,
	innerW, innerH int,
	progress float64,
	styles LyricAnimationStyles,
	wrap func(string, int) []string,
) string {
	if innerH < 1 || innerW < 1 {
		return ""
	}

	if progress >= 1.0 || prev == "" {
		lines := wrap(curr, innerW)
		if len(lines) > innerH {
			lines = lines[:innerH]
		}

		return lipgloss.Place(
			innerW,
			innerH,
			lipgloss.Center,
			lipgloss.Center,
			styles.Highlight.
				Width(innerW).
				Align(lipgloss.Center).
				Render(strings.Join(lines, "\n")),
		)
	}

	oldLines := wrap(prev, innerW)
	newLines := wrap(curr, innerW)

	gap := 2
	offset := int(progress * float64(gap))
	cY := innerH / 2
	lines := make([]string, innerH)

	putLines := func(y int, texts []string, style lipgloss.Style) {
		for i, text := range texts {
			yy := y + i
			if yy < 0 || yy >= innerH || text == "" {
				continue
			}
			lines[yy] = style.
				Width(innerW).
				Align(lipgloss.Center).
				Render(text)
		}
	}

	oldY := cY - offset
	newY := cY + gap - offset

	oldStyle := styles.Highlight
	newStyle := styles.Dim
	if progress > 0.5 {
		oldStyle = styles.Dim
		newStyle = styles.Highlight
	}

	putLines(oldY, oldLines, oldStyle)
	putLines(newY, newLines, newStyle)

	return strings.Join(lines, "\n")
}
