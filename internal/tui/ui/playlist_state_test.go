package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"som/internal/storage"
)

func testPlaylists() []storage.Playlist {
	return []storage.Playlist{
		{ID: "pl_alpha", Name: "Alpha"},
		{ID: "pl_beta", Name: "Beta"},
	}
}

func TestRebindActivePlaylistAfterSliceChange(t *testing.T) {
	p := LeftPanel{playlists: make([]storage.Playlist, 1, 1)}
	p.playlists[0] = storage.Playlist{ID: "pl_alpha", Name: "Alpha"}
	p.activePlaylist = &p.playlists[0]

	p.playlists = append(p.playlists, storage.Playlist{ID: "pl_beta", Name: "Beta"})
	p.rebindActivePlaylist()

	if p.activePlaylist == nil {
		t.Fatal("active playlist should stay selected")
	}
	if p.activePlaylist.ID != "pl_alpha" {
		t.Fatalf("active playlist=%q, want %q", p.activePlaylist.ID, "pl_alpha")
	}
	if p.activePlaylist != &p.playlists[0] {
		t.Fatal("active playlist should point into the current slice")
	}
}

func TestRebindActivePlaylistClearsRemovedSelection(t *testing.T) {
	p := LeftPanel{playlists: testPlaylists()}
	p.activePlaylist = &p.playlists[0]
	p.plCursor = 4
	p.plOffset = 2

	p.playlists = []storage.Playlist{{ID: "pl_beta", Name: "Beta"}}
	p.rebindActivePlaylist()

	if p.activePlaylist != nil {
		t.Fatal("removed playlist should clear active selection")
	}
	if p.plCursor != 0 || p.plOffset != 0 {
		t.Fatalf("cursor state=%d/%d, want 0/0", p.plCursor, p.plOffset)
	}
}

func TestPlaylistFilterClampsCursor(t *testing.T) {
	p := LeftPanel{
		playlists:  testPlaylists(),
		activeTab: SidePlaylists,
		height:     30,
	}
	p.input.Focus()
	p.input.SetValue("alpha")
	p.plCursor = 5
	p.plOffset = 4

	p.Update(tea.WindowSizeMsg{Width: 80, Height: 30}, true, nil)

	if got := len(p.getFilteredPlaylists()); got != 1 {
		t.Fatalf("filtered playlists=%d, want 1", got)
	}
	if p.plCursor != 0 {
		t.Fatalf("cursor=%d, want 0", p.plCursor)
	}
	if p.plOffset != 0 {
		t.Fatalf("offset=%d, want 0", p.plOffset)
	}
}

func TestPlaylistFilterRestoresPreviousSelection(t *testing.T) {
	p := LeftPanel{
		playlists:      testPlaylists(),
		activeTab:      SidePlaylists,
		height:         30,
		plPreFilterID:  "pl_beta",
	}
	p.input.Focus()
	p.input.SetValue("b")
	p.plCursor = 0

	p.Update(tea.KeyPressMsg{Code: tea.KeyBackspace}, true, nil)

	if p.plCursor != 1 {
		t.Fatalf("cursor=%d, want 1", p.plCursor)
	}
	if p.plPreFilterID != "" {
		t.Fatalf("prefilter ID=%q, want empty", p.plPreFilterID)
	}
}
