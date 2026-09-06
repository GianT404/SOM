package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type themeMode int

const (
	themeDefault themeMode = iota
	themeMono
)

var currentTheme = themeDefault

func isMono() bool {
	return currentTheme == themeMono
}

func themeCol(hex string) color.Color {
	if isMono() {
		return lipgloss.Color("#ffffff")
	}
	return lipgloss.Color(hex)
}

func setTheme(t themeMode) {
	currentTheme = t
	rebuildTheme()
}

// themeName dùng để hiển thị / lưu xuống DB.
func themeName() string {
	if isMono() {
		return "mono"
	}
	return "default"
}

func parseTheme(s string) themeMode {
	if s == "mono" {
		return themeMono
	}
	return themeDefault
}

const themeSettingKey = "theme"

func themeOption() settingOpt {
	return settingOpt{
		title:   "Theme",
		desc:    "Default keeps the colorful palette. Mono turns text, borders and highlights white (#fff) for a clean monochrome look.",
		offText: "Default",
		onText:  "Mono",
		on:      func(a *App) bool { return isMono() },
		apply: func(a *App, mono bool) {
			mode := themeDefault
			if mono {
				mode = themeMono
			}
			setTheme(mode)
			if a.left.plStore != nil {
				a.left.plStore.SetSetting(themeSettingKey, themeName())
			}
		},
	}
}

func loadThemeSetting(a *App) {
	if a.left.plStore == nil {
		return
	}
	setTheme(parseTheme(a.left.plStore.GetSetting(themeSettingKey)))
}
