package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-runewidth"
)

// ==========================================
// 1. SORT MODAL
// ==========================================
type ApplySortMsg struct {
	Key  string
	Name string
}

type SortModal struct {
	cursor  int
	current string
}

func NewSortModal(current string) *SortModal {
	idx := 0
	for i, s := range sortOptions {
		if s.Key == current {
			idx = i
			break
		}
	}
	return &SortModal{cursor: idx, current: current}
}

func (m *SortModal) Init() tea.Cmd { return nil }

func (m *SortModal) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc", ":", "q":
			return m, func() tea.Msg { return CloseModalMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(sortOptions) - 1
			}
		case "down", "j":
			if m.cursor < len(sortOptions)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
		case "enter":
			chosen := sortOptions[m.cursor]
			return m, tea.Batch(
				func() tea.Msg { return CloseModalMsg{} },
				func() tea.Msg { return ApplySortMsg{Key: chosen.Key, Name: chosen.Name} }, // Bắn message về cho App
			)
		}
	}
	return m, nil
}

func (m *SortModal) View() string {
	const boxW = 35
	const innerW = boxW - 4
	var b strings.Builder
	b.WriteString("\n")
	for i, s := range sortOptions {
		tick := " "
		if s.Key == m.current {
			tick = "+"
		}
		namePart := fmt.Sprintf("   [%s] %s", tick, s.Name)
		if i == m.cursor {
			pad := innerW - runewidth.StringWidth(namePart)
			if pad < 0 {
				pad = 0
			}
			b.WriteString(SelectedItemStyle.Render(namePart+strings.Repeat(" ", pad)) + "\n")
		} else {
			b.WriteString(NormalItemStyle.Render(namePart) + "\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(DimItemStyle.Render(" (enter: apply  | esc: back)"))
	return renderBox(boxW, "Sort by", b.String(), themeCol("#e8593c"))
}

// ==========================================
// 2. SPEED MODAL
// ==========================================
type ApplySpeedMsg struct {
	Index int
	Value float64
	Label string
}

type SpeedModal struct {
	cursor int
	active int
}

func NewSpeedModal(active int) *SpeedModal {
	return &SpeedModal{cursor: active, active: active}
}

func (m *SpeedModal) Init() tea.Cmd { return nil }

func (m *SpeedModal) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc", ":", "q":
			return m, func() tea.Msg { return CloseModalMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(playbackSpeeds) - 1
			}
		case "down", "j":
			if m.cursor < len(playbackSpeeds)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
		case "enter":
			chosen := playbackSpeeds[m.cursor]
			return m, tea.Batch(
				func() tea.Msg { return CloseModalMsg{} },
				func() tea.Msg { return ApplySpeedMsg{Index: m.cursor, Value: chosen.Value, Label: chosen.Label} },
			)
		}
	}
	return m, nil
}

func (m *SpeedModal) View() string {
	const boxW = 35
	const innerW = boxW - 4
	var b strings.Builder
	b.WriteString("\n")
	for i, s := range playbackSpeeds {
		cursor := "   "
		tick := " "
		if i == m.active {
			tick = "+"
		}
		namePart := fmt.Sprintf(" %s [%s] %s", cursor, tick, s.Label)
		if i == m.cursor {
			pad := innerW - runewidth.StringWidth(namePart)
			if pad < 0 {
				pad = 0
			}
			b.WriteString(SelectedItemStyle.Render(namePart+strings.Repeat(" ", pad)) + "\n")
		} else {
			b.WriteString(NormalItemStyle.Render(namePart) + "\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(DimItemStyle.Render(" (enter: apply  | esc: back)"))
	return renderBox(boxW, "Playback speed", b.String(), themeCol("#e8593c"))
}

// ==========================================
// 3. PRESET MODAL
// ==========================================
type ApplyPresetMsg struct {
	Index  int
	Filter string
	Name   string
}

type PresetModal struct {
	cursor int
	active int
}

func NewPresetModal(active int) *PresetModal {
	return &PresetModal{cursor: active, active: active}
}

func (m *PresetModal) Init() tea.Cmd { return nil }

func (m *PresetModal) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc", ":", "q":
			return m, func() tea.Msg { return CloseModalMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(audioPresets) - 1
			}
		case "down", "j":
			if m.cursor < len(audioPresets)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
		case "enter":
			chosen := audioPresets[m.cursor]
			return m, tea.Batch(
				func() tea.Msg { return CloseModalMsg{} },
				func() tea.Msg { return ApplyPresetMsg{Index: m.cursor, Filter: chosen.Filter, Name: chosen.Name} },
			)
		}
	}
	return m, nil
}

func (m *PresetModal) View() string {
	const boxW = 55
	const innerW = boxW - 4
	var b strings.Builder
	b.WriteString("\n")
	for i, p := range audioPresets {
		cursor := " "
		tick := " "
		if i == m.active {
			tick = "+"
		}
		namePart := fmt.Sprintf(" %s [%s] %s", cursor, tick, p.Name)
		if i == m.cursor {
			pad := innerW - runewidth.StringWidth(namePart)
			if pad < 0 {
				pad = 0
			}
			b.WriteString(SelectedItemStyle.Render(namePart+strings.Repeat(" ", pad)) + "\n")
		} else {
			b.WriteString(NormalItemStyle.Render(namePart) + "\n")
		}
	}
	b.WriteString("\n")
	activeDesc := audioPresets[m.cursor].Desc
	b.WriteString(DimItemStyle.Render("  "+activeDesc) + "\n\n")
	b.WriteString(DimItemStyle.Render(" (enter: apply  | esc: back)"))
	return renderBox(boxW, "Audio settings", b.String(), themeCol("#e8593c"))
}
