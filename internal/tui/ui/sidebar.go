package ui

import (
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

const sidebarWidth = 19

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
		return sidebarNumStyle.Render("¹") + "Search"
	case SideDownloads:
		return sidebarNumStyle.Render("²") + "Downloads"
	case SideImport:
		return sidebarNumStyle.Render("³") + "Import"
	case SideQueue:
		return sidebarNumStyle.Render("⁴") + "Queue"
	case SidePlaylists:
		return sidebarNumStyle.Render("⁵") + "Playlists"
	case SideLyrics:
		return sidebarNumStyle.Render("⁶") + "Lyrics"
	case SideLogs:
		return sidebarNumStyle.Render("⁷") + "Logs"
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
	sidebarNumStyle  = lipgloss.NewStyle().Foreground(colorAccent)
)

func renderSidebar(active SidebarItem, anim sidebarAnimState, height int) string {
	var b strings.Builder

	items := []SidebarItem{SideSearch, SideDownloads, SideImport, SideQueue, SidePlaylists, SideLyrics, SideLogs}
	for i, item := range items {
		if i > 0 {
			b.WriteString("\n")
		}
		label := item.String()
		padding := sidebarWidth - 4 - lipgloss.Width(label)
		if padding < 0 {
			padding = 0
		}
		switch {
		case item == active:
			b.WriteString(" ")
			b.WriteString(sidebarActiveStyle.Render("| " + label))
			b.WriteString(strings.Repeat(" ", padding+1))

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
