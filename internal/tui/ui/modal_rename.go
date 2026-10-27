package ui

import (
	"path/filepath"
	"strings"

	"som/internal/storage"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type RenameModal struct {
	input    textinput.Model
	errStr   string
	target   *LocalFile
	plStore  *storage.DB
	width    int
	renaming bool
}

func NewRenameModal(target *LocalFile, plStore *storage.DB, appWidth int) *RenameModal {
	ti := textinput.New()
	ti.CharLimit = 200

	iw := 60
	if appWidth > 0 && appWidth-12 < iw {
		iw = appWidth - 12
	}
	if iw < 20 {
		iw = 20
	}
	ti.SetWidth(iw)
	ti.SetValue(target.Name)
	ti.Focus()
	ti.CursorEnd()

	return &RenameModal{
		input:   ti,
		target:  target,
		plStore: plStore,
		width:   appWidth,
	}
}

func (m *RenameModal) Init() tea.Cmd {
	return textinput.Blink
}

func (m *RenameModal) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	if done, ok := msg.(RenameDoneMsg); ok {
		m.renaming = false
		if done.Err != nil {
			m.errStr = StatusErrStyle.Render("X " + done.Err.Error())
			return m, nil
		}
		return m, func() tea.Msg { return CloseAllModalsMsg{} }
	}

	if m.renaming {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			return m, nil
		}
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			// Yêu cầu App đóng Modal
			return m, func() tea.Msg { return CloseModalMsg{} }

		case "enter":
			if m.renaming {
				return m, nil
			}
			newTitle := strings.TrimSpace(m.input.Value())
			if newTitle == "" {
				m.errStr = StatusErrStyle.Render("X Title cannot be empty")
				return m, nil
			}

			oldPath := m.target.Path
			newBase := sanitizeLocalName(newTitle)
			if newBase == "" {
				newBase = m.target.VideoID
			}
			newPath := filepath.Join(filepath.Dir(oldPath), newBase+filepath.Ext(oldPath))
			m.errStr = ""
			m.renaming = true
			return m, renameCmd(m.plStore, oldPath, newPath, newTitle)
		}
	}

	// Xử lý gõ phím cho input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *RenameModal) View() string {
	w := m.input.Width() + 8
	if w < 48 {
		w = 48
	}
	if m.width > 0 && w > m.width-2 {
		w = m.width - 2
	}

	innerW := w - 4
	content := lipgloss.NewStyle().Width(innerW).Render(" " + m.input.View())
	if m.errStr != "" {
		content += "\n  " + m.errStr
	}
	return renderBox(w, "Rename Title", content, themeCol("#e8593c"))
}
