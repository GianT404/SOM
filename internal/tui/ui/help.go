package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type helpBind [2]string

type helpSection struct {
	title string
	binds []helpBind
}

var helpSections = []helpSection{
	{
		title: "Global & Navigation",
		binds: []helpBind{
			{"?", "Toggle Help"},
			{"tab", "Switch Tab"},
			{"1-5", "Jump to Tab"},
			{"/", "Search"},
			{":", "Command Palette"},
			{"q", "Quit"},
		},
	},
	{
		title: "Player",
		binds: []helpBind{
			{"enter", "Play Selected"},
			{"space", "Play / Pause"},
			{"] / [", "Next / Prev"},
			{"r", "Toggle Random"},
			{"←/→", "Seek +/- 5s"},
		},
	},
	{
		title: "Playlists & Actions",
		binds: []helpBind{
			{"a", "Add to Playlist"},
			{"/", "Create Playlist"},
			{"delete", "Remove Playlist"},
			{"d", "Download"},
			{"l", "Lyrics Language"},
		},
	},
	{
		title: "Flags",
		binds: []helpBind{
			{"--version", "Show version"},
			{"--upgrade", "True to it's name."},
		},
	},
}

const helpMaxColumns = 3

func renderHelpSection(sec helpSection, keyWidth int) string {
	headerStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(colorWhite)
	dividerStyle := lipgloss.NewStyle().Foreground(colorBorder)

	var b strings.Builder
	b.WriteString(headerStyle.Render(sec.title) + "\n")
	b.WriteString(dividerStyle.Render(strings.Repeat("─", keyWidth+16)) + "\n")

	for _, bind := range sec.binds {
		key := fmt.Sprintf("%*s", keyWidth, bind[0])
		b.WriteString(keyStyle.Render(key) + descStyle.Render("  "+bind[1]) + "\n")
	}
	return b.String()
}

func maxKeyWidth(secs []helpSection) int {
	w := 0
	for _, sec := range secs {
		for _, b := range sec.binds {
			if l := len([]rune(b[0])); l > w {
				w = l
			}
		}
	}
	return w
}

func splitHelpColumns(secs []helpSection, numCols int) [][]helpSection {
	cols := make([][]helpSection, numCols)
	heights := make([]int, numCols)

	for _, sec := range secs {
		shortest := 0
		for i := 1; i < numCols; i++ {
			if heights[i] < heights[shortest] {
				shortest = i
			}
		}
		cols[shortest] = append(cols[shortest], sec)
		heights[shortest] += len(sec.binds) + 3
	}
	return cols
}

func (a *App) renderHelpPopup() string {
	numCols := len(helpSections)
	if numCols > helpMaxColumns {
		numCols = helpMaxColumns
	}
	if numCols < 1 {
		numCols = 1
	}

	columns := splitHelpColumns(helpSections, numCols)

	renderedCols := make([]string, 0, numCols)
	for _, col := range columns {
		if len(col) == 0 {
			continue
		}
		keyW := maxKeyWidth(col)

		parts := make([]string, 0, len(col))
		for _, sec := range col {
			parts = append(parts, renderHelpSection(sec, keyW))
		}
		renderedCols = append(renderedCols, strings.Join(parts, "\n"))
	}

	joinArgs := make([]string, 0, len(renderedCols)*2-1)
	for i, col := range renderedCols {
		if i > 0 {
			joinArgs = append(joinArgs, "    ")
		}
		joinArgs = append(joinArgs, col)
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, joinArgs...)
	bodyW := lipgloss.Width(body)

	footer := lipgloss.NewStyle().
		Width(bodyW).
		Align(lipgloss.Center).
		Render(DimItemStyle.Render("Press ? or Esc to close"))

	content := "\n" + body + "\n\n" + footer

	return renderBox(bodyW+6, "Keyboard Shortcuts", content, lipgloss.Color("#e8593c"))
}
