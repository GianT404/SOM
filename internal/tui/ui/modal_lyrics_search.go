package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"som/internal/domain"
	"som/internal/tui/player"

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
	lyricsSearchPreview
)

type LyricsSearchModal struct {
	provider     domain.LyricsSearchProvider
	player       *player.Player
	previewPanel *RightPanel
	track        domain.Track
	input    textinput.Model
	stage    lyricsSearchStage
	results  []domain.LyricsSearchResult
	cursor   int
	preview  int
	offset   int
	width    int
	height   int
	query    string
	err      string
	loading  bool
	request  uint64
}

func NewLyricsSearchModal(provider domain.LyricsSearchProvider, track domain.Track, width, height int, p *player.Player) *LyricsSearchModal {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "Enter track title or artist..."
	input.CharLimit = 100
	input.SetValue(track.Title)
	input.SetWidth(minInt(maxInt(width-20, 24), 72))
	input.Focus()
	input.CursorEnd()
	return &LyricsSearchModal{
		provider: provider,
		player:   p,
		track:    track,
		input:    input,
		width:    width,
		height:   height,
		preview:  -1,
	}
}

func (m *LyricsSearchModal) Init() tea.Cmd { return textinput.Blink }

func (m *LyricsSearchModal) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	if result, ok := msg.(ManualLyricsSearchResultsMsg); ok {
		if result.RequestID != m.request || result.Query != m.query {
			return m, nil
		}
		m.loading = false
		m.results = result.Results
		m.cursor, m.offset, m.preview = 0, 0, -1
		m.previewPanel = nil
		m.err = ""
		if result.Err != nil {
			m.err = result.Err.Error()
		} else if len(result.Results) == 0 {
			m.err = "No lyrics found. Try another title or artist."
		}
		m.stage = lyricsSearchResults
		return m, nil
	}

	if m.stage == lyricsSearchPreview && m.previewPanel != nil {
		if _, ok := msg.(PlaybackTickMsg); ok {
			updated, cmd := m.previewPanel.Update(msg, false, false)
			m.previewPanel = &updated
			return m, cmd
		}
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

	if m.stage == lyricsSearchPreview {
		switch key.String() {
		case "esc":
			m.stage = lyricsSearchResults
			m.preview = -1
			m.previewPanel = nil
		case "enter":
			if m.preview < 0 || m.preview >= len(m.results) {
				return m, nil
			}
			chosen := m.results[m.preview]
			return m, func() tea.Msg {
				return ManualLyricsChosenMsg{VideoID: m.track.ID, Result: chosen}
			}
		}
		return m, nil
	}

	switch key.String() {
	case "esc":
		if m.loading {
			m.request++
			m.loading = false
		}
		m.stage = lyricsSearchInput
		m.preview = -1
		m.previewPanel = nil
		m.input.Focus()
		m.input.CursorEnd()
		m.err = ""
	case "enter":
		if m.loading || m.cursor < 0 || m.cursor >= len(m.results) {
			return m, nil
		}
		// First Enter previews the candidate through the existing playback-synced
		// RightPanel renderer. A second Enter applies and saves it.
		m.beginPreview()
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		m.keepCursorVisible()
	case "down", "j":
		if m.cursor < len(m.results)-1 {
			m.cursor++
		}
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
	m.preview = -1
	m.previewPanel = nil
	m.loading = true
	m.stage = lyricsSearchResults
	return manualLyricsSearchCmd(m.provider, m.query, m.request)
}

func (m *LyricsSearchModal) keepCursorVisible() {
	visible := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m *LyricsSearchModal) visibleRows() int {
	rows := m.height - 13
	if rows < 1 {
		return 1
	}
	if rows > manualLyricsLimit {
		return manualLyricsLimit
	}
	return rows
}

func (m *LyricsSearchModal) previewVisibleRows() int {
	rows := m.height - 15
	if rows < 3 {
		return 3
	}
	if rows > 12 {
		return 12
	}
	return rows
}

func (m *LyricsSearchModal) View() string {
	width := minInt(maxInt(m.width-8, 48), 78)
	if m.width <= 0 {
		width = 68
	}
	innerW := width - 4
	var b strings.Builder
	b.WriteString("  Track: " + m.track.Title + "\n")

	if m.stage == lyricsSearchInput {
		b.WriteString("\n  LRCLib query\n  " + m.input.View() + "\n")
		if m.err != "" {
			b.WriteString("\n  " + StatusErrStyle.Render(m.err) + "\n")
		}
		b.WriteString("\n  Enter: search   Esc: close")
		return renderBox(width, "Search Lyrics on LRCLib", lipgloss.NewStyle().Width(innerW).Render(b.String()), themeCol("#e8593c"))
	}

	if m.stage == lyricsSearchPreview {
		return m.renderPreview(width, innerW)
	}

	b.WriteString("\n  Query: " + m.query + "\n")
	if m.loading {
		b.WriteString("\n  Searching LRCLib...\n")
	} else if len(m.results) == 0 {
		b.WriteString("\n  " + StatusErrStyle.Render(m.err) + "\n\n  Esc: edit query   r: retry")
	} else {
		b.WriteString("\n")
		end := m.offset + m.visibleRows()
		if end > len(m.results) {
			end = len(m.results)
		}
		for i := m.offset; i < end; i++ {
			result := m.results[i]
			kind := "PLAIN"
			if result.HasSynced {
				kind = "SYNCED"
			}
			line := fmt.Sprintf("%d. %s — %s [%s] %s", i+1, result.TrackName,
				fallbackLabel(result.ArtistName, "Unknown artist"), kind, FormatDuration(result.Duration))
			line = runewidth.Truncate(line, innerW-2, "…")
			if i == m.cursor {
				pad := innerW - runewidth.StringWidth(line)
				if pad < 0 {
					pad = 0
				}
				b.WriteString(SelectedItemStyle.Render(line + strings.Repeat(" ", pad)) + "\n")
			} else {
				b.WriteString(LyricNormalStyle.Render(line) + "\n")
			}
		}
		b.WriteString("\n  Up/Down: select   Enter: preview   Esc: edit query")
	}
	return renderBox(width, "Search Lyrics on LRCLib", lipgloss.NewStyle().Width(innerW).Render(strings.TrimSuffix(b.String(), "\n")), themeCol("#e8593c"))
}

func (m *LyricsSearchModal) beginPreview() {
	if m.cursor < 0 || m.cursor >= len(m.results) {
		return
	}

	m.preview = m.cursor
	result := m.results[m.preview]
	panel := NewRightPanel(m.player)
	width := minInt(maxInt(m.width-8, 48), 78)
	if m.width <= 0 {
		width = 68
	}
	panel.SetSize(width, m.previewVisibleRows()+2)
	track := m.track
	panel.SetTrack(&track)
	panel.SetLyrics(manualLyricsResponse(m.track.ID, result))
	if m.player != nil {
		panel.elapsed = m.player.Position()
	}
	panel.syncSyncedLine(true)

	m.previewPanel = &panel
	m.stage = lyricsSearchPreview
}

func (m *LyricsSearchModal) renderPreview(width, innerW int) string {
	var b strings.Builder
	if m.preview < 0 || m.preview >= len(m.results) || m.previewPanel == nil {
		b.WriteString("  No result selected.\n\n  Esc: back to results")
		return renderBox(width, "Lyrics Preview", lipgloss.NewStyle().Width(innerW).Render(b.String()), themeCol("#e8593c"))
	}

	result := m.results[m.preview]
	title := fallbackLabel(result.TrackName, m.track.Title)
	artist := fallbackLabel(result.ArtistName, "Unknown artist")
	b.WriteString("  " + runewidth.Truncate(title+" — "+artist, innerW-2, "…") + "\n")

	kind := "Plain lyrics (not time-synced)"
	if len(result.Lyrics.Synced) > 0 {
		kind = "Synced lyrics · follows playback"
	}
	b.WriteString("  " + DimItemStyle.Render(kind) + "\n\n")

	m.previewPanel.SetSize(width, m.previewVisibleRows()+2)
	b.WriteString(m.previewPanel.renderLyrics(innerW, 0) + "\n")
	b.WriteString("\n  Enter: apply & save   Esc: back to results")
	return renderBox(width, "Lyrics Preview", lipgloss.NewStyle().Width(innerW).Render(strings.TrimSuffix(b.String(), "\n")), themeCol("#e8593c"))
}

func manualLyricsResponse(videoID string, result domain.LyricsSearchResult) domain.LyricsResp {
	lyricTrack := result.Lyrics
	lyricTrack.Language = "lrclib"
	if lyricTrack.TrackName == "" {
		lyricTrack.TrackName = result.TrackName
	}
	if lyricTrack.ArtistName == "" {
		lyricTrack.ArtistName = result.ArtistName
	}
	lr := domain.LyricsResp{
		Title: result.TrackName, Artist: result.ArtistName, VideoID: videoID,
		AllTracks: []domain.LyricsTrack{lyricTrack},
	}
	lr.SelectLanguage(0)
	return lr
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
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
