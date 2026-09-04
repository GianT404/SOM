package ui

import (
	"image/color"
	"math"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

type SidebarItem int

const (
	SideSearch SidebarItem = iota
	SideDownloads
	SideImport
	SideQueue
	SidePlaylists
	SideLyrics
	SideLogs
	sideCount
)

const sidebarWidth = 18

const sidebarGhostDuration = 120 * time.Millisecond

type sidebarAnimState struct {
	on    bool
	from  SidebarItem
	to    SidebarItem
	start time.Time
	end   time.Time
}

func (s SidebarItem) String() string {
	switch s {
	case SideSearch:
		return "Search"
	case SideDownloads:
		return "Downloads"
	case SideImport:
		return "Import"
	case SideQueue:
		return "Queue"
	case SideLyrics:
		return "Lyrics"
	case SideLogs:
		return "Logs"
	case SidePlaylists:
		return "Playlists"
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

	ghostStrongStyle = lipgloss.NewStyle().Foreground(ghostStrong)
)

func renderSidebar(active SidebarItem, anim sidebarAnimState, height int) string {
	var b strings.Builder

	items := []SidebarItem{SideSearch, SideDownloads, SideImport, SideQueue, SidePlaylists, SideLyrics, SideLogs}
	for i, item := range items {
		if i > 0 {
			b.WriteString("\n")
		}
		label := item.String()
		padding := sidebarWidth - 4 - len(label)
		if padding < 0 {
			padding = 0
		}
		switch {
		case item == active:
			// Con trỏ chính: hiện ngay ở tab mới.
			b.WriteString("  ")
			b.WriteString(sidebarActiveStyle.Render("| " + label))
			b.WriteString(strings.Repeat(" ", padding))
		default:
			if gi := ghostIntensity(item, active, anim); gi > 0 {
				b.WriteString("  ")
				b.WriteString(ghostStyle(gi).Render("| " + label))
				b.WriteString(strings.Repeat(" ", padding))
			} else {
				b.WriteString("  ")
				b.WriteString(sidebarInactiveStyle.Render("  " + label))
				b.WriteString(strings.Repeat(" ", padding))
			}
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

func ghostIntensity(item SidebarItem, active SidebarItem, anim sidebarAnimState) float64 {
	if !anim.on {
		return 0
	}
	now := time.Now()
	if !now.Before(anim.end) {
		return 0
	}

	// Vệt chỉ nằm giữa tab cũ (from, bao gồm) và tab đích (to, loại trừ —
	// tab đích vốn đã hiện con trỏ chính).
	if anim.to > anim.from {
		if item < anim.from || item >= anim.to {
			return 0
		}
	} else {
		if item > anim.from || item <= anim.to {
			return 0
		}
	}

	p := float64(now.Sub(anim.start)) / float64(anim.end.Sub(anim.start))
	if p > 1 {
		p = 1
	}
	if p < 0 {
		p = 0
	}

	span := math.Abs(float64(anim.to) - float64(anim.from))
	if span < 1 {
		span = 1
	}
	drow := math.Abs(float64(item) - float64(anim.to))

	gi := (1-p)*0.9 - 0.3*drow/span
	if gi <= 0 {
		return 0
	}
	if gi > 1 {
		gi = 1
	}
	return gi
}

func ghostStyle(gi float64) lipgloss.Style {
	switch {
	case gi >= 0.7:
		return ghostStrongStyle
	}
	return lipgloss.NewStyle().Foreground(ghostStrong).Faint(true)
}

func renderSOMLogo() string {
	purples := []color.Color{
		lipgloss.Color("#FFE8DF"),
		lipgloss.Color("#FFB9A7"),
		lipgloss.Color("#E8593C"),
		lipgloss.Color("#C84328"),
		lipgloss.Color("#9D311A"),
		lipgloss.Color("#6B1F0E"),
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
