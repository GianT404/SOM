package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (p LeftPanel) renderSearchList(innerW int) string {
	if !p.searched {
		return SubtitleStyle.Render(" Type to search…") + "\n"
	}
	if len(p.tracks) == 0 {
		return DimItemStyle.Render(" No results.") + "\n"
	}
	var b strings.Builder
	vis := p.visibleRows()
	end := p.offset + vis
	if end > len(p.tracks) {
		end = len(p.tracks)
	}
	titleW := innerW - 12
	if titleW < 10 {
		titleW = 10
	}

	for i := p.offset; i < end; i++ {
		t := p.tracks[i]
		mark := "  "
		if i == p.cursor {
			mark = ""
		}
		safeTitle := truncate(t.Title, titleW)
		titleBlock := lipgloss.NewStyle().Width(titleW).Render(safeTitle)
		durationBlock := FormatDuration(t.Duration)

		line := mark + titleBlock + " " + durationBlock

		if i == p.cursor {
			b.WriteString(SelectedItemStyle.Width(innerW).Render(line))
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}
	if len(p.tracks) > vis {
		b.WriteString(DimItemStyle.Render(fmt.Sprintf(" %d/%d", p.cursor+1, len(p.tracks))))
		b.WriteString("\n")
	}
	return b.String()
}

func (p LeftPanel) renderLocalList(innerW int) string {
	locals := p.getFilteredLocals()
	if len(locals) == 0 {
		if strings.TrimSpace(p.input.Value()) != "" {
			return DimItemStyle.Render(" No matching downloaded files found.") + "\n"
		}
		return DimItemStyle.Render(" No downloaded files in ~/Music/SOM_Downloads/") + "\n"
	}
	var b strings.Builder
	vis := p.visibleRows()
	end := p.offset + vis
	if end > len(locals) {
		end = len(locals)
	}
	for i := p.offset; i < end; i++ {
		f := locals[i]
		mark := "  "
		if i == p.cursor {
			mark = ""
		}
		line := mark + truncate(f.Name, innerW-4)
		if i == p.cursor {
			b.WriteString(LocalFileSelectedStyle.Width(innerW).Render(line))
		} else {
			b.WriteString(LocalFileStyle.Render(line))
		}
		b.WriteString("\n")
	}
	if len(locals) > vis {
		b.WriteString(DimItemStyle.Render(fmt.Sprintf(" %d/%d", p.cursor+1, len(locals))))
		b.WriteString("\n")
	}
	return b.String()
}

func (p LeftPanel) renderPanel(content string, focused bool) string {
	style := PanelStyle
	if focused {
		style = PanelFocusedStyle
	}
	return style.
		Width(p.width - 2).
		Height(p.height - 2).
		Render(content)
}

func (p LeftPanel) itemCount() int {
	if p.tab == TabSearch {
		return len(p.tracks)
	}
	return len(p.getFilteredLocals())
}

func (p LeftPanel) visibleRows() int {
	rows := p.height - 10
	if rows < 3 {
		return 3
	}
	return rows
}

func (p LeftPanel) getFilteredLocals() []LocalFile {
	if p.tab != TabDownloads {
		return p.locals
	}
	q := strings.ToLower(strings.TrimSpace(p.input.Value()))
	if q == "" {
		return p.locals
	}
	var filtered []LocalFile
	for _, f := range p.locals {
		if strings.Contains(strings.ToLower(f.Name), q) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}
