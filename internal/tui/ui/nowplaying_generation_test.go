package ui

import (
	"testing"

	"som/internal/domain"
)

func TestLeftPanelIgnoresStaleTrackChange(t *testing.T) {
	p := LeftPanel{
		tracks:   []domain.Track{{ID: "current"}},
		trackGen: 5,
	}
	p.searchCursor = 0

	p, _ = p.Update(TrackChangedMsg{
		Track: domain.Track{ID: "old"},
		Gen:   4,
	}, false, nil)

	if p.trackGen != 5 {
		t.Fatalf("track generation=%d, want 5", p.trackGen)
	}
	if p.searchCursor != 0 {
		t.Fatalf("search cursor=%d, want 0", p.searchCursor)
	}
}

func TestLeftPanelIgnoresStaleStreamResult(t *testing.T) {
	p := LeftPanel{
		activeTab:     SideSearch,
		trackGen:      5,
		loadingStream: true,
	}

	p, _ = p.Update(StreamResolvedMsg{Gen: 4}, false, nil)
	if !p.loadingStream {
		t.Fatal("stale stream result should not clear loading state")
	}

	p, _ = p.Update(StreamResolvedMsg{Gen: 5}, false, nil)
	if p.loadingStream {
		t.Fatal("current stream result should clear loading state")
	}
}
