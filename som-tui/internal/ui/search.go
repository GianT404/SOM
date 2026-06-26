// internal/ui/search.go
// Search screen: text input at the top, scrollable results list below.
// Keys: enter to search, ↑/↓ to navigate, enter on a result to play,
//
//	d on a result to download.
package ui

import (
	"fmt"
	"strings"

	"github.com/GianT404/SOM/tui/internal/api"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ─── Model ───────────────────────────────────────────────────────────────────

type SearchModel struct {
	client   *api.Client
	input    textinput.Model
	spinner  spinner.Model
	tracks   []api.Track
	cursor   int
	loading  bool
	searched bool // true once first search done
	width    int
	height   int
	err      string
	offset   int // scroll offset
}

func NewSearchModel(c *api.Client) SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Search YouTube…  (press Enter)"
	ti.CharLimit = 120
	ti.Width = 58
	ti.PromptStyle = InputPromptStyle
	ti.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = StatusMsgStyle

	return SearchModel{
		client:  c,
		input:   ti,
		spinner: sp,
	}
}

func (m *SearchModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.input.Width = min(w-10, 70)
}

// ─── Init ─────────────────────────────────────────────────────────────────────

func (m SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

// ─── Update ───────────────────────────────────────────────────────────────────

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.loading {
				break
			}
			q := strings.TrimSpace(m.input.Value())
			if q == "" {
				break
			}
			m.loading = true
			m.err = ""
			cmds = append(cmds, m.spinner.Tick, searchCmd(m.client, q))

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}

		case "down", "j":
			if m.cursor < len(m.tracks)-1 {
				m.cursor++
				visible := m.visibleRows()
				if m.cursor >= m.offset+visible {
					m.offset++
				}
			}

		case "d":
			if len(m.tracks) > 0 && !m.loading {
				t := m.tracks[m.cursor]
				return m, DownloadCmd(m.client, t, "downloads")
			}
		}

		// play on Enter when list is focused (input not focused after search)
		if msg.String() == "enter" && m.searched && !m.input.Focused() && len(m.tracks) > 0 {
			t := m.tracks[m.cursor]
			return m, func() tea.Msg { return PlayStartedMsg{Track: t} }
		}

	case SearchResultMsg:
		m.loading = false
		m.searched = true
		m.cursor = 0
		m.offset = 0
		m.input.Blur()
		if msg.Err != nil {
			m.err = msg.Err.Error()
			m.tracks = nil
		} else {
			m.tracks = msg.Tracks
			if len(m.tracks) == 0 {
				m.err = "No results found."
			}
		}

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// delegate input events
	if m.input.Focused() || !m.searched {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (m SearchModel) View() string {
	var b strings.Builder

	// Input row
	b.WriteString("\n")
	b.WriteString(InputPromptStyle.Render("🔍 "))
	b.WriteString(m.input.View())
	if m.loading {
		b.WriteString("  " + m.spinner.View() + " searching…")
	}
	b.WriteString("\n\n")

	// Error
	if m.err != "" {
		b.WriteString(StatusErrStyle.Render("✗ " + m.err))
		b.WriteString("\n")
		return b.String()
	}

	if !m.searched {
		b.WriteString(SubtitleStyle.Render("Type a song name or artist and press Enter."))
		return b.String()
	}

	if len(m.tracks) == 0 {
		return b.String()
	}

	// Column header
	b.WriteString(DimItemStyle.Render(
		fmt.Sprintf("  %-52s %-28s %6s", "Title", "Artist", "Dur"),
	))
	b.WriteString("\n")
	b.WriteString(DimItemStyle.Render(strings.Repeat("─", 90)))
	b.WriteString("\n")

	// Track rows
	visible := m.visibleRows()
	end := m.offset + visible
	if end > len(m.tracks) {
		end = len(m.tracks)
	}
	for i := m.offset; i < end; i++ {
		t := m.tracks[i]
		line := fmt.Sprintf("%2d  %-52s %-28s %6s",
			i+1,
			truncate(t.Title, 52),
			truncate(t.Artist, 28),
			FormatDuration(t.Duration),
		)
		if i == m.cursor {
			b.WriteString(SelectedItemStyle.Render(line))
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	// Scroll hint
	if len(m.tracks) > visible {
		b.WriteString(DimItemStyle.Render(
			fmt.Sprintf("  %d–%d of %d  (↑/↓ to scroll)", m.offset+1, end, len(m.tracks)),
		))
	}

	return b.String()
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func (m SearchModel) visibleRows() int {
	rows := m.height - 10
	if rows < 4 {
		return 4
	}
	return rows
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─── Commands ─────────────────────────────────────────────────────────────────

func searchCmd(c *api.Client, q string) tea.Cmd {
	return func() tea.Msg {
		tracks, err := c.Search(q)
		return SearchResultMsg{Tracks: tracks, Err: err}
	}
}
