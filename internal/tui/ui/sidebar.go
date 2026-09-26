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
	SideImport
	SideDownloads
	SidePlaylists
	SideQueue
	SideLyrics
	SideLogs
	sideCount
)

const sidebarWidth = 15
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
	ghostStrongStyle     lipgloss.Style
	sidebarNumStyle      lipgloss.Style
)

func renderSidebar(active SidebarItem, anim sidebarAnimState, height int, borderHeight int) string {
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
			if gi := ghostIntensity(item, active, anim); gi > 0 {
				lines = append(lines, ghostStyle(gi).Render(label+strings.Repeat(" ", pad)))
			} else {
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
	}

	// Bơm thêm các dòng trắng để chiều cao Box khớp với Panel bên cạnh
	targetH := height - 2
	for len(lines) < targetH {
		lines = append(lines, "")
	}

	contentStr := strings.Join(lines, "\n")
	return renderSidebarBox(sidebarWidth, "Menu", contentStr, themeCol("#7c7986"))
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

func renderSidebarBox(w int, title string, content string, borderColor color.Color) string {
	box := renderBox(w, "", content, borderColor)

	lines := strings.Split(box, "\n")
	if len(lines) == 0 {
		return box
	}

	borderChar := lipgloss.NewStyle().Foreground(borderColor)
	titleRendered := PanelTitleStyle.Foreground(borderColor).Render(title)

	titleW := lipgloss.Width(titleRendered)
	remain := w - 2 - titleW

	if remain < 0 {
		remain = 0
	}

	lines[0] =
		borderChar.Render("╭─") +
			titleRendered +
			borderChar.Render(strings.Repeat("─", remain)+"╮")

	return strings.Join(lines, "\n")
}
