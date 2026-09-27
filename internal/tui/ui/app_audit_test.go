package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"som/internal/domain"
	"som/internal/storage"
)

func TestApplyMoveToPlaylistUsesPathIDs(t *testing.T) {
	db, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	path := t.TempDir() + "/song.mp3"
	if err := db.UpsertLocalFile(storage.LocalFile{Path: path, Name: "Song", Duration: 180}); err != nil {
		t.Fatal(err)
	}

	pl, err := db.CreatePlaylist("Test")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AddTrackToPlaylist(pl.ID, path); err != nil {
		t.Fatal(err)
	}

	a := &App{
		left: LeftPanel{
			plStore: db,
			playlists: []storage.Playlist{{
				ID:   pl.ID,
				Name: pl.Name,
				Tracks: []storage.PlaylistTrack{{
					ID: path, Path: path, Title: "Song", Duration: 180,
				}},
			}},
			locals: []LocalFile{{Path: path, Name: "Song", Duration: 180}},
		},
		moveSession: &MoveSession{
			TargetPlIdx: 0,
			Selected:    map[string]bool{path: true},
		},
	}

	a.applyMoveToPlaylist()
	if got := len(a.left.playlists[0].Tracks); got != 0 {
		t.Fatalf("expected track to be removed, got %d tracks", got)
	}

	a.applyMoveToPlaylist()
	if got := len(a.left.playlists[0].Tracks); got != 1 {
		t.Fatalf("expected track to be added back, got %d tracks", got)
	}
	if got := a.left.playlists[0].Tracks[0].ID; got != path {
		t.Fatalf("track ID=%q, want path %q", got, path)
	}
}

func TestHandleDataEventsIgnoresInvalidPlaylistIndex(t *testing.T) {
	db, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	a := &App{
		left: LeftPanel{
			plStore:    db,
			playlists:  []storage.Playlist{{ID: "pl_test", Name: "Test"}},
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("invalid playlist index panicked: %v", r)
		}
	}()

	a.handleDataEvents(ExecuteRemovePlMsg{PlIdx: 99})
}

func TestHandleAudioEventsIgnoresInvalidPlaylistIndex(t *testing.T) {
	a := &App{}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("invalid track index panicked: %v", r)
		}
	}()

	a.handleAudioEvents(PlayPlaylistMsg{
		Tracks: nil,
		Index:  0,
	})
}


func TestHandleAudioEventsIgnoresStaleGeneration(t *testing.T) {
	a := &App{
		playback: &PlaybackManager{PlayerGen: 5},
	}

	if cmd := a.handleAudioEvents(TrackChangedMsg{
		Track: domain.Track{ID: "stale", Title: "Stale"},
		Gen:   4,
	}); cmd != nil {
		t.Fatal("stale track change should not schedule a command")
	}
	if a.statusMsg != "" {
		t.Fatalf("status=%q, want empty", a.statusMsg)
	}

	if cmd := a.handleAudioEvents(StreamResolvedMsg{
		Gen: 4,
		Err: fmt.Errorf("stale stream failure"),
	}); cmd != nil {
		t.Fatal("stale stream result should not schedule a command")
	}
	if a.statusMsg != "" {
		t.Fatalf("status after stale stream=%q, want empty", a.statusMsg)
	}
}


func TestDeleteCmdReportsFilesystemFailure(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "keep"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	msg := deleteCmd(nil, nested, "nested")()
	res, ok := msg.(DeleteDoneMsg)
	if !ok {
		t.Fatalf("message=%T, want DeleteDoneMsg", msg)
	}
	if res.Err == nil {
		t.Fatal("expected filesystem deletion error")
	}
}
