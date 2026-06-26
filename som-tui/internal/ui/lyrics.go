// internal/ui/lyrics.go
// Lyrics screen.
// – Shows synced lyrics (LRC) if available; highlights the current line
//
//	by interpolating elapsed time from when playback started.
//
// – Falls back to plain text if no synced lines exist.
// – Scrolls automatically to keep the highlighted line centered.
package ui

import (
	"strings"
	"time"

	"github.com/GianT404/SOM/tui/internal/api"
	tea "github.com/charmbracelet/bubbletea"
)

type LyricsModel struct {
	lyrics    api.LyricsResp
	startedAt time.Time // when playback started (approximate)
	curLine   int
	offset    int
	width     int
	height    int
	loaded    bool
}

func NewLyricsModel() LyricsModel {
	return LyricsModel{startedAt: time.Now()}
}

func (m *LyricsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *LyricsModel) SetLyrics(lr api.LyricsResp) {
	m.lyrics = lr
	m.startedAt = time.Now()
	m.curLine = 0
	m.offset = 0
	m.loaded = true
}

// Tick is called every 500 ms; advances the current lyric line pointer.
func (m *LyricsModel) Tick() {
	if !m.loaded || len(m.lyrics.Synced) == 0 {
		return
	}
	elapsed := time.Since(m.startedAt).Seconds()
	for i, line := range m.lyrics.Synced {
		if line.Time <= elapsed {
			m.curLine = i
		}
	}
	// auto-scroll: keep curLine near the center
	visible := m.visibleRows()
	target := m.curLine - visible/2
	if target < 0 {
		target = 0
	}
	m.offset = target
}

// ─── Init / Update / View ─────────────────────────────────────────────────────

func (m LyricsModel) Init() tea.Cmd { return nil }

func (m LyricsModel) Update(msg tea.Msg) (LyricsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.offset > 0 {
				m.offset--
			}
		case "down", "j":
			if m.offset < len(m.lyrics.Synced)-1 {
				m.offset++
			}
		}
	}
	return m, nil
}

func (m LyricsModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(TitleStyle.Render("♪ Lyrics"))
	b.WriteString("\n\n")

	if !m.loaded {
		b.WriteString(SubtitleStyle.Render("Loading lyrics…"))
		return b.String()
	}

	// Synced lyrics
	if len(m.lyrics.Synced) > 0 {
		visible := m.visibleRows()
		end := m.offset + visible
		if end > len(m.lyrics.Synced) {
			end = len(m.lyrics.Synced)
		}
		for i := m.offset; i < end; i++ {
			text := m.lyrics.Synced[i].Text
			if i == m.curLine {
				b.WriteString(LyricHighlightStyle.Width(m.width - 4).Render("▸ " + text))
			} else {
				b.WriteString(LyricNormalStyle.Width(m.width - 4).Render("  " + text))
			}
			b.WriteString("\n")
		}
		return b.String()
	}

	// Plain lyrics fallback
	if m.lyrics.Plain != "" {
		lines := strings.Split(m.lyrics.Plain, "\n")
		visible := m.visibleRows()
		end := m.offset + visible
		if end > len(lines) {
			end = len(lines)
		}
		for _, line := range lines[m.offset:end] {
			b.WriteString(LyricNormalStyle.Render(line))
			b.WriteString("\n")
		}
		return b.String()
	}

	b.WriteString(SubtitleStyle.Render("(no lyrics available for this track)"))
	return b.String()
}

func (m LyricsModel) visibleRows() int {
	rows := m.height - 8
	if rows < 4 {
		return 4
	}
	return rows
}
