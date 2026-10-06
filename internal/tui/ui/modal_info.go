package ui

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type InfoModal struct {
	target *LocalFile
}

func NewInfoModal(target *LocalFile) *InfoModal {
	return &InfoModal{target: target}
}

func (m *InfoModal) Init() tea.Cmd {
	return nil
}

func (m *InfoModal) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc", "enter", ":", "q":
			return m, func() tea.Msg { return CloseModalMsg{} }
		}
	}
	return m, nil
}

func (m *InfoModal) View() string {
	var b strings.Builder

	name := m.target.Name
	artist := m.target.Artist
	if artist == "" {
		artist = "-"
	}

	durStr := FormatDuration(m.target.Duration)
	var sizeStr, bitrateStr string
	var pathStr string

	if fi, err := os.Stat(m.target.Path); err == nil {
		sizeStr = formatBytes(fi.Size())
		if m.target.Duration > 0 {
			kbps := (fi.Size() * 8) / (1000 * int64(m.target.Duration))
			bitrateStr = fmt.Sprintf("~%d kbps", kbps)
		} else {
			bitrateStr = "-"
		}
		pathStr = m.target.Path
	} else {
		sizeStr = "-"
		bitrateStr = "-"
		pathStr = m.target.Path
	}

	b.WriteString("\n  " + StyleHint("Title", name))
	b.WriteString("\n  " + StyleHint("Artist", artist))
	b.WriteString("\n  " + StyleHint("Duration", durStr))
	b.WriteString("\n  " + StyleHint("Size", sizeStr))
	b.WriteString("\n  " + StyleHint("Bitrate", bitrateStr))
	b.WriteString("\n  " + StyleHint("Video ID", m.target.VideoID))
	b.WriteString("\n  " + StyleHint("Modified", formatDBTime(m.target.FileMTime)))
	b.WriteString("\n  " + StyleHint("Created", formatDBTime(m.target.CreatedAt)))
	b.WriteString("\n  " + StyleHint("Path", pathStr))

	b.WriteString("\n")

	return renderBox(64, "File Info", b.String(), themeCol("#E8593C"))
}
