package render

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

func Box(w int, title, content string, borderColor color.Color, titleStyle lipgloss.Style) string {
	if w < 4 {
		w = 4
	}

	borderChar := lipgloss.NewStyle().Foreground(borderColor)
	lines := strings.Split(content, "\n")

	var topBorder string
	if title == "" {
		topBorder = borderChar.Render("╭" + strings.Repeat("─", w-2) + "╮")
	} else {
		prefix := "╭── "
		prefixW := lipgloss.Width(prefix)
		maxTitleW := w - prefixW - 1

		if maxTitleW < 1 {
			title = ""
		}

		if title == "" {
			topBorder = borderChar.Render("╭" + strings.Repeat("─", w-2) + "╮")
		} else {
			titleRendered := titleStyle.Foreground(borderColor).Render(title)
			titleRendered = lipgloss.NewStyle().Inline(true).MaxWidth(maxTitleW).Render(titleRendered)

			titleW := lipgloss.Width(titleRendered)
			prefixStyled := borderChar.Render(prefix)
			remain := w - prefixW - titleW - 1
			if remain < 0 {
				remain = 0
			}

			topBorder = prefixStyled + titleRendered + borderChar.Render(strings.Repeat("─", remain)+"╮")
		}
	}

	innerW := w - 4
	var bodyLines []string
	for _, line := range lines {
		padded := lipgloss.NewStyle().MaxWidth(innerW).Render(line)
		if pad := innerW - lipgloss.Width(padded); pad > 0 {
			padded += strings.Repeat(" ", pad)
		}
		bodyLines = append(bodyLines, borderChar.Render("│ ")+padded+borderChar.Render(" │"))
	}

	body := strings.Join(bodyLines, "\n")
	bottomBorder := borderChar.Render("╰" + strings.Repeat("─", w-2) + "╯")
	return topBorder + "\n" + body + "\n" + bottomBorder
}
