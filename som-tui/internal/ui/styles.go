// internal/ui/styles.go
package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Accent: a warm orange that matches SOM's vibe
	accent = lipgloss.Color("#E8593C")
	subtle = lipgloss.Color("#555555")
	white  = lipgloss.Color("#FFFFFF")
	dark   = lipgloss.Color("#1C1C1C")
	green  = lipgloss.Color("#3DCFA0")
	red    = lipgloss.Color("#E24B4A")
	yellow = lipgloss.Color("#EF9F27")

	// ── Layout ──────────────────────────────────────────────────────────────

	AppStyle = lipgloss.NewStyle().
			Padding(1, 2)

	// ── Header ───────────────────────────────────────────────────────────────

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Italic(true)

	// ── Search bar ───────────────────────────────────────────────────────────

	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1).
			Width(60)

	InputPromptStyle = lipgloss.NewStyle().
				Foreground(accent).
				Bold(true)

	// ── Track list ───────────────────────────────────────────────────────────

	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(white).
				Background(accent).
				Padding(0, 1)

	NormalItemStyle = lipgloss.NewStyle().
			Foreground(white).
			Padding(0, 1)

	DimItemStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Padding(0, 1)

	// ── Now playing bar ──────────────────────────────────────────────────────

	NowPlayingBarStyle = lipgloss.NewStyle().
				Background(dark).
				Foreground(white).
				Padding(0, 2).
				Bold(true)

	PlayingIconStyle = lipgloss.NewStyle().
				Foreground(green)

	PausedIconStyle = lipgloss.NewStyle().
			Foreground(yellow)

	// ── Lyrics ───────────────────────────────────────────────────────────────

	LyricHighlightStyle = lipgloss.NewStyle().
				Foreground(accent).
				Bold(true).
				Padding(0, 2)

	LyricNormalStyle = lipgloss.NewStyle().
				Foreground(subtle).
				Padding(0, 2)

	// ── Status / notifications ───────────────────────────────────────────────

	StatusOKStyle  = lipgloss.NewStyle().Foreground(green)
	StatusErrStyle = lipgloss.NewStyle().Foreground(red)
	StatusMsgStyle = lipgloss.NewStyle().Foreground(yellow)

	// ── Help bar ─────────────────────────────────────────────────────────────

	HelpStyle = lipgloss.NewStyle().
			Foreground(subtle).
			MarginTop(1)
)

// FormatDuration converts seconds → mm:ss string.
func FormatDuration(sec int) string {
	if sec <= 0 {
		return "--:--"
	}
	m := sec / 60
	s := sec % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}
