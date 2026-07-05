package ui

import (
	"strings"
	"github.com/charmbracelet/lipgloss"
)

func renderBox(w int, title string, content string, borderColor lipgloss.TerminalColor) string {
	borderChar := lipgloss.NewStyle().Foreground(borderColor)

	// Top border
	var topBorder string
	if title == "" {
		topBorder = borderChar.Render("╭" + strings.Repeat("─", w-2) + "╮")
	} else {
		titleRendered := PanelTitleStyle.Render(title)
		titleW := lipgloss.Width(titleRendered)
		prefix := "╭── "
		prefixStyled := borderChar.Render(prefix)
		prefixW := lipgloss.Width(prefixStyled)
		remain := w - prefixW - titleW - 1
		if remain < 0 {
			remain = 0
		}
		topBorder = prefixStyled + titleRendered + borderChar.Render(strings.Repeat("─", remain)+"╮")
	}

	// Content lines: wrap each with │ ... │
	innerW := w - 4
	lines := strings.Split(content, "\n")
	var bodyLines []string
	for _, line := range lines {
		padded := lipgloss.NewStyle().Width(innerW).Render(line)
		bodyLines = append(bodyLines, borderChar.Render("│ ")+padded+borderChar.Render(" │"))
	}
	body := strings.Join(bodyLines, "\n")

	// Bottom border
	bottomBorder := borderChar.Render("╰" + strings.Repeat("─", w-2) + "╯")

	return topBorder + "\n" + body + "\n" + bottomBorder
}
