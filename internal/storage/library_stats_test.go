package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetLibraryStats(t *testing.T) {
	downloadDir := t.TempDir()
	importDir := t.TempDir()

	downloadPath := filepath.Join(downloadDir, "download.opus")
	importPath := filepath.Join(importDir, "imported.mp3")
	if err := os.WriteFile(downloadPath, []byte("download-audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(importPath, []byte("import-audio-data"), 0o600); err != nil {
		t.Fatal(err)
	}

	db, err := Open(downloadDir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.UpsertLocalFile(LocalFile{Path: downloadPath, Name: "Downloaded"}); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertLocalFile(LocalFile{Path: importPath, Name: "Imported"}); err != nil {
		t.Fatal(err)
	}

	favorites, err := db.CreatePlaylist("Favorites")
	if err != nil {
		t.Fatal(err)
	}
	mix, err := db.CreatePlaylist("Mix")
	if err != nil {
		t.Fatal(err)
	}
	for _, trackPath := range []string{downloadPath, importPath} {
		if err := db.AddTrackToPlaylist(favorites.ID, trackPath); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.AddTrackToPlaylist(mix.ID, downloadPath); err != nil {
		t.Fatal(err)
	}

	stats, err := db.GetLibraryStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.DownloadedTracks != 1 || stats.ImportedTracks != 1 {
		t.Fatalf("unexpected track counts: downloaded=%d imported=%d", stats.DownloadedTracks, stats.ImportedTracks)
	}
	if got, want := stats.DownloadedBytes, int64(len("download-audio")); got != want {
		t.Fatalf("downloaded bytes = %d, want %d", got, want)
	}
	if got, want := stats.ImportedBytes, int64(len("import-audio-data")); got != want {
		t.Fatalf("imported bytes = %d, want %d", got, want)
	}
	if stats.PlaylistCount != 2 || stats.PlaylistTrackCount != 3 {
		t.Fatalf("unexpected playlist stats: playlists=%d track entries=%d", stats.PlaylistCount, stats.PlaylistTrackCount)
	}
	if len(stats.Playlists) != 2 {
		t.Fatalf("got %d playlist details, want 2", len(stats.Playlists))
	}
}
