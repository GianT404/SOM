package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"som/internal/domain"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
)

const manualLyricsLimit = 7

type lyricsSearchStage int

const (
	lyricsSearchInput lyricsSearchStage = iota
	lyricsSearchResults
)

type LyricsSearchModal struct {
	provider domain.LyricsSearchProvider
	track domain.Track
	input textinput.Model
	stage lyricsSearchStage
	results []domain.LyricsSearchResult
	cursor int
	offset int
	width int
	height int
	query string
	err string
	loading bool
	request uint64
}

func NewLyricsSearchModal(provider domain.LyricsSearchProvider, track domain.Track, width, height int) *LyricsSearchModal {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "Enter track title or artist..."
	input.CharLimit = 100
	input.SetValue(track.Title)
	input.SetWidth(minInt(maxInt(width-20, 24), 72))
	input.Focus()
	input.CursorEnd()
	return &LyricsSearchModal{provider: provider, track: track, input: input, width: width, height: height}
}

func (m *LyricsSearchModal) Init() tea.Cmd { return textinput.Blink }

func (m *LyricsSearchModal) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	if result, ok := msg.(ManualLyricsSearchResultsMsg); ok {
		if result.RequestID != m.request || result.Query != m.query {
			return m, nil
		}
		m.loading = false
		m.results = result.Results
		m.cursor, m.offset = 0, 0
		m.err = ""
		if result.Err != nil {
			m.err = result.Err.Error()
		} else if len(result.Results) == 0 {
			m.err = "No lyrics found. Try another title or artist."
		}
		m.stage = lyricsSearchResults
		return m, nil
	}

	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		if m.stage == lyricsSearchInput {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		return m, nil
	}
	if m.stage == lyricsSearchInput {
		switch key.String() {
		case "esc":
			return m, func() tea.Msg { return CloseModalMsg{} }
		case "enter":
			return m, m.searchAgain()
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			m.err = ""
			return m, cmd
		}
	}

	switch key.String() {
	case "esc":
		if m.loading {
			m.request++
			m.loading = false
		}
		m.stage = lyricsSearchInput
		m.input.Focus()
		m.input.CursorEnd()
		m.err = ""
	case "enter":
		if m.loading || m.cursor < 0 || m.cursor >= len(m.results) {
			return m, nil
		}
		chosen := m.results[m.cursor]
		return m, func() tea.Msg { return ManualLyricsChosenMsg{VideoID: m.track.ID, Result: chosen} }
	case "up", "k":
		if m.cursor > 0 { m.cursor-- }
		m.keepCursorVisible()
	case "down", "j":
		if m.cursor < len(m.results)-1 { m.cursor++ }
		m.keepCursorVisible()
	case "r":
		return m, m.searchAgain()
	}
	return m, nil
}

func (m *LyricsSearchModal) searchAgain() tea.Cmd {
	m.request++
	m.query = strings.TrimSpace(m.input.Value())
	if m.query == "" {
		m.stage = lyricsSearchInput
		m.err = "Enter a title or artist to search."
		return nil
	}
	m.results, m.err = nil, ""
	m.loading = true
	m.stage = lyricsSearchResults
	return manualLyricsSearchCmd(m.provider, m.query, m.request)
}

func (m *LyricsSearchModal) keepCursorVisible() {
	visible := m.visibleRows()
	if m.cursor < m.offset { m.offset = m.cursor }
	if m.cursor >= m.offset+visible { m.offset = m.cursor-visible+1 }
	if m.offset < 0 { m.offset = 0 }
}

func (m *LyricsSearchModal) visibleRows() int {
	rows := m.height - 13
	if rows < 1 { return 1 }
	if rows > manualLyricsLimit { return manualLyricsLimit }
	return rows
}

func (m *LyricsSearchModal) View() string {
	width := minInt(maxInt(m.width-8, 48), 78)
	if m.width <= 0 { width = 68 }
	innerW := width - 4
	var b strings.Builder
	b.WriteString("  Track: " + m.track.Title + "\n")

	if m.stage == lyricsSearchInput {
		b.WriteString("\n  LRCLib query\n  " + m.input.View() + "\n")
		if m.err != "" { b.WriteString("\n  " + StatusErrStyle.Render(m.err) + "\n") }
		b.WriteString("\n  Enter: search   Esc: close")
		return renderBox(width, "Search Lyrics on LRCLib", lipgloss.NewStyle().Width(innerW).Render(b.String()), themeCol("#e8593c"))
	}

	b.WriteString("\n  Query: " + m.query + "\n")
	if m.loading {
		b.WriteString("\n  Searching LRCLib...\n")
	} else if len(m.results) == 0 {
		b.WriteString("\n  " + StatusErrStyle.Render(m.err) + "\n\n  Esc: edit query   r: retry")
	} else {
		b.WriteString("\n")
		end := m.offset + m.visibleRows()
		if end > len(m.results) { end = len(m.results) }
		for i := m.offset; i < end; i++ {
			result := m.results[i]
			kind := "PLAIN"
			if result.HasSynced { kind = "SYNCED" }
			line := fmt.Sprintf("%d. %s — %s [%s] %s", i+1, result.TrackName,
				fallbackLabel(result.ArtistName, "Unknown artist"), kind, FormatDuration(result.Duration))
			line = runewidth.Truncate(line, innerW-2, "…")
			if i == m.cursor {
				pad := innerW-runewidth.StringWidth(line)
				if pad < 0 { pad = 0 }
				b.WriteString(SelectedItemStyle.Render(line+strings.Repeat(" ", pad)) + "\n")
			} else {
				b.WriteString(LyricNormalStyle.Render(line) + "\n")
			}
		}
		b.WriteString("\n  Up/Down: select   Enter: apply   Esc: edit query")
	}
	return renderBox(width, "Search Lyrics on LRCLib", lipgloss.NewStyle().Width(innerW).Render(strings.TrimSuffix(b.String(), "\n")), themeCol("#e8593c"))
}

func manualLyricsSearchCmd(provider domain.LyricsSearchProvider, query string, requestID uint64) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		results, err := provider.SearchLyrics(ctx, query, manualLyricsLimit)
		return ManualLyricsSearchResultsMsg{Query: query, RequestID: requestID, Results: results, Err: err}
	}
}

func fallbackLabel(value, fallback string) string {
	if strings.TrimSpace(value) == "" { return fallback }
	return value
}
