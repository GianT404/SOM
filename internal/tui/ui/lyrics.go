package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (r RightPanel) renderLyricsBox(focused bool, borderColor lipgloss.TerminalColor) string {
	if focused {
		borderColor = lipgloss.Color("#e8593c")
	}
	innerW := r.width - 4
	if innerW < 10 {
		innerW = 10
	}

	if r.showLangPopup {
		return renderBox(r.width, "Select Language", r.renderLangPopup(innerW), borderColor)
	}

	content := r.renderLyrics(innerW)

	if content == "" {
		return renderBox(r.width, "Lyrics", "\n", borderColor)
	}
	return renderBox(r.width, "Lyrics", content, borderColor)
}

// languageNames maps common language/subtitle codes to friendly display
// names for the language picker popup.
var languageNames = map[string]string{
	"lrclib":  "Synced Lyrics (LRCLib)",
	"en":      "English",
	"vi":      "Tiếng Việt",
	"ja":      "日本語 (Japanese)",
	"ko":      "한국어 (Korean)",
	"zh":      "中文 (Chinese)",
	"zh-Hans": "中文简体 (Chinese Simplified)",
	"zh-Hant": "中文繁體 (Chinese Traditional)",
	"fr":      "Français",
	"es":      "Español",
	"de":      "Deutsch",
	"th":      "ไทย (Thai)",
	"ru":      "Русский",
	"pt":      "Português",
	"id":      "Bahasa Indonesia",
}

func languageLabel(code string) string {
	if code == "" {
		return "Unknown"
	}
	if name, ok := languageNames[code]; ok {
		return name
	}
	return strings.ToUpper(code)
}

func (r RightPanel) renderLangPopup(innerW int) string {
	lyrH := r.lyricsHeight()
	var b strings.Builder

	for i, t := range r.lyrics.AllTracks {
		label := languageLabel(t.Language)
		marker := "  "
		if i == r.lyrics.LanguageIndex() {
			marker = " \u2713"
		}
		line := marker + " " + label
		if i == r.langCursor {
			b.WriteString(LyricHighlightStyle.Render("\u25b8 " + line))
		} else {
			b.WriteString(LyricNormalStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}

	written := len(r.lyrics.AllTracks)
	if written == 0 {
		hint := "(no languages available)"
		b.WriteString(DimItemStyle.Render(hint))
		written++
	}
	b.WriteString("\n")
	written++
	b.WriteString(DimItemStyle.Render("  up/down: choose   enter: select   l/esc: cancel"))
	written++

	for written < lyrH {
		b.WriteString("\n")
		written++
	}
	return b.String()
}

func (r RightPanel) renderLyrics(innerW int) string {
	lyrH := r.lyricsHeight()
	var b strings.Builder

	if r.loadingLyrics {
		pad := lyrH/2 - 1
		for i := 0; i < pad; i++ {
			b.WriteString("\n")
		}
		loading := r.spinner.View() + " Loading lyrics..."
		padLeft := (innerW - lipgloss.Width(DimItemStyle.Render(loading))) / 2
		if padLeft < 0 {
			padLeft = 0
		}
		b.WriteString(DimItemStyle.Render(strings.Repeat(" ", padLeft) + loading))
		b.WriteString("\n")
		for i := pad + 1; i < lyrH; i++ {
			b.WriteString("\n")
		}
		return b.String()
	}

	if !r.loaded {
		pad := lyrH/2 - 1
		for i := 0; i < pad; i++ {
			b.WriteString("\n")
		}
		placeholder := "Play a track to load lyrics..."
		padLeft := (innerW - len([]rune(placeholder))) / 2
		if padLeft < 0 {
			padLeft = 0
		}
		b.WriteString(DimItemStyle.Render(strings.Repeat(" ", padLeft) + placeholder))
		b.WriteString("\n")
		for i := pad + 1; i < lyrH; i++ {
			b.WriteString("\n")
		}
		return b.String()
	}

	if len(r.lyrics.Synced) > 0 {
		written := 0
		for i := r.offset; i < len(r.lyrics.Synced) && written < lyrH; i++ {
			text := r.lyrics.Synced[i].Text
			if text == "" {
				b.WriteString("\n")
				written++
				continue
			}
			maxTextW := innerW - 4
			if maxTextW < 10 {
				maxTextW = 10
			}
			segments := wordWrap(text, maxTextW)
			for _, seg := range segments {
				if written >= lyrH {
					break
				}
				textW := len([]rune(seg)) + 2
				padLeft := (innerW - textW) / 2
				if padLeft < 0 {
					padLeft = 0
				}
				prefix := strings.Repeat(" ", padLeft)
				var rendered string
				if i == r.curLine {
					rendered = LyricHighlightStyle.Render(prefix + "\u25b8 " + seg)
				} else {
					rendered = LyricNormalStyle.Render(prefix + "  " + seg)
				}
				b.WriteString(rendered + "\n")
				written++
			}
		}
		for written < lyrH {
			b.WriteString("\n")
			written++
		}
		return b.String()
	}

	if r.lyrics.Plain != "" {
		plainWrapped := LyricNormalStyle.Width(innerW).Render(
			strings.ReplaceAll(r.lyrics.Plain, "\r\n", "\n"),
		)
		lines := strings.Split(plainWrapped, "\n")
		written := 0
		end := r.offset + lyrH
		if end > len(lines) {
			end = len(lines)
		}
		for _, line := range lines[r.offset:end] {
			b.WriteString(line + "\n")
			written++
		}
		for written < lyrH {
			b.WriteString("\n")
			written++
		}
		return b.String()
	}

	for i := 0; i < lyrH; i++ {
		if i == lyrH/2 {
			noLyr := "(no lyrics available)"
			padLeft := (innerW - len([]rune(noLyr))) / 2
			if padLeft < 0 {
				padLeft = 0
			}
			b.WriteString(DimItemStyle.Render(strings.Repeat(" ", padLeft) + noLyr))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (r RightPanel) lyricsHeight() int {
	playerTotal := 7
	h := r.height - playerTotal - 2
	if h < 5 {
		return 5
	}
	return h
}
