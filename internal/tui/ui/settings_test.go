package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"som/internal/storage"
)

func kp(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

func TestSettingsDefaults(t *testing.T) {
	setTheme(themeDefault)
	a := &App{
		width:    100,
		height:   30,
		playback: NewPlaybackManager(),
		palette:  NewCommandPalette(),
	}
	if a.hideHint || a.hideLogo {
		t.Fatal("hide options should default to false")
	}
	if isMono() {
		t.Fatal("theme should default to default")
	}
}

func TestRenderSettingsPopup(t *testing.T) {
	setTheme(themeDefault)
	defer setTheme(themeDefault)
	a := &App{width: 110, height: 30}
	a.applySetting(0, true)

	modal := NewSettingsModal(a.settingSwitches(), a.width)
	out := modal.View()

	if out == "" {
		t.Fatal("settings popup should not be empty")
	}
	for _, want := range []string{"Hide hint bar", "Hide SOM logo", "Theme", "Mouse support", "Default", "ON"} {
		if !strings.Contains(out, want) {
			t.Errorf("popup missing %q", want)
		}
	}
}

func TestSettingsToggleViaKeys(t *testing.T) {
	setTheme(themeDefault)
	defer setTheme(themeDefault)
	a := &App{
		width:    100,
		height:   30,
		playback: NewPlaybackManager(),
		palette:  NewCommandPalette(),
	}

	// 1. Ấn esc mở menu
	_, cmd := a.Update(kp(tea.KeyEsc))
	if len(a.modals) == 0 { // Kiểm tra Modal stack
		t.Fatal("esc should open the esc menu")
	}

	a.Update(OpenSettingsMsg{})

	if len(a.modals) == 0 {
		t.Fatal("enter on Settings should open settings popup")
	}

	// Lấy SettingsModal ra khỏi stack để test phím
	settingsModal, ok := a.modals[len(a.modals)-1].(*SettingsModal)
	if !ok {
		t.Fatal("Top modal is not SettingsModal")
	}

	// 3. Giả lập bấm phím trong Settings
	// Dùng m.Update của Modal, KHÔNG dùng a.Update
	_, cmd = settingsModal.Update(kp(tea.KeyRight))

	// Khi ấn qua trái phải, nó trả về ApplySettingMsg
	applyMsg, ok := cmd().(ApplySettingMsg)
	if !ok {
		t.Fatal("expected ApplySettingMsg")
	}
	a.Update(applyMsg) // Ép App áp dụng setting

	if !a.hideHint {
		t.Fatal("right on option 0 should enable hide hint")
	}

	// Tương tự, gõ xuống và qua phải để đổi logo
	settingsModal.Update(kp(tea.KeyDown))
	_, cmd = settingsModal.Update(kp(tea.KeyRight))
	applyMsg = cmd().(ApplySettingMsg)
	a.Update(applyMsg)

	if !a.hideLogo {
		t.Fatal("right on option 1 should enable hide logo")
	}

	// Đổi theme
	settingsModal.Update(kp(tea.KeyDown))
	_, cmd = settingsModal.Update(kp(tea.KeyRight))
	applyMsg = cmd().(ApplySettingMsg)
	a.Update(applyMsg)

	if !isMono() {
		t.Fatal("right on option 2 should enable mono theme")
	}

	// 4. Test thoát
	_, cmd = settingsModal.Update(kp(tea.KeyEsc))
	closeMsg, ok := cmd().(CloseModalMsg)
	if !ok {
		t.Fatal("esc should return CloseModalMsg")
	}

	a.Update(closeMsg) // Ép App đóng modal
	if len(a.modals) != 0 {
		t.Fatal("esc should close the settings popup")
	}
}
func TestSettingsPersistAcrossRestart(t *testing.T) {
	setTheme(themeDefault)
	defer setTheme(themeDefault)
	db, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	a := &App{width: 100, height: 30, left: LeftPanel{plStore: db}}
	a.applySetting(0, true)
	a.applySetting(1, true)
	a.applySetting(2, true) // mono
	for key, want := range map[string]string{
		"hide_hint": "1",
		"hide_logo": "1",
		"theme":     "mono",
	} {
		if got := db.GetSetting(key); got != want {
			t.Fatalf("expected %s=%q, got %q", key, want, got)
		}
	}

	b := &App{width: 100, height: 30, left: LeftPanel{plStore: db}}
	b.loadSettings()
	if !b.hideHint || !b.hideLogo || !isMono() {
		t.Fatalf("settings should restore (hint=%v logo=%v mono=%v)", b.hideHint, b.hideLogo, isMono())
	}
}

func TestHideLogoReclaimsRows(t *testing.T) {
	setTheme(themeDefault)
	a := &App{width: 100, height: 40}
	shown := a.mainContentHeight()

	a.applySetting(1, true)
	if a.somRowHeight() != 0 {
		t.Fatal("somRowHeight should be 0 when logo hidden")
	}
	if hidden := a.mainContentHeight(); hidden != shown {
		t.Fatalf("hiding logo should add %d rows, got %d -> %d", shown, hidden)
	}
}

func TestMouseSettingToggleAndPersist(t *testing.T) {
	setTheme(themeDefault)
	db, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	// Mặc định  tắt.
	b := &App{width: 100, height: 30, left: LeftPanel{plStore: db}}
	b.loadSettings()
	if b.mouseEnabled {
		t.Fatal("mouse support should default to disabled")
	}

	a := &App{width: 100, height: 30, left: LeftPanel{plStore: db}, mouseEnabled: false}
	a.applySetting(3, true)
	if !a.mouseEnabled {
		t.Fatal("applySetting(true) should enable mouse")
	}
	if got := db.GetSetting("mouse"); got != "1" {
		t.Fatalf("expected mouse=1, got %q", got)
	}

	c := &App{width: 100, height: 30, left: LeftPanel{plStore: db}}
	c.loadSettings()
	if !c.mouseEnabled {
		t.Fatal("mouse support should be restored as enabled")
	}
}
