package ui

const skipSilenceSettingKey = "skip_silence"

func skipSilenceOption() settingOpt {
	return settingOpt{
		title: "Skip silence",
		desc:  "Automatically skip silent sections in the music track.",
		on:    func(a *App) bool { return a.skipSilence },
		apply: func(a *App, on bool) {
			a.skipSilence = on
			if a.left.plStore != nil {
				v := "0"
				if on {
					v = "1"
				}
				a.left.plStore.SetSetting(skipSilenceSettingKey, v)
			}
			if a.player != nil {
				a.player.SetSkipSilence(on)
			}
		},
	}
}

func loadSkipSilenceSetting(a *App) {
	a.skipSilence = false
	if a.left.plStore == nil {
		return
	}
	if v := a.left.plStore.GetSetting(skipSilenceSettingKey); v == "1" {
		a.skipSilence = true
	}
	if a.player != nil {
		a.player.SetSkipSilence(a.skipSilence)
	}
}
