package ui

const hideLogoSettingKey = "hide_logo"

func logoOption() settingOpt {
	return settingOpt{
		title: "Hide SOM logo",
		desc:  "Hide the SOM banner at the top to free up its terminal rows for the current tab's content (like search or the playlist list).",
		on:    func(a *App) bool { return a.hideLogo },
		apply: func(a *App, on bool) {
			a.hideLogo = on
			// Lưu xuống SQLite để giữ nguyên sau khi thoát app.
			if a.left.plStore != nil {
				v := "0"
				if on {
					v = "1"
				}
				a.left.plStore.SetSetting(hideLogoSettingKey, v)
			}
			if a.width > 0 && a.height > 0 {
				a.resizePanels()
			}
		},
	}
}

func loadHideLogoSetting(a *App) {
	if a.left.plStore == nil {
		return
	}
	if v := a.left.plStore.GetSetting(hideLogoSettingKey); v == "1" || v == "true" {
		a.hideLogo = true
	}
}
