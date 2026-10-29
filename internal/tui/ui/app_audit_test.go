package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/bubbles/v2/spinner"
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
			plStore:   db,
			playlists: []storage.Playlist{{ID: "pl_test", Name: "Test"}},
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

// Filesystem lỗi phải được trả về cho UI.
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

func TestSpinnerKeepsTickingWhileLoadingMore(t *testing.T) {
	a := &App{
		left: LeftPanel{
			loadingMore: true,
			spinner:     spinner.New(),
		},
	}

	cmds, handled, stop := a.updateSystem(spinner.TickMsg{})
	if !handled || stop {
		t.Fatalf("updateSystem handled=%v stop=%v, want handled=true stop=false", handled, stop)
	}
	if len(cmds) == 0 || cmds[0] == nil {
		t.Fatal("spinner tick should schedule the next animation tick while loading more")
	}
}


func TestApplyMoveToPlaylistAddOnlySkipsExistingTracks(t *testing.T) {
	db, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	existingPath := filepath.Join(t.TempDir(), "existing.mp3")
	newPath := filepath.Join(t.TempDir(), "new.mp3")
	for _, path := range []string{existingPath, newPath} {
		if err := db.UpsertLocalFile(storage.LocalFile{Path: path, Name: filepath.Base(path), Duration: 180}); err != nil {
			t.Fatal(err)
		}
	}

	pl, err := db.CreatePlaylist("Add only")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AddTrackToPlaylist(pl.ID, existingPath); err != nil {
		t.Fatal(err)
	}

	a := &App{
		left: LeftPanel{
			plStore: db,
			playlists: []storage.Playlist{{
				ID: pl.ID,
				Name: pl.Name,
				Tracks: []storage.PlaylistTrack{{
					ID: existingPath, Path: existingPath, Title: "Existing",
				}},
			}},
			locals: []LocalFile{
				{Path: existingPath, Name: "Existing", Duration: 180},
				{Path: newPath, Name: "New", Duration: 180},
			},
		},
		moveSession: &MoveSession{
			TargetPlIdx: 0,
			Selected: map[string]bool{
				existingPath: true,
				newPath:      true,
			},
			AddOnly: true,
		},
	}

	a.applyMoveToPlaylist()

	tracks := a.left.playlists[0].Tracks
	if len(tracks) != 2 {
		t.Fatalf("track count=%d, want 2", len(tracks))
	}
	if tracks[0].ID != existingPath || tracks[1].ID != newPath {
		t.Fatalf("track IDs=%q, %q; want existing then new", tracks[0].ID, tracks[1].ID)
	}
	if !strings.Contains(a.statusMsg, "Added 1 track") {
		t.Fatalf("status=%q, want one added track", a.statusMsg)
	}
}

func TestCmdOptionListIncludesAddTracksForActivePlaylist(t *testing.T) {
	playlist := storage.Playlist{ID: "pl_test", Name: "Test"}
	a := &App{
		sidebarActive: SidePlaylists,
		left: LeftPanel{
			activePlaylist: &playlist,
			playlists:      []storage.Playlist{playlist},
		},
	}

	for _, option := range a.cmdOptionList() {
		if option == "Add tracks to playlist" {
			return
		}
	}
	t.Fatal("Add tracks to playlist option not available inside an active playlist")
}
