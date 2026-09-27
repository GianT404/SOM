package render

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestProgressBarEmpty(t *testing.T) {
	style := lipgloss.NewStyle()
	got := ProgressBar(20, 0, 120, "", "00:00", ProgressBarStyles{
		Border:     style,
		Filled:     style,
		Time:       style,
		TimeFilled: style,
		Controls:   style,
	})
	if !strings.Contains(got, "00:00") {
		t.Fatalf("missing time label: %q", got)
	}
	if lines := strings.Count(got, "\n") + 1; lines != 3 {
		t.Fatalf("expected 3 lines, got %d", lines)
	}
}

func TestProgressBarClampsInput(t *testing.T) {
	style := lipgloss.NewStyle()
	got := ProgressBar(2, 999, 120, "Track", "20:00", ProgressBarStyles{
		Border:     style,
		Title:      style,
		Filled:     style,
		Time:       style,
		TimeFilled: style,
		Controls:   style,
	})
	if !strings.Contains(got, "Track") || !strings.Contains(got, "20:00") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestProgressBarDoesNotFillWithoutDuration(t *testing.T) {
	style := lipgloss.NewStyle()
	got := ProgressBar(20, 30, 0, "", "00:30", ProgressBarStyles{
		Border:     style,
		Filled:     style,
		Time:       style,
		TimeFilled: style,
		Controls:   style,
	})
	if strings.Contains(got, "█") {
		t.Fatalf("expected no fill without total duration: %q", got)
	}
}
