package render

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func testWrap(s string, _ int) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func TestLyricAnimationEmptySize(t *testing.T) {
	got := LyricAnimation("old", "new", 0, 10, 0.5, LyricAnimationStyles{}, testWrap)
	if got != "" {
		t.Fatalf("expected empty output, got %q", got)
	}
}

func TestLyricAnimationCompleted(t *testing.T) {
	style := lipgloss.NewStyle()
	got := LyricAnimation(
		"old",
		"new",
		12,
		5,
		1,
		LyricAnimationStyles{Highlight: style},
		testWrap,
	)

	if lines := strings.Count(got, "\n") + 1; lines != 5 {
		t.Fatalf("expected 5 lines, got %d", lines)
	}
	if !strings.Contains(got, "new") {
		t.Fatalf("expected current lyric in output, got %q", got)
	}
}

func TestLyricAnimationTransition(t *testing.T) {
	style := lipgloss.NewStyle()
	got := LyricAnimation(
		"old",
		"new",
		12,
		6,
		0.25,
		LyricAnimationStyles{Highlight: style, Dim: style},
		testWrap,
	)

	if !strings.Contains(got, "old") || !strings.Contains(got, "new") {
		t.Fatalf("expected both lyric states during transition, got %q", got)
	}
}
