package storage

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func testDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	conn, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestPlaylistCRUD(t *testing.T) {
	db := testDB(t)

	pl, err := db.CreatePlaylist("My Playlist")
	if err != nil {
		t.Fatal(err)
	}
	if pl.Name != "My Playlist" {
		t.Fatalf("expected name 'My Playlist', got %q", pl.Name)
	}

	playlists, err := db.ListPlaylists()
	if err != nil {
		t.Fatal(err)
	}
	if len(playlists) != 1 {
		t.Fatalf("expected 1 playlist, got %d", len(playlists))
	}

	err = db.AddTrackToPlaylist(pl.ID, PlaylistTrack{
		ID:       "abc123",
		Title:    "Test Song",
		Artist:   "Test Artist",
		Duration: 180,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = db.AddTrackToPlaylist(pl.ID, PlaylistTrack{
		ID:    "abc123",
		Title: "Test Song",
	})
	if err == nil {
		t.Fatal("expected duplicate error")
	}

	tracks, err := db.GetPlaylistTracks(pl.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(tracks))
	}
	if tracks[0].Title != "Test Song" {
		t.Fatalf("expected title 'Test Song', got %q", tracks[0].Title)
	}

	err = db.RemoveTrackFromPlaylist(pl.ID, "abc123")
	if err != nil {
		t.Fatal(err)
	}

	tracks, _ = db.GetPlaylistTracks(pl.ID)
	if len(tracks) != 0 {
		t.Fatalf("expected 0 tracks, got %d", len(tracks))
	}

	err = db.DeletePlaylist(pl.ID)
	if err != nil {
		t.Fatal(err)
	}

	playlists, _ = db.ListPlaylists()
	if len(playlists) != 0 {
		t.Fatalf("expected 0 playlists, got %d", len(playlists))
	}
}

func TestLocalFiles(t *testing.T) {
	db := testDB(t)

	err := db.UpsertLocalFile(LocalFile{
		Path:     "/tmp/test.opus",
		Name:     "Test Song",
		Artist:   "Artist",
		Duration: 200,
		VideoID:  "dQw4w9WgXcQ",
	})
	if err != nil {
		t.Fatal(err)
	}

	files, err := db.ListAllLocalFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Name != "Test Song" {
		t.Fatalf("expected name 'Test Song', got %q", files[0].Name)
	}

	f, err := db.GetLocalFile("/tmp/test.opus")
	if err != nil {
		t.Fatal(err)
	}
	if f == nil {
		t.Fatal("expected file, got nil")
	}

	err = db.UpsertLocalFile(LocalFile{
		Path:     "/tmp/test.opus",
		Name:     "Updated Song",
		Artist:   "New Artist",
		Duration: 250,
		VideoID:  "dQw4w9WgXcQ",
	})
	if err != nil {
		t.Fatal(err)
	}
	f, _ = db.GetLocalFile("/tmp/test.opus")
	if f.Name != "Updated Song" {
		t.Fatalf("expected updated name, got %q", f.Name)
	}

	if !db.IsDownloaded("dQw4w9WgXcQ", "", 0) {
		t.Fatal("expected IsDownloaded true for video ID")
	}

	err = db.DeleteLocalFile("/tmp/test.opus")
	if err != nil {
		t.Fatal(err)
	}
	f, _ = db.GetLocalFile("/tmp/test.opus")
	if f != nil {
		t.Fatal("expected nil after delete")
	}
}

func TestFTSSearch(t *testing.T) {
	db := testDB(t)

	db.UpsertLocalFile(LocalFile{Path: "/a.opus", Name: "Bohemian Rhapsody", Artist: "Queen", VideoID: "v1"})
	db.UpsertLocalFile(LocalFile{Path: "/b.opus", Name: "Stairway to Heaven", Artist: "Led Zeppelin", VideoID: "v2"})
	db.UpsertLocalFile(LocalFile{Path: "/c.opus", Name: "Hotel California", Artist: "Eagles", VideoID: "v3"})

	files, err := db.SearchLocalFiles("bohemian")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Name != "Bohemian Rhapsody" {
		t.Fatalf("expected Bohemian Rhapsody, got %v", files)
	}

	files, err = db.SearchLocalFiles("led zeppelin")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Name != "Stairway to Heaven" {
		t.Fatalf("expected Stairway to Heaven, got %v", files)
	}

	files, err = db.SearchLocalFiles("hotel california")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Name != "Hotel California" {
		t.Fatalf("expected Hotel California, got %v", files)
	}

	files, err = db.SearchLocalFiles("")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}
}

func TestRenameLocalFile(t *testing.T) {
	db := testDB(t)

	db.UpsertLocalFile(LocalFile{Path: "/old.opus", Name: "Old Name", VideoID: "v1"})

	pl, _ := db.CreatePlaylist("Test")
	db.AddTrackToPlaylist(pl.ID, PlaylistTrack{ID: "local:/old.opus", Title: "Old Name", IsLocal: true})

	err := db.RenameLocalFile("/old.opus", "/new.opus", "New Name")
	if err != nil {
		t.Fatal(err)
	}

	f, _ := db.GetLocalFile("/new.opus")
	if f == nil || f.Name != "New Name" {
		t.Fatalf("expected updated file, got %v", f)
	}

	tracks, _ := db.GetPlaylistTracks(pl.ID)
	if len(tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(tracks))
	}
	if tracks[0].ID != "local:/new.opus" {
		t.Fatalf("expected track ID 'local:/new.opus', got %q", tracks[0].ID)
	}
}
