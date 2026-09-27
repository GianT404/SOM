package ui

import (
	"image/color"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestRenderBoxKeepsRequestedWidth(t *testing.T) {
	const width = 20

	got := renderBox(width, "", "this content is intentionally wider than the box", color.White)
	for i, line := range strings.Split(got, "\n") {
		if gotWidth := lipgloss.Width(line); gotWidth != width {
			t.Fatalf("line %d has width %d, want %d: %q", i, gotWidth, width, line)
		}
	}
}

func TestRenderBoxTruncatesLongTitle(t *testing.T) {
	const width = 16

	got := renderBox(width, "A very long title", "body", color.White)
	lines := strings.Split(got, "\n")
	if len(lines) == 0 {
		t.Fatal("renderBox returned no lines")
	}

	if gotWidth := lipgloss.Width(lines[0]); gotWidth != width {
		t.Fatalf("top border has width %d, want %d: %q", gotWidth, width, lines[0])
	}
}
