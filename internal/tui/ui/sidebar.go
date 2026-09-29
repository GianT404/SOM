package ui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

type SidebarItem int

const (
	SideSearch SidebarItem = iota
	SideImport
	SideDownloads
	SidePlaylists
	SideQueue
	SideLyrics
	SideLogs
	sideCount
)

const sidebarWidth = 15

func (s SidebarItem) Num() string {
	switch s {
	case SideSearch:
		return "¹"
	case SideImport:
		return "²"
	case SideDownloads:
		return "³"
	case SidePlaylists:
		return "⁴"
	case SideQueue:
		return "⁵"
	case SideLyrics:
		return "⁶"
	case SideLogs:
		return "⁷"
	default:
		return ""
	}
}

func (s SidebarItem) Title() string {
	switch s {
	case SideSearch:
		return "Search"
	case SideImport:
		return "Import"
	case SideDownloads:
		return "Downloads"
	case SidePlaylists:
		return "Playlists"
	case SideQueue:
		return "Queue"
	case SideLyrics:
		return "Lyrics"
	case SideLogs:
		return "Logs"
	default:
		return ""
	}
}

func (s SidebarItem) String() string {
	return s.Num() + s.Title()
}

func RowToSidebarItem(row int) (SidebarItem, bool) {
	switch row {
	case 0:
		return SideSearch, true
	case 1:
		return SideImport, true
	case 3:
		return SideDownloads, true
	case 4:
		return SidePlaylists, true
	case 6:
		return SideQueue, true
	case 7:
		return SideLyrics, true
	case 8:
		return SideLogs, true
	default:
		return 0, false
	}
}

var (
	sidebarActiveStyle   lipgloss.Style
	sidebarInactiveStyle lipgloss.Style
	sidebarNumStyle      lipgloss.Style
)

func renderSidebar(active SidebarItem, height int, borderHeight int) string {
	var lines []string
	innerW := sidebarWidth - 4
	if innerW < 1 {
		innerW = 1
	}

	items := []SidebarItem{SideSearch, SideImport, SideDownloads, SidePlaylists, SideQueue, SideLyrics, SideLogs}

	for i, item := range items {
		if i > 0 && (item == SideDownloads || item == SideQueue) {
			lines = append(lines, "")
		}

		label := item.String()

		// Cắt bớt chữ nếu quá dài
		if lipgloss.Width(label) > innerW {
			runes := []rune(label)
			label = string(runes[:innerW])
		}

		pad := innerW - lipgloss.Width(label)
		if pad < 0 {
			pad = 0
		}

		switch {
		case item == active:
			// Match the track-list selection affordance: inactive rows reserve
			// one leading cell, while the active row uses that cell.
			lines = append(lines, sidebarActiveStyle.Render(label+strings.Repeat(" ", pad)))
		default:
			num := item.Num()
			title := item.Title()

			if lipgloss.Width(num)+lipgloss.Width(title) > innerW {
				runes := []rune(title)
				title = string(runes[:innerW-lipgloss.Width(num)])
			}

			var b strings.Builder
			b.WriteString(" ")
			b.WriteString(sidebarNumStyle.Render(num))
			b.WriteString(sidebarInactiveStyle.Render(title))
			if pad > 0 {
				b.WriteString(strings.Repeat(" ", pad))
			}
			lines = append(lines, b.String())
		}
	}

	// Bơm thêm các dòng trắng để chiều cao Box khớp với Panel bên cạnh
	targetH := height - 2
	for len(lines) < targetH {
		lines = append(lines, "")
	}

	contentStr := strings.Join(lines, "\n")
	return renderSidebarBox(sidebarWidth, "Menu", contentStr, themeCol("#7c7986"))
}
func renderSidebarBox(w int, title string, content string, borderColor color.Color) string {
	box := renderBox(w, "", content, borderColor)

	lines := strings.Split(box, "\n")
	if len(lines) == 0 {
		return box
	}

	borderChar := lipgloss.NewStyle().Foreground(borderColor)
	titleRendered := PanelTitleStyle.Foreground(borderColor).Render(title)

	prefix := "╭─"
	suffix := "╮"

	titleW := lipgloss.Width(titleRendered)
	remain := w - lipgloss.Width(prefix) - titleW - lipgloss.Width(suffix)
	if remain < 0 {
		remain = 0
	}

	lines[0] =
		borderChar.Render(prefix) +
			titleRendered +
			borderChar.Render(strings.Repeat("─", remain)+suffix)

	return strings.Join(lines, "\n")
}
