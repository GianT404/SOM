package ui

import (
	"testing"

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
			},
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
