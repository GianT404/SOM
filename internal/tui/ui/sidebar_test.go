package ui

import (
	"strings"
	"testing"
	"time"
)

func TestRenderSidebarGhost(t *testing.T) {
	now := time.Now()
	// anim đang giữa chừng (p~0.5), từ Search(0) tới Logs(4)
	anim := sidebarAnimState{
		on:    true,
		from:  SideSearch,
		to:    SideLogs,
		start: now.Add(-200 * time.Millisecond),
		end:   now.Add(200 * time.Millisecond),
	}
	out := renderSidebar(SideLogs, anim, 5)
	markers := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "| ") {
			markers++
		}
	}
	t.Logf("markers at mid-flight = %d (expect active 1 + ghost trail > 0)", markers)
	if markers < 2 {
		t.Fatalf("expected ghost trail during animation, got %d markers\n%s", markers, out)
	}

	// anim đã xong → chỉ còn 1 marker active
	done := sidebarAnimState{on: true, from: SideSearch, to: SideLogs,
		start: now.Add(-2 * time.Second), end: now.Add(-1 * time.Second)}
	outDone := renderSidebar(SideLogs, done, 5)
	doneMarkers := strings.Count(outDone, "| ")
	if doneMarkers != 1 {
		t.Fatalf("after animation finished expected exactly 1 marker, got %d\n%s", doneMarkers, outDone)
	}

	// không anim → 1 marker
	off := renderSidebar(SideLogs, sidebarAnimState{}, 5)
	if n := strings.Count(off, "| "); n != 1 {
		t.Fatalf("no anim expected 1 marker, got %d", n)
	}
	t.Logf("OK: %d markers mid, %d after done, %d off", markers, doneMarkers, strings.Count(off, "| "))
}
