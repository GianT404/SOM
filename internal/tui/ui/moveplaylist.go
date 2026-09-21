package ui

import (
	"fmt"
	"som/internal/storage"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-runewidth"
)

// ==========================================
// 1. CREATE PLAYLIST MODAL
// ==========================================
type MoveCreateModal struct {
	input textinput.Model
	width int
}

func NewMoveCreateModal(appWidth int) *MoveCreateModal {
	ti := textinput.New()
	ti.CharLimit = 50
	ti.Prompt = ""
	ti.Focus()
	return &MoveCreateModal{input: ti, width: appWidth}
}

func (m *MoveCreateModal) Init() tea.Cmd { return textinput.Blink }
func (m *MoveCreateModal) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return CloseModalMsg{} }
		case "enter":
			name := strings.TrimSpace(m.input.Value())
			if name != "" {
				return m, tea.Batch(
					func() tea.Msg { return CloseAllModalsMsg{} },
					func() tea.Msg { return InitMoveSessionMsg{TargetPlIdx: -1, NewPlName: name} },
				)
			}
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}
func (m *MoveCreateModal) View() string {
	var b strings.Builder
	b.WriteString("\n ")
	b.WriteString(NormalItemStyle.Render("Enter name for new playlist:"))
	b.WriteString("\n\n  ")
	b.WriteString(m.input.View())
	b.WriteString("\n\n ")
	b.WriteString(DimItemStyle.Render(" (enter: create  | esc: cancel)"))

	w := m.input.Width() + 8
	if w < 48 {
		w = 48
	}
	if m.width > 0 && w > m.width-2 {
		w = m.width - 2
	}
	return renderBox(w, "Create New Playlist", b.String(), themeCol("#e8593c"))
}

// ==========================================
// 2. PICK PLAYLIST MODAL
// ==========================================
type MovePickModal struct {
	playlists []storage.Playlist
	cursor    int
}

func NewMovePickModal(playlists []storage.Playlist) *MovePickModal {
	return &MovePickModal{playlists: playlists, cursor: 0}
}
func (m *MovePickModal) Init() tea.Cmd { return nil }
func (m *MovePickModal) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc", "q", ":":
			return m, func() tea.Msg { return CloseModalMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.playlists) - 1
			}
		case "down", "j":
			if m.cursor < len(m.playlists)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
		case "enter":
			idx := m.cursor
			return m, tea.Batch(
				func() tea.Msg { return CloseAllModalsMsg{} },
				func() tea.Msg { return InitMoveSessionMsg{TargetPlIdx: idx} },
			)
		}
	}
	return m, nil
}
func (m *MovePickModal) View() string {
	var b strings.Builder
	b.WriteString("\n ")
	b.WriteString(NormalItemStyle.Render("Select target playlist:"))
	b.WriteString("\n\n ")
	for i, pl := range m.playlists {
		line := fmt.Sprintf("  %s (%d)", pl.Name, len(pl.Tracks))
		if i == m.cursor {
			pad := 51 - runewidth.StringWidth(line)
			if pad < 0 {
				pad = 0
			}
			b.WriteString(SelectedItemStyle.Render(line + strings.Repeat(" ", pad)))
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
		b.WriteString("\n ")
	}
	b.WriteString("\n ")
	b.WriteString(DimItemStyle.Render(" (enter: select  | esc: cancel)"))
	return renderBox(56, "Select Playlist", b.String(), themeCol("#e8593c"))
}

// ==========================================
// 3. CONFIRM MOVE MODAL
// ==========================================
type MoveConfirmModal struct {
	plName string
	count  int
	cursor int
}

func NewMoveConfirmModal(plName string, count int) *MoveConfirmModal {
	return &MoveConfirmModal{plName: plName, count: count, cursor: 1}
}
func (m *MoveConfirmModal) Init() tea.Cmd { return nil }
func (m *MoveConfirmModal) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc", ":", "q":
			return m, func() tea.Msg { return CloseModalMsg{} }
		case "left", "h", "right", "l", "tab":
			m.cursor = 1 - m.cursor
		case "enter":
			if m.cursor == 1 {
				return m, tea.Batch(
					func() tea.Msg { return CloseAllModalsMsg{} },
					func() tea.Msg { return ExecuteMoveMsg{} },
				)
			}
			return m, func() tea.Msg { return CloseAllModalsMsg{} }
		}
	}
	return m, nil
}
func (m *MoveConfirmModal) View() string {
	var b strings.Builder
	b.WriteString("\n ")
	b.WriteString(DimItemStyle.Render(fmt.Sprintf("Move %d track to \"%s\"?", m.count, m.plName)))
	b.WriteString("\n\n ")
	cancelStyle, confirmStyle := NormalItemStyle, NormalItemStyle
	if m.cursor == 0 {
		cancelStyle = SelectedItemStyle
	} else {
		confirmStyle = SelectedItemStyle
	}
	b.WriteString(fmt.Sprintf("%s     %s", cancelStyle.Render("[ Cancel ]"), confirmStyle.Render("[ Confirm ]")))
	b.WriteString("\n\n")
	b.WriteString(DimItemStyle.Render(" (enter: confirm  | esc: cancel)"))
	return renderBox(60, "Confirm Move", b.String(), themeCol("#e8593c"))
}
