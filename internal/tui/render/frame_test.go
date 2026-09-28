package render

import "testing"

func TestBaseFrameOrder(t *testing.T) {
	got := BaseFrame("logo", "separator", "content", "status", "progress", "help", false, false)
	want := "logo\nseparator\ncontent\nstatus\nprogress\nhelp"
	if got != want {
		t.Fatalf("frame = %q, want %q", got, want)
	}
}

func TestBaseFrameHiddenSections(t *testing.T) {
	got := BaseFrame("logo", "separator", "content", "status", "progress", "help", true, true)
	want := "separator\ncontent\nstatus\nprogress\n"
	if got != want {
		t.Fatalf("frame = %q, want %q", got, want)
	}
}
