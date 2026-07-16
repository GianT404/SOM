package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type SidebarItem int

const (
	SideSearch SidebarItem = iota
	SideDownloads
	SideLyrics
	SideLogs
	sideCount
)

const sidebarWidth = 22

func (s SidebarItem) String() string {
	switch s {
	case SideSearch:
		return "Search"
	case SideDownloads:
		return "Downloads"
	case SideLyrics:
		return "Lyrics"
	case SideLogs:
		return "Logs"
	default:
		return ""
	}
}

var (
	sidebarActiveStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	sidebarInactiveStyle = lipgloss.NewStyle().
				Foreground(colorSubtle2)
)

func renderSidebar(active SidebarItem, height int) string {
	var b strings.Builder

	items := []SidebarItem{SideSearch, SideDownloads, SideLyrics, SideLogs}
	for i, item := range items {
		if i > 0 {
			b.WriteString("\n")
		}
		label := item.String()
		padding := sidebarWidth - 4 - len(label)
		if padding < 0 {
			padding = 0
		}
		if item == active {
			b.WriteString("  ")
			b.WriteString(sidebarActiveStyle.Render("| " + label))
			b.WriteString(strings.Repeat(" ", padding))
		} else {
			b.WriteString("  ")
			b.WriteString(sidebarInactiveStyle.Render("  " + label))
			b.WriteString(strings.Repeat(" ", padding))
		}
	}

	remaining := height - len(items)
	if remaining < 0 {
		remaining = 0
	}
	for i := 0; i < remaining; i++ {
		b.WriteString("\n")
		b.WriteString(strings.Repeat(" ", sidebarWidth))
	}

	return b.String()
}

func renderSOMLogo() string {
	purples := []lipgloss.Color{
		"#FFE8DF",
		"#FFB9A7",
		"#E8593C",
		"#C84328",
		"#9D311A",
		"#6B1F0E",
	}
	art := []string{
		"     ███████╗   ██████╗   ███╗   ███╗",
		"     ██╔════╝  ██╔═══██╗  ████╗ ████║",
		"     ███████╗  ██║   ██║  ██╔████╔██║",
		"     ╚════██║  ██║   ██║  ██║╚██╔╝██║",
		"     ███████║  ╚██████╔╝  ██║ ╚═╝ ██║",
		"     ╚══════╝   ╚═════╝   ╚═╝     ╚═╝",
	}
	var b strings.Builder
	for i, line := range art {
		style := lipgloss.NewStyle().Foreground(purples[i]).Bold(true)
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(style.Render(line))
	}
	return b.String()
}
