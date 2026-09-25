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

func (s SidebarItem) Num() string {
	switch s {
	case SideSearch:
		return "¹ "
	case SideDownloads:
		return "² "
	case SideImport:
		return "³ "
	case SideQueue:
		return "⁴ "
	case SidePlaylists:
		return "⁵ "
	case SideLyrics:
		return "⁶ "
	case SideLogs:
		return "⁷ "
	default:
		return ""
	}
}

func (s SidebarItem) Title() string {
	switch s {
	case SideSearch:
		return "Search"
	case SideDownloads:
		return "Downloads"
	case SideImport:
		return "Import"
	case SideQueue:
		return "Queue"
	case SidePlaylists:
		return "Playlists"
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

var (
	sidebarActiveStyle   lipgloss.Style
	sidebarInactiveStyle lipgloss.Style
	ghostStrongStyle     lipgloss.Style
	sidebarNumStyle      lipgloss.Style
)

func renderSidebar(active SidebarItem, anim sidebarAnimState, height int, borderHeight int) string {
	var b strings.Builder
	borderStyle := lipgloss.NewStyle().Foreground(colorBorder)

	items := []SidebarItem{SideSearch, SideDownloads, SideImport, SideQueue, SidePlaylists, SideLyrics, SideLogs}

	currentRow := 0

	for i, item := range items {
		if i > 0 {
			b.WriteString("\n")
		}
		currentRow++

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
				b.WriteString(sidebarInactiveStyle.Render("  "))
				b.WriteString(sidebarNumStyle.Render(item.Num()))
				b.WriteString(sidebarInactiveStyle.Render(item.Title()))
				b.WriteString(strings.Repeat(" ", padding))
			}
		}

		if currentRow <= borderHeight {
			b.WriteString(borderStyle.Render("│"))
		} else {
			b.WriteString(" ")
		}
	}

	remaining := height - len(items)
	if remaining < 0 {
		remaining = 0
	}

	for i := 0; i < remaining; i++ {
		b.WriteString("\n")
		currentRow++

		b.WriteString(strings.Repeat(" ", sidebarWidth))

		if currentRow <= borderHeight {
			b.WriteString(borderStyle.Render("│"))
		} else {
			b.WriteString(" ")
		}
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
