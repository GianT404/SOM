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
}

func (a *App) settingOptions() []settingOpt {
	return []settingOpt{
		hintBarOption(),
		logoOption(),
	}
}

func (a *App) loadSettings() {
	loadHintBarSetting(a)
	loadHideLogoSetting(a)
}

func (a *App) settingSwitches() []Switch {
	opts := a.settingOptions()
	if len(a.settingsItems) != len(opts) {
		a.settingsItems = nil
		for _, o := range opts {
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
	const rowsPerOpt = 3
	var leftLines []string
	for i, sw := range items {
		block := sw.View(leftW, i == a.settingsCursor)
		leftLines = append(leftLines, block)
		leftLines = append(leftLines, "")
	}

	// Cột phải: mô tả tuỳ chọn đang chọn, căn ngang với block của nó.
	rightLines := make([]string, len(leftLines))
	descWrap := wordWrap(items[a.settingsCursor].Desc(), rightW-2)
	start := a.settingsCursor * rowsPerOpt
	for j, l := range descWrap {
		if start+j < len(rightLines) {
			rightLines[start+j] = l
		}
	}

	left := lipgloss.NewStyle().Width(leftW).Render(strings.Join(leftLines, "\n"))
	right := lipgloss.NewStyle().Width(rightW).Render(strings.Join(rightLines, "\n"))
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, lipgloss.NewStyle().Width(3).Render("   "), right)

	return renderBox(boxW, "Settings", body, colorAccent)
}
