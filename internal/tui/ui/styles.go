package ui

import (
	"fmt"
	"strings"

	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
)

// Nerd-font icon codepoints (Material Design via nerd-fonts)
// Font Awesome (Nerd Font)
const (
	IconCheck = "\uf00c"
)

var (
	colorAccent  color.Color
	colorSubtle  color.Color
	colorSubtle2 color.Color
	colorWhite   color.Color
	colorDark    color.Color
	colorDark2   color.Color
	colorGreen   color.Color
	colorRed     color.Color
	deleteColor  color.Color
	colorYellow  color.Color
	colorBorder  color.Color
	ghostStrong  color.Color

	// ── Panel containers ────────────────────────────────────────────────────────

	PanelTitleStyle lipgloss.Style

	// ── Search input ────────────────────────────────────────────────────────────

	InputPromptStyle lipgloss.Style

	// ── Track list ──────────────────────────────────────────────────────────────

	SelectedItemStyle      lipgloss.Style
	NormalItemStyle        lipgloss.Style
	DimItemStyle           lipgloss.Style
	LocalFileStyle         lipgloss.Style
	LocalFileSelectedStyle lipgloss.Style

	// ── Lyrics ──────────────────────────────────────────────────────────────────

	LyricHighlightStyle lipgloss.Style
	LyricSelectStyle    lipgloss.Style
	LyricNormalStyle    lipgloss.Style

	// ── Status / Help ────────────────────────────────────────────────────────────

	StatusOKStyle  lipgloss.Style
	StatusErrStyle lipgloss.Style
	StatusMsgStyle lipgloss.Style
	HelpStyle      lipgloss.Style
	SubtitleStyle  lipgloss.Style

	// ── Progress bar ─────────────────────────────────────────────────────────────

	ProgressFilledStyle     lipgloss.Style
	ProgressTimeStyle       lipgloss.Style
	ProgressTimeOnFillStyle lipgloss.Style
	ProgressDimStyle        lipgloss.Style
)

// selFg màu chữ khi nằm trên nền accent (mono đảo thành đen để đọc được).
func selFg() color.Color {
	if isMono() {
		return lipgloss.Color("#000000")
	}
	return lipgloss.Color("#ffffff")
}

// rebuildTheme gán màu gốc + dựng lại toàn bộ style package-level theo theme
// hiện tại. Gọi lúc init và mỗi khi đổi theme (setTheme).
func rebuildTheme() {
	colorAccent = themeCol("#E8593C")
	colorSubtle = themeCol("#4A4A4A")
	colorWhite = themeCol("#7c7986")
	colorDark = themeCol("#fff")
	colorGreen = themeCol("#3DCFA0")
	colorRed = themeCol("#E24B4A")
	deleteColor = themeCol("#ff2a00")
	colorYellow = themeCol("#EF9F27")
	colorBorder = themeCol("#2E2E2E")
	ghostStrong = themeCol("#E8593C")
	colorSubtle2 = lipgloss.NoColor{}
	colorDark2 = lipgloss.NoColor{}

	PanelTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(colorWhite).
		Background(colorDark2).
		Padding(0, 1)

	InputPromptStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true)

	SelectedItemStyle = lipgloss.NewStyle().
		Foreground(selFg()).
		Background(colorAccent).
		Bold(true)

	NormalItemStyle = lipgloss.NewStyle().
		Foreground(colorWhite)

	DimItemStyle = lipgloss.NewStyle().
		Foreground(colorSubtle2)

	LocalFileStyle = lipgloss.NewStyle().
		Foreground(colorWhite)

	LocalFileSelectedStyle = lipgloss.NewStyle().
		Foreground(selFg()).
		Background(colorAccent).
		Bold(true)

	LyricHighlightStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true)

	LyricSelectStyle = lipgloss.NewStyle().
		Foreground(selFg()).
		Background(colorAccent).
		Bold(true)

	LyricNormalStyle = lipgloss.NewStyle().
		Foreground(colorSubtle2)

	StatusOKStyle = lipgloss.NewStyle().Foreground(colorDark)
	StatusErrStyle = lipgloss.NewStyle().Foreground(colorRed)
	StatusMsgStyle = lipgloss.NewStyle().Foreground(colorYellow)
	HelpStyle = lipgloss.NewStyle().Foreground(colorSubtle2)

	SubtitleStyle = lipgloss.NewStyle().
		Foreground(colorSubtle2).
		Italic(true)

	ProgressFilledStyle = lipgloss.NewStyle().
		Foreground(colorAccent)

	ProgressTimeStyle = lipgloss.NewStyle().
		Foreground(themeCol("#ffffff")).
		Bold(true)

	ProgressTimeOnFillStyle = lipgloss.NewStyle().
		Foreground(selFg()).
		Background(colorAccent).
		Bold(true)

	ProgressDimStyle = lipgloss.NewStyle().
		Foreground(colorWhite)

	// Sidebar (khai báo ở sidebar.go).
	sidebarActiveStyle = lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true)
	sidebarInactiveStyle = lipgloss.NewStyle().
		Foreground(colorSubtle2)
	ghostStrongStyle = lipgloss.NewStyle().Foreground(ghostStrong)
}

func init() {
	rebuildTheme()
}

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
