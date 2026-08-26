package ui

import (
	"time"

	"som/internal/domain"
	"som/internal/tui/player"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type splashDoneMsg struct {
	player *player.Player
	left   LeftPanel
}

type splashTickMsg time.Time

func splashTick() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return splashTickMsg(t)
	})
}

// không chặn vòng lặp Bubble Tea nên splash vẫn animate mượt trong lúc chờ.
func bootCmd(provider domain.MusicProvider) tea.Cmd {
	return func() tea.Msg {
		p := player.New()
		left := NewLeftPanel(provider)
		return splashDoneMsg{player: p, left: left}
	}
}

func renderSplash(width, height, frame int) string {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}

	logo := animeFrames[frame%len(animeFrames)]

	spinnerFrames := []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
	dot := spinnerFrames[frame%len(spinnerFrames)]
	loadingLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#C84328")).
		Render(string(dot) + " loading your library...")

	block := lipgloss.JoinVertical(lipgloss.Center,
		logo,
		"",
		loadingLine,
	)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, block)
}
