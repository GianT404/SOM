package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type VersionModal struct {
	version string
}

func NewVersionModal(version string) *VersionModal {
	if strings.TrimSpace(version) == "" {
		version = "dev"
	}
	return &VersionModal{version: version}
}

func (m *VersionModal) Init() tea.Cmd {
	return nil
}

func (m *VersionModal) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc", "enter", "q":
			return m, func() tea.Msg {
				return CloseModalMsg{}
			}
		}
	}
	return m, nil
}

func (m *VersionModal) View() string {
	madeByStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF"))

	return strings.Join([]string{
		styleHint("Version", m.version),
		madeByStyle.Render("Made by ミＧＩＡＮ4０４シ"),
	}, "\n")
}
