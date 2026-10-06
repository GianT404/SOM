package ui

import (
	"fmt"
	"strings"

	"som/internal/storage"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type RenamePlaylistModal struct {
	input   textinput.Model
	errStr  string
	target  storage.Playlist
	plStore *storage.DB
	width   int
}

func NewRenamePlaylistModal(
	target storage.Playlist,
	plStore *storage.DB,
	appWidth int,
) *RenamePlaylistModal {
	ti := textinput.New()
	ti.CharLimit = 100
	ti.Prompt = ""

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

	return &RenamePlaylistModal{
		input:   ti,
		target:  target,
		plStore: plStore,
		width:   appWidth,
	}
}

func (m *RenamePlaylistModal) Init() tea.Cmd {
	return textinput.Blink
}

func (m *RenamePlaylistModal) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg {
				return CloseModalMsg{}
			}

		case "enter":
			newName := strings.TrimSpace(m.input.Value())
			if newName == "" {
				m.errStr = StatusErrStyle.Render("X Playlist name cannot be empty")
				return m, nil
			}

			return m, tea.Batch(
				func() tea.Msg {
					return CloseAllModalsMsg{}
				},
				renamePlaylistCmd(
					m.plStore,
					m.target.ID,
					m.target.Name,
					newName,
				),
			)
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *RenamePlaylistModal) View() string {
	var b strings.Builder

	b.WriteString("\n  ")
	b.WriteString(m.input.View())

	if m.errStr != "" {
		b.WriteString("\n  " + m.errStr)
	}

	b.WriteString("\n\n")
	b.WriteString(
		DimItemStyle.Render(" (enter: rename  | esc: cancel)"),
	)

	w := m.input.Width() + 8
	if w < 48 {
		w = 48
	}
	if m.width > 0 && w > m.width-2 {
		w = m.width - 2
	}

	return renderBox(
		w,
		"Rename Playlist",
		b.String(),
		themeCol("#e8593c"),
	)
}

func renamePlaylistCmd(
	plStore *storage.DB,
	id string,
	oldName string,
	newName string,
) tea.Cmd {
	return func() tea.Msg {
		if plStore == nil {
			return RenamePlaylistDoneMsg{
				ID:      id,
				OldName: oldName,
				NewName: newName,
				Err:     fmt.Errorf("playlist storage is not available"),
			}
		}

		if err := plStore.RenamePlaylist(id, newName); err != nil {
			return RenamePlaylistDoneMsg{
				ID:      id,
				OldName: oldName,
				NewName: newName,
				Err:     fmt.Errorf("rename playlist: %w", err),
			}
		}

		return RenamePlaylistDoneMsg{
			ID:      id,
			OldName: oldName,
			NewName: newName,
		}
	}
}
