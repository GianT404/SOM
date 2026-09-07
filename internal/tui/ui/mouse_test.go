package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"som/internal/domain"
)

func mouseApp(t *testing.T) *App {
	t.Helper()
	setTheme(themeDefault)
	panel := NewLeftPanel(nil, t.TempDir())
	panel.SetSize(80, 20)
	panel.locals = []LocalFile{
		{Name: "Song A", Duration: 180},
		{Name: "Song B", Duration: 200},
		{Name: "Song C", Duration: 220},
	}
	return &App{
		width:         100,
		height:        30,
		left:          panel,
		sidebarActive: SideDownloads,
		activeContext: SideDownloads,
		mouseEnabled:  true,
	}
}

func TestMouseClickSelectsRowInDownloads(t *testing.T) {
	a := mouseApp(t)
	if !a.left.input.Focused() {
		t.Fatal("precondition: NewLeftPanel starts with input focused")
	}
	// Downloads: item0 tại contentTop(7) + 5 = 12. Click y=13 => item 1.
	if cmd := a.handleMouseClick(tea.MouseClickMsg{X: 60, Y: 13, Button: tea.MouseLeft}); cmd != nil {
		t.Fatalf("unexpected cmd: %v", cmd)
	}
	if a.left.dlCursor != 1 {
		t.Fatalf("expected dlCursor=1, got %d", a.left.dlCursor)
	}
	// Click vào track phải blur input để border playlist được focus.
	if a.left.input.Focused() {
		t.Fatal("clicking a track should blur the search input so the list border is focused")
	}
}

func TestMouseClickSwitchesSidebarTab(t *testing.T) {
	a := mouseApp(t)
	// Hàng sidebar 2 (SideImport) bắt đầu tại contentTop=7 => y=9.
	a.handleMouseClick(tea.MouseClickMsg{X: 3, Y: 9, Button: tea.MouseLeft})
	if a.sidebarActive != SideImport {
		t.Fatalf("expected SideImport, got %v", a.sidebarActive)
	}
}

func TestMouseClickFocusesSearchInput(t *testing.T) {
	a := mouseApp(t)
	a.left.input.Blur()
	if a.left.input.Focused() {
		t.Fatal("precondition: input should be blurred")
	}
	// Downloads: dòng input nằm tại contentTop(7)+1 = 8.
	a.handleMouseClick(tea.MouseClickMsg{X: 60, Y: 8, Button: tea.MouseLeft})
	if !a.left.input.Focused() {
		t.Fatal("clicking the search row should focus the input")
	}
}

func TestMouseRowMappingInDownloads(t *testing.T) {
	a := mouseApp(t)
	a.left.input.Blur()
	for want, y := range []int{12, 13, 14} { // item0..2
		a.handleMouseClick(tea.MouseClickMsg{X: 60, Y: y, Button: tea.MouseLeft})
		if a.left.dlCursor != want {
			t.Fatalf("click y=%d should select item %d, got dlCursor=%d", y, want, a.left.dlCursor)
		}
	}
}

func TestMouseWheelDisabledWhenOff(t *testing.T) {
	a := mouseApp(t)
	a.mouseEnabled = false
	a.left.dlCursor = 0

	a.Update(tea.MouseWheelMsg{X: 60, Y: 20, Button: tea.MouseWheelDown})
	if a.left.dlCursor != 0 {
		t.Fatalf("wheel should be ignored when mouse is off, got dlCursor=%d", a.left.dlCursor)
	}

	a.mouseEnabled = true
	a.left.input.Blur()
	a.Update(tea.MouseWheelMsg{X: 60, Y: 20, Button: tea.MouseWheelDown})
	if a.left.dlCursor != 1 {
		t.Fatalf("wheel should move cursor when mouse is on, got dlCursor=%d", a.left.dlCursor)
	}
}

func TestProgressBarSeekGeometry(t *testing.T) {
	a := mouseApp(t)
	base := a.progressBarTop()
	if base <= 0 {
		t.Fatalf("expected positive progress bar top, got %d", base)
	}

	// Không có player → click progress không crash, trả về không xử lý.
	a.nowPlay = &domain.Track{Duration: 200}
	if a.seekFromProgressClick(tea.MouseClickMsg{X: 30, Y: base + 1, Button: tea.MouseLeft}) {
		t.Fatal("should not seek without a player")
	}
}
