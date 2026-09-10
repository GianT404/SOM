package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

var escMenuItems = []string{
	"Settings",
	"Help",
	"Quit",
}

func (a *App) renderEscMenu() string {
	boxW := 30
	innerW := boxW - 4

	var lines []string
	for i, item := range escMenuItems {
		line := "  " + item
		if i == a.escMenuCursor {
			pad := innerW - lipgloss.Width(line)
			if pad < 0 {
				pad = 0
			}
			lines = append(lines, SelectedItemStyle.Render(line+strings.Repeat(" ", pad)))
		} else {
			pad := innerW - lipgloss.Width(line)
			if pad < 0 {
				pad = 0
			}
			lines = append(lines, NormalItemStyle.Render(line+strings.Repeat(" ", pad)))
		}
	}

	body := "\n" + strings.Join(lines, "\n") + "\n"

	footer := DimItemStyle.Render(" (enter: select  | esc: close)")
	content := body + "\n" + footer

	return renderBox(boxW, "Menu", content, colorAccent)
}
