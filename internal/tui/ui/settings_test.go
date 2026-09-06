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
	a := &App{width: 100, height: 30}
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
	a := &App{width: 110, height: 30, settingsCursor: 0}
	a.applySetting(0, true)

	out := a.renderSettingsPopup()
	if out == "" {
		t.Fatal("settings popup should not be empty")
	}
	for _, want := range []string{"Hide hint bar", "Hide SOM logo", "Theme", "Default", "ON"} {
		if !strings.Contains(out, want) {
			t.Errorf("popup missing %q", want)
		}
	}
}

func TestSettingsToggleViaKeys(t *testing.T) {
	setTheme(themeDefault)
	defer setTheme(themeDefault)
	a := &App{width: 100, height: 30}

	if _, cmd := a.Update(kp(tea.KeyEsc)); cmd != nil || !a.showSettings {
		t.Fatal("esc should open the settings popup")
	}
	a.Update(kp(tea.KeyRight)) // hide hint: ON
	if !a.hideHint {
		t.Fatal("right on option 0 should enable hide hint")
	}
	a.Update(kp(tea.KeyDown))
	a.Update(kp(tea.KeyRight)) // hide logo: ON
	if !a.hideLogo {
		t.Fatal("right on option 1 should enable hide logo")
	}
	a.Update(kp(tea.KeyDown))
	a.Update(kp(tea.KeyRight)) // theme: Mono
	if !isMono() {
		t.Fatal("right on option 2 should enable mono theme")
	}
	if _, cmd := a.Update(kp(tea.KeyEsc)); cmd != nil || a.showSettings {
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
	if hidden := a.mainContentHeight(); hidden != shown+somLogoRows {
		t.Fatalf("hiding logo should add %d rows, got %d -> %d", somLogoRows, shown, hidden)
	}
}

func TestThemeMonoTurnsWhite(t *testing.T) {
	setTheme(themeDefault)
	defer setTheme(themeDefault)

	setTheme(themeMono)
	if !isMono() {
		t.Fatal("expected mono active")
	}
	sel := SelectedItemStyle.Render("x")
	if !strings.Contains(sel, "48;2;255;255;255") {
		t.Errorf("selected bg should be white in mono: %q", sel)
	}
	if !strings.Contains(renderSOMLogo(), "255;255;255") {
		t.Errorf("logo should be white in mono")
	}
}
