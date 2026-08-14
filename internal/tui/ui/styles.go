package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// Nerd-font icon codepoints (Material Design via nerd-fonts)
// Font Awesome (Nerd Font)
const (
	IconCheck = "\uf00c"
)

var (
	colorAccent  = lipgloss.Color("#E8593C")
	colorSubtle  = lipgloss.Color("#4A4A4A")
	colorSubtle2 = lipgloss.NoColor{}
	colorWhite   = lipgloss.Color("#7c7986")
	colorDark    = lipgloss.Color("#fff")
	colorDark2   = lipgloss.NoColor{}
	colorGreen   = lipgloss.Color("#3DCFA0")
	colorRed     = lipgloss.Color("#E24B4A")
	deleteColor  = lipgloss.Color("#ff2a00")
	colorYellow  = lipgloss.Color("#EF9F27")
	colorBorder  = lipgloss.Color("#2E2E2E")
	ghostStrong  = lipgloss.Color("#E8593C")

	// ── Panel containers ────────────────────────────────────────────────────────

	PanelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Background(colorDark2).
			Padding(0, 1)

	// ── Search input ────────────────────────────────────────────────────────────

	InputPromptStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	// ── Track list ──────────────────────────────────────────────────────────────

	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(colorDark).
				Background(colorAccent).
				Bold(true)

	NormalItemStyle = lipgloss.NewStyle().
			Foreground(colorWhite)

	DimItemStyle = lipgloss.NewStyle().
			Foreground(colorSubtle2)

	LocalFileStyle = lipgloss.NewStyle().
			Foreground(colorWhite)

	LocalFileSelectedStyle = lipgloss.NewStyle().
				Foreground(colorDark).
				Background(colorAccent).
				Bold(true)

	// ── Lyrics ──────────────────────────────────────────────────────────────────

	LyricHighlightStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	LyricNormalStyle = lipgloss.NewStyle().
				Foreground(colorSubtle2)

	// ── Status / Help ────────────────────────────────────────────────────────────

	StatusOKStyle  = lipgloss.NewStyle().Foreground(colorDark)
	StatusErrStyle = lipgloss.NewStyle().Foreground(colorRed)
	StatusMsgStyle = lipgloss.NewStyle().Foreground(colorYellow)
	HelpStyle      = lipgloss.NewStyle().Foreground(colorSubtle2)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(colorSubtle2).
			Italic(true)

	// ── Progress bar ─────────────────────────────────────────────────────────────

	ProgressFilledStyle = lipgloss.NewStyle().
				Foreground(colorAccent)

	ProgressTimeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffffff")).
				Bold(true)

	ProgressTimeOnFillStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffffff")).
				Background(colorAccent).
				Bold(true)

	ProgressDimStyle = lipgloss.NewStyle().
				Foreground(colorWhite)
)

// ─── Helpers ─────────────────────────────────────────────────────────────────

func FormatDuration(sec int) string {
	if sec <= 0 {
		return "--:--"
	}
	return fmt.Sprintf("%02d:%02d", sec/60, sec%60)
}

func truncate(s string, max int) string {
	if runewidth.StringWidth(s) <= max {
		return s
	}
	w := 0
	var b strings.Builder
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if w+rw > max-1 {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	b.WriteRune('…')
	return b.String()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func wordWrap(text string, maxW int) []string {
	if maxW < 1 {
		maxW = 1
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	current := ""
	for _, w := range words {
		if runewidth.StringWidth(w) > maxW {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			runes := []rune(w)
			line := ""
			lineW := 0
			for _, r := range runes {
				rw := runewidth.RuneWidth(r)
				if lineW+rw > maxW && line != "" {
					lines = append(lines, line)
					line = ""
					lineW = 0
				}
				line += string(r)
				lineW += rw
			}
			current = line
			continue
		}
		probe := w
		if current != "" {
			probe = current + " " + w
		}
		if runewidth.StringWidth(probe) <= maxW {
			current = probe
		} else {
			if current != "" {
				lines = append(lines, current)
			}
			current = w
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
