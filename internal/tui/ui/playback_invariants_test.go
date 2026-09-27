package ui

import (
	"testing"

	"som/internal/domain"
)

func TestNormalizePlaybackStateClampsCoreState(t *testing.T) {
	pm := NewPlaybackManager()
	pm.Playlist = []domain.Track{
		{ID: "a"},
		{ID: "b"},
	}
	pm.CurrentIdx = 99
	pm.NowPlay = &pm.Playlist[0]
	pm.SongStarted = true
	pm.ShuffleHist = []int{-1, 0, 7, 1}

	pm.normalizeState()

	if pm.CurrentIdx != 1 {
		t.Fatalf("current index=%d, want 1", pm.CurrentIdx)
	}
	if !pm.SongStarted {
		t.Fatal("song should stay started while NowPlay is set")
	}
	if len(pm.ShuffleHist) != 2 || pm.ShuffleHist[0] != 0 || pm.ShuffleHist[1] != 1 {
		t.Fatalf("shuffle history=%v, want [0 1]", pm.ShuffleHist)
	}
}

func TestNormalizePlaybackStateClearsInvalidEmptyState(t *testing.T) {
	pm := NewPlaybackManager()
	pm.CurrentIdx = 5
	pm.SongStarted = true
	pm.ShuffleHist = []int{0, 1}

	pm.normalizeState()

	if pm.CurrentIdx != -1 {
		t.Fatalf("current index=%d, want -1", pm.CurrentIdx)
	}
	if pm.SongStarted {
		t.Fatal("song cannot stay started without now playing")
	}
	if pm.ShuffleHist != nil {
		t.Fatalf("shuffle history=%v, want nil", pm.ShuffleHist)
	}
}

func TestSetPlaylistReconcilesCurrentTrackIndex(t *testing.T) {
	pm := NewPlaybackManager()
	pm.NowPlay = &domain.Track{ID: "b", Title: "B"}
	pm.CurrentIdx = 7

	pm.Update(SetPlaylistMsg{
		Tracks: []domain.Track{
			{ID: "a", Title: "A"},
			{ID: "b", Title: "B"},
			{ID: "c", Title: "C"},
		},
	})

	if pm.CurrentIdx != 1 {
		t.Fatalf("current index=%d, want 1", pm.CurrentIdx)
	}

	pm.Update(SetPlaylistMsg{
		Tracks: []domain.Track{
			{ID: "x", Title: "X"},
			{ID: "y", Title: "Y"},
		},
	})

	if pm.CurrentIdx != -1 {
		t.Fatalf("current index after missing track=%d, want -1", pm.CurrentIdx)
	}
}
