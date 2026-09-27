package ui

import (
	"testing"

	"som/internal/domain"
	"som/internal/tui/player"
)

func TestPlayTrackCmdRecordsPreviousTrackOnceInRandomMode(t *testing.T) {
	pm := NewPlaybackManager()
	pm.Player = &player.Player{}
	pm.Random = true

	previous := domain.Track{ID: "track-1", Title: "Previous"}
	current := domain.Track{ID: "track-2", Title: "Current"}
	pm.NowPlay = &previous

	if cmd := pm.playTrackCmd(1, current); cmd == nil {
		t.Fatal("expected play command")
	}

	if len(pm.History) != 1 {
		t.Fatalf("history length=%d, want 1", len(pm.History))
	}
	if pm.History[0].ID != previous.ID {
		t.Fatalf("history[0]=%q, want %q", pm.History[0].ID, previous.ID)
	}
	if pm.NowPlay == nil || pm.NowPlay.ID != current.ID {
		t.Fatalf("now playing=%v, want %q", pm.NowPlay, current.ID)
	}
}
