package ui

import (
	"fmt"
	"strings"

	"som/internal/storage"

	tea "charm.land/bubbletea/v2"
)

type DeleteModal struct {
	target  *LocalFile
	plStore *storage.DB
	cursor  int
}

func NewDeleteModal(target *LocalFile, plStore *storage.DB) *DeleteModal {
	return &DeleteModal{
		target:  target,
		plStore: plStore,
		cursor:  0,
	}
}

func (m *DeleteModal) Init() tea.Cmd {
	return nil
}

func (m *DeleteModal) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc", ":", "q":
			return m, func() tea.Msg { return CloseAllModalsMsg{} }
		case "left", "h", "right", "l", "up", "down", "k", "j", "tab":
			m.cursor = 1 - m.cursor
		case "enter":
			if m.cursor == 1 {
				return m, tea.Batch(
					func() tea.Msg { return CloseAllModalsMsg{} },
					deleteCmd(m.plStore, m.target.Path, m.target.Name),
				)
			}
			return m, func() tea.Msg { return CloseAllModalsMsg{} }
		}
	}
	return m, nil
}

func (m *DeleteModal) View() string {
	var b strings.Builder
	name := "(No local track)"
	if m.target != nil {
		name = m.target.Name
	}
	b.WriteString("\n ")
	b.WriteString(DimItemStyle.Render("Delete \"" + name + "\" permanently?"))
	b.WriteString("\n\n ")

	cancelStyle := NormalItemStyle
	confirmStyle := NormalItemStyle
	if m.cursor == 0 {
		cancelStyle = SelectedItemStyle
	} else {
		confirmStyle = SelectedItemStyle.Foreground(deleteColor)
	}
	b.WriteString(fmt.Sprintf("%s     %s", cancelStyle.Render("[ Cancel ]"), confirmStyle.Render("[ Delete ]")))
	b.WriteString("\n\n")
	b.WriteString(DimItemStyle.Render(" (enter: confirm  | esc: back)"))

	return renderBox(60, "Delete Track", b.String(), themeCol("#E24B4A"))
}
