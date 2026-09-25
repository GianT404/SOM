package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var escMenuItems = []string{
	"Settings",
	"Help",
	"Quit",
}

type EscMenuModal struct {
	cursor int
}

func NewEscMenuModal() *EscMenuModal {
	return &EscMenuModal{cursor: 0}
}
func (m *EscMenuModal) Init() tea.Cmd { return nil }
func (m *EscMenuModal) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc", "q":
			return m, func() tea.Msg { return CloseModalMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(escMenuItems)-1 {
				m.cursor++
			}
		case "enter":
			switch m.cursor {
			case 0:
				return m, func() tea.Msg { return OpenSettingsMsg{} }
			case 1:
				return m, func() tea.Msg { return OpenHelpMsg{} }
			case 2:
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m *EscMenuModal) View() string {
	boxW := 30
	innerW := boxW - 4
	var lines []string
	for i, item := range escMenuItems {
		line := "  " + item
		pad := innerW - lipgloss.Width(line)
		if pad < 0 {
			pad = 0
		}

		if i == m.cursor {
			lines = append(lines, SelectedItemStyle.Render(line+strings.Repeat(" ", pad)))
		} else {
			lines = append(lines, NormalItemStyle.Render(line+strings.Repeat(" ", pad)))
		}
	}
	body := "\n" + strings.Join(lines, "\n")
	content := body + "\n"
	return renderBox(boxW, "Menu", content, colorAccent)
}
