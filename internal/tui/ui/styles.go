package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
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
	colorYellow  = lipgloss.Color("#EF9F27")
	colorBorder  = lipgloss.Color("#2E2E2E")
	colorBorderF = lipgloss.Color("#E8593C")

	// ── App shell ───────────────────────────────────────────────────────────────

	AppStyle = lipgloss.NewStyle().
			Background(colorDark)

	HeaderSubStyle = lipgloss.NewStyle().
			Foreground(colorSubtle2).
			Background(colorDark).
			Italic(true)

	// ── Panel containers ────────────────────────────────────────────────────────

	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Background(colorDark2)

	PanelFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBorderF).
				Background(colorDark2)

	PanelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Background(colorDark2).
			Padding(0, 1)

	PanelTitleFocusedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorAccent).
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

	// ── Progress bar ────────────────────────────────────────────────────────────

	ProgressBarFill  = lipgloss.NewStyle().Foreground(colorAccent)
	ProgressBarEmpty = lipgloss.NewStyle().Foreground(colorSubtle)

	ProgressLabelStyle = lipgloss.NewStyle().
				Foreground(colorWhite).
				Bold(true)

	ProgressTimeStyle = lipgloss.NewStyle().
				Foreground(colorSubtle2)

	// ── Now-playing ─────────────────────────────────────────────────────────────

	NowPlayingStyle = lipgloss.NewStyle().
			Background(colorDark).
			Foreground(colorWhite).
			Bold(true).
			Padding(0, 1)

	PlayingIconStyle = lipgloss.NewStyle().Foreground(colorGreen)
	PausedIconStyle  = lipgloss.NewStyle().Foreground(colorYellow)

	// ── Status / Help ────────────────────────────────────────────────────────────

	StatusOKStyle  = lipgloss.NewStyle().Foreground(colorGreen)
	StatusErrStyle = lipgloss.NewStyle().Foreground(colorRed)
	StatusMsgStyle = lipgloss.NewStyle().Foreground(colorYellow)
	HelpStyle      = lipgloss.NewStyle().Foreground(colorSubtle2)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(colorSubtle2).
			Italic(true)
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
		wr := []rune(w)
		if len(wr) > maxW {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			for len(wr) > maxW {
				lines = append(lines, string(wr[:maxW]))
				wr = wr[maxW:]
			}
			current = string(wr)
			continue
		}
		if current == "" {
			current = w
		} else if len([]rune(current))+1+len(wr) <= maxW {
			current += " " + w
		} else {
			lines = append(lines, current)
			current = w
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func RenderProgressBar(width int, percent float64) string {
	if width <= 0 {
		return ""
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}
	filled := int(float64(width) * percent)
	empty := width - filled
	return ProgressBarFill.Render(strings.Repeat("█", filled)) +
		ProgressBarEmpty.Render(strings.Repeat("░", empty))
}
