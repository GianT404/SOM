package ui

const hintBarSettingKey = "hide_hint"

func hintBarOption() settingOpt {
	return settingOpt{
		title: "Hide hint bar",
		desc:  "Hide the keyboard-shortcut hint line below the progress bar to free up one terminal row. Press '?' any time to see all shortcuts.",
		on:    func(a *App) bool { return a.hideHint },
		apply: func(a *App, on bool) {
			a.hideHint = on
			// Lưu xuống SQLite để giữ nguyên sau khi thoát app.
			if a.left.plStore != nil {
				v := "0"
				if on {
					v = "1"
				}
				a.left.plStore.SetSetting(hintBarSettingKey, v)
			}
			// Cập nhật chiều cao panel theo contentH mới để tận dụng hàng vừa
			// giải phóng (playlist không còn dư 1 hàng trống).
			if a.width > 0 && a.height > 0 {
				a.resizePanels()
			}
		},
	}
}

// loadHintBarSetting khôi phục trạng thái ẩn hint từ DB lúc khởi động.
func loadHintBarSetting(a *App) {
	if a.left.plStore == nil {
		return
	}
	if v := a.left.plStore.GetSetting(hintBarSettingKey); v == "1" || v == "true" {
		a.hideHint = true
	}
}
