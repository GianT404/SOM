package ui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
)

func (r RightPanel) renderLyricsBox(focused bool, borderColor color.Color, frame int) string {
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

	content := r.renderLyrics(innerW, frame)

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
		cursor := " "
		tick := " "
		if i == r.lyrics.LanguageIndex() {
			tick = "+"
		}
		line := fmt.Sprintf(" %s [%s] %s", cursor, tick, label)

		if i == r.langCursor {
			pad := innerW - runewidth.StringWidth(line)
			if pad < 0 {
				pad = 0
			}
			b.WriteString(LyricSelectStyle.Render(line + strings.Repeat(" ", pad)))
		} else {
			b.WriteString(LyricNormalStyle.Render(line))
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

	for written < lyrH {
		b.WriteString("\n")
		written++
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func (r RightPanel) renderLyrics(innerW int, frame int) string {
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
		return strings.TrimSuffix(b.String(), "\n")
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
		return strings.TrimSuffix(b.String(), "\n")
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
				textW := runewidth.StringWidth(seg)
				padLeft := (innerW - textW) / 2
				if padLeft < 0 {
					padLeft = 0
				}
				prefix := strings.Repeat(" ", padLeft)
				style := LyricNormalStyle
				if i == r.curLine && !r.manualSelect {
					style = LyricHighlightStyle
				} else if r.manualSelect && i == r.highlightLine {
					highlightStyle := LyricSelectStyle.Copy().
						Width(innerW).
						Align(lipgloss.Center)
					b.WriteString(highlightStyle.Render(seg) + "\n")
					written++
					continue
				}
				b.WriteString(style.Render(prefix+seg) + "\n")
				written++
			}
		}
		for written < lyrH {
			b.WriteString("\n")
			written++
		}
		return strings.TrimSuffix(b.String(), "\n")
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
		return strings.TrimSuffix(b.String(), "\n")
	}

	logo := animeFrames[frame%len(animeFrames)]
	noLyr := DimItemStyle.Render("(no lyrics available)")
	block := lipgloss.JoinVertical(lipgloss.Center, logo, "", noLyr)

	blockLines := strings.Split(block, "\n")
	var final []string
	for _, l := range blockLines {
		if len(final) >= lyrH {
			break
		}
		final = append(final, l)
	}

	topPad := (lyrH - len(final)) / 2
	if topPad < 0 {
		topPad = 0
	}
	for i := 0; i < topPad; i++ {
		b.WriteString("\n")
	}
	centerStyle := lipgloss.NewStyle().Width(innerW).Align(lipgloss.Center)
	for _, line := range final {
		b.WriteString(centerStyle.Render(line) + "\n")
	}
	for i := topPad + len(final); i < lyrH; i++ {
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func (r RightPanel) lyricsHeight() int {
	h := r.height - 2
	if h < 5 {
		return 5
	}
	return h
}
