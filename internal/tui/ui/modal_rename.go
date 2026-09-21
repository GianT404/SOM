package ui

import (
	"path/filepath"
	"strings"

	"som/internal/storage"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type RenameModal struct {
	input   textinput.Model
	errStr  string
	target  *LocalFile
	plStore *storage.DB
	width   int
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
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			// Yêu cầu App đóng Modal
			return m, func() tea.Msg { return CloseModalMsg{} }

		case "enter":
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

			// Trả về 2 việc cùng lúc: Gửi lệnh đóng modal, và gửi lệnh đổi tên file
			return m, tea.Batch(
				func() tea.Msg { return CloseModalMsg{} },
				renameCmd(m.plStore, oldPath, newPath, newTitle),
			)
		}
	}

	// Xử lý gõ phím cho input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *RenameModal) View() string {
	var b strings.Builder
	b.WriteString("\n  ")
	b.WriteString(m.input.View())
	if m.errStr != "" {
		b.WriteString("\n  " + m.errStr)
	}
	b.WriteString("\n\n")
	b.WriteString(DimItemStyle.Render(" (enter: rename  | esc: cancel)"))

	w := m.input.Width() + 8
	if w < 48 {
		w = 48
	}
	if m.width > 0 && w > m.width-2 {
		w = m.width - 2
	}

	return renderBox(w, "Rename Title", b.String(), themeCol("#e8593c"))
}
