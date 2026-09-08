package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type settingOpt struct {
	title string
	desc  string
	on    func(a *App) bool
	apply func(a *App, on bool)
	// offText/onText hiển thị ở button (mặc định OFF/ON nếu để trống).
	offText string
	onText  string
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
	if len(a.settingsItems) != len(opts) {
		a.settingsItems = nil
		for _, o := range opts {
			if o.offText != "" || o.onText != "" {
				on := o.onText
				if on == "" {
					on = "ON"
				}
				off := o.offText
				if off == "" {
					off = "OFF"
				}
				a.settingsItems = append(a.settingsItems, NewSwitchChoice(o.title, o.desc, o.on(a), off, on))
				continue
			}
			a.settingsItems = append(a.settingsItems, NewSwitch(o.title, o.desc, o.on(a)))
		}
	}
	return a.settingsItems
}

func (a *App) applySetting(i int, on bool) {
	opts := a.settingOptions()
	if i < 0 || i >= len(opts) {
		return
	}
	opts[i].apply(a, on)
}

// popup 2 cột: trái 30% danh sách Switch  phải 70% mô tả
func (a *App) renderSettingsPopup() string {
	items := a.settingSwitches()
	if len(items) == 0 {
		return ""
	}
	if a.settingsCursor < 0 {
		a.settingsCursor = 0
	}
	if a.settingsCursor >= len(items) {
		a.settingsCursor = len(items) - 1
	}

	boxW := a.width - 2
	if boxW > 87 {
		boxW = 87
	}
	if boxW < 40 {
		boxW = 40
	}
	innerW := boxW - 4
	leftW := innerW * 30 / 100
	if leftW < 16 {
		leftW = 16
	}
	rightW := innerW - leftW - 3
	if rightW < 8 {
		rightW = 8
	}

	// Mỗi Switch chiếm 3 dòng cột trái: 2 dòng của View + 1 dòng trống.
	var leftLines []string
	for i, sw := range items {
		block := sw.View(leftW, i == a.settingsCursor)
		leftLines = append(leftLines, block)
		leftLines = append(leftLines, "")
	}

	rightLines := make([]string, len(leftLines))
	descWrap := wordWrap(items[a.settingsCursor].Desc(), rightW-2)
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
