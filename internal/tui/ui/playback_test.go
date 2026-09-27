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

func TestPrevRandomDoesNotGrowHistory(t *testing.T) {
	pm := NewPlaybackManager()
	pm.Player = &player.Player{}
	pm.Random = true
	pm.Playlist = []domain.Track{
		{ID: "a", Title: "A"},
		{ID: "b", Title: "B"},
		{ID: "c", Title: "C"},
	}
	pm.CurrentIdx = 2
	pm.NowPlay = &pm.Playlist[2]
	pm.History = []domain.Track{pm.Playlist[0], pm.Playlist[1]}

	pm.Update(PlayPrevMsg{})

	if got := len(pm.History); got != 1 {
		t.Fatalf("history length=%d, want 1", got)
	}
	if pm.NowPlay == nil || pm.NowPlay.ID != "b" {
		t.Fatalf("now playing=%v, want b", pm.NowPlay)
	}
	if pm.CurrentIdx != 1 {
		t.Fatalf("current index=%d, want 1", pm.CurrentIdx)
	}

	pm.Update(PlayPrevMsg{})
	if got := len(pm.History); got != 0 {
		t.Fatalf("history length after second prev=%d, want 0", got)
	}
	if pm.NowPlay == nil || pm.NowPlay.ID != "a" {
		t.Fatalf("now playing after second prev=%v, want a", pm.NowPlay)
	}
	if pm.CurrentIdx != 0 {
		t.Fatalf("current index after second prev=%d, want 0", pm.CurrentIdx)
	}
}

func TestQueuePlaybackPreservesPlaylistCursor(t *testing.T) {
	pm := NewPlaybackManager()
	pm.Player = &player.Player{}
	pm.Playlist = []domain.Track{
		{ID: "a", Title: "A"},
		{ID: "b", Title: "B"},
		{ID: "c", Title: "C"},
	}
	pm.CurrentIdx = 1
	pm.NowPlay = &pm.Playlist[1]
	pm.Queue = []domain.Track{{ID: "queued", Title: "Queued"}}

	pm.Update(PlayNextMsg{})

	if pm.NowPlay == nil || pm.NowPlay.ID != "queued" {
		t.Fatalf("now playing=%v, want queued", pm.NowPlay)
	}
	if pm.CurrentIdx != 1 {
		t.Fatalf("current index=%d, want 1", pm.CurrentIdx)
	}

	next, idx, fromQueue := pm.NextTrack()
	if next == nil || next.ID != "c" {
		t.Fatalf("next=%v, want c", next)
	}
	if idx != 2 || fromQueue {
		t.Fatalf("next state=%d/%v, want 2/false", idx, fromQueue)
	}
}

func TestDeleteCurrentTrackKeepsNextPlaylistPosition(t *testing.T) {
	pm := NewPlaybackManager()
	pm.Playlist = []domain.Track{
		{ID: "local:a", Title: "A"},
		{ID: "local:b", Title: "B"},
		{ID: "local:c", Title: "C"},
	}
	pm.CurrentIdx = 1
	pm.NowPlay = &pm.Playlist[1]
	pm.SongStarted = true

	pm.Update(DeleteDoneMsg{Path: "b"})

	if pm.NowPlay != nil {
		t.Fatalf("now playing=%v, want nil", pm.NowPlay)
	}
	if pm.SongStarted {
		t.Fatal("song should be stopped")
	}
	if pm.CurrentIdx != 0 {
		t.Fatalf("current index=%d, want 0", pm.CurrentIdx)
	}
	next, idx, fromQueue := pm.NextTrack()
	if next == nil || next.ID != "local:c" || idx != 1 || fromQueue {
		t.Fatalf("next=%v idx=%d queue=%v, want local:c 1 false", next, idx, fromQueue)
	}
}

func TestDeleteTrackClearsBufferedNextAndHistory(t *testing.T) {
	pm := NewPlaybackManager()
	pm.Playlist = []domain.Track{
		{ID: "local:a", Title: "A"},
		{ID: "local:b", Title: "B"},
		{ID: "local:c", Title: "C"},
	}
	pm.CurrentIdx = 0
	pm.NowPlay = &pm.Playlist[0]
	pm.NextPlay = &pm.Playlist[1]
	pm.History = []domain.Track{pm.Playlist[1], pm.Playlist[2]}
	pm.ShuffleHist = []int{1, 2}

	pm.Update(DeleteDoneMsg{Path: "b"})

	if pm.NextPlay != nil {
		t.Fatalf("next play=%v, want nil", pm.NextPlay)
	}
	if len(pm.History) != 1 || pm.History[0].ID != "local:c" {
		t.Fatalf("history=%v, want [local:c]", pm.History)
	}
	if len(pm.ShuffleHist) != 0 {
		t.Fatalf("shuffle history=%v, want empty", pm.ShuffleHist)
	}
}

func TestRenameUpdatesAllPlaybackReferences(t *testing.T) {
	pm := NewPlaybackManager()
	pm.Playlist = []domain.Track{{ID: "local:old.mp3", Title: "Old"}}
	pm.Queue = []domain.Track{{ID: "local:old.mp3", Title: "Old"}}
	pm.History = []domain.Track{{ID: "local:old.mp3", Title: "Old"}}
	pm.NextPlay = &domain.Track{ID: "local:old.mp3", Title: "Old"}
	pm.NowPlay = &domain.Track{ID: "local:old.mp3", Title: "Old"}
	pm.CurrentIdx = 0
	pm.PlayerGen = 7

	pm.Update(RenameDoneMsg{OldPath: "old.mp3", NewPath: "new.mp3", NewTitle: "New"})

	for name, track := range map[string]*domain.Track{
		"playlist": &pm.Playlist[0],
		"queue":    &pm.Queue[0],
		"history":  &pm.History[0],
		"next":     pm.NextPlay,
		"now":      pm.NowPlay,
	} {
		if track == nil {
			t.Fatalf("%s track is nil", name)
		}
		if track.ID != "local:new.mp3" || track.Title != "New" {
			t.Fatalf("%s=%+v, want renamed track", name, *track)
		}
	}
}

func TestNewPlayRequestInvalidatesPreviousGeneration(t *testing.T) {
	pm := NewPlaybackManager()
	pm.Player = &player.Player{}
	pm.NowPlay = &domain.Track{ID: "old"}

	pm.playTrackCmd(0, domain.Track{ID: "new-1"})
	first := pm.PlayerGen
	pm.playTrackCmd(0, domain.Track{ID: "new-2"})
	second := pm.PlayerGen

	if first == second {
		t.Fatalf("playback generations=%d/%d, want different generations", first, second)
	}
	if second <= first {
		t.Fatalf("playback generations=%d/%d, want monotonic increase", first, second)
	}
}

func TestPlaybackStopClearsState(t *testing.T) {
	pm := NewPlaybackManager()
	pm.Player = &player.Player{}
	pm.NowPlay = &domain.Track{ID: "current"}
	pm.NextPlay = &domain.Track{ID: "next"}
	pm.SongStarted = true

	cmd := pm.Stop()
	if cmd == nil {
		t.Fatal("expected stop notification command")
	}
	if pm.NowPlay != nil || pm.NextPlay != nil || pm.SongStarted {
		t.Fatalf("state after stop=%v/%v/%v, want nil/nil/false", pm.NowPlay, pm.NextPlay, pm.SongStarted)
	}
	if pm.PlayerGen != 1 {
		t.Fatalf("player generation=%d, want 1 for fresh test player", pm.PlayerGen)
	}
}
