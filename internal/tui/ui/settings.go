package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type settingOpt struct {
	title   string
	desc    string
	on      func(a *App) bool
	apply   func(a *App, on bool)
	offText string
	onText  string
}

type SettingsModal struct {
	cursor int
	items  []Switch
	width  int
}

func (a *App) settingOptions() []settingOpt {
	return []settingOpt{
		hintBarOption(),
		logoOption(),
		themeOption(),
		mouseOption(),
		skipSilenceOption(),
	}
}

func (a *App) loadSettings() {
	loadHintBarSetting(a)
	loadHideLogoSetting(a)
	loadThemeSetting(a)
	loadMouseSetting(a)
	loadSkipSilenceSetting(a)
}

func (a *App) settingSwitches() []Switch {
	opts := a.settingOptions()
	var items []Switch

	for _, o := range opts {
		if o.offText != "" || o.onText != "" {
			on := o.onText
			if on == "" {
				on = "On"
			}
			off := o.offText
			if off == "" {
				off = "Off"
			}
			items = append(items, NewSwitchChoice(o.title, o.desc, o.on(a), off, on))
			continue
		}
		items = append(items, NewSwitch(o.title, o.desc, o.on(a)))
	}

	return items
}

func (a *App) applySetting(i int, on bool) {
	opts := a.settingOptions()
	if i < 0 || i >= len(opts) {
		return
	}
	opts[i].apply(a, on)
}

func NewSettingsModal(items []Switch, w int) *SettingsModal {
	return &SettingsModal{cursor: 0, items: items, width: w}
}
func (m *SettingsModal) Init() tea.Cmd { return nil }
func (m *SettingsModal) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc", "q":
			return m, func() tea.Msg { return CloseModalMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "left", "h":
			m.items[m.cursor].ToggleLeft()
			idx := m.cursor
			val := m.items[idx].Value()
			return m, func() tea.Msg { return ApplySettingMsg{Index: idx, Value: val} }
		case "right", "l":
			m.items[m.cursor].ToggleRight()
			idx := m.cursor
			val := m.items[idx].Value()
			return m, func() tea.Msg { return ApplySettingMsg{Index: idx, Value: val} }
		}
	}
	return m, nil
}

func (m *SettingsModal) View() string {
	if len(m.items) == 0 {
		return ""
	}
	boxW := m.width - 2
	if boxW > 85 {
		boxW = 85
	}
	if boxW < 40 {
		boxW = 40
	}
	innerW := boxW - 4
	leftW := innerW * 40 / 100
	if leftW < 16 {
		leftW = 16
	}
	rightW := innerW - leftW - 3
	if rightW < 8 {
		rightW = 8
	}

	var leftLines []string
	for i, sw := range m.items {
		block := sw.View(leftW, i == m.cursor)
		leftLines = append(leftLines, block)
		leftLines = append(leftLines, "")
	}
	rightLines := make([]string, len(leftLines))
	descWrap := wordWrap(m.items[m.cursor].Desc(), rightW-2)
	for j, l := range descWrap {
		if j < len(rightLines) {
			rightLines[j] = l
		}
	}
	left := lipgloss.NewStyle().Width(leftW).Render(strings.Join(leftLines, "\n"))
	right := lipgloss.NewStyle().Width(rightW).Render(strings.Join(rightLines, "\n"))
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, lipgloss.NewStyle().Width(3).Render("   "), right)
	return renderBox(boxW, "Settings", body, colorAccent)
}
