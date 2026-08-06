package playlist

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	return &Store{filePath: filepath.Join(dir, "playlists.json")}
}

func TestStore_SaveLoadRoundTrip(t *testing.T) {
	s := newTestStore(t)

	want := []Playlist{{ID: "pl_1", Name: "Test", Tracks: []Track{{ID: "t1", Title: "Song A"}}}}
	if err := s.Save(want); err != nil {
		t.Fatalf("Save loi: %v", err)
	}

	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load loi: %v", err)
	}
	if len(got) != 1 || got[0].ID != "pl_1" || len(got[0].Tracks) != 1 {
		t.Fatalf("du lieu doc lai khong khop, got=%+v", got)
	}
}

func TestStore_Load_MissingFileReturnsEmpty(t *testing.T) {
	s := newTestStore(t)

	got, err := s.Load()
	if err != nil {
		t.Fatalf("file chua ton tai khong duoc tra loi, got err=%v", err)
	}
	if len(got) != 0 {
		t.Fatalf("phai tra ve slice rong, got=%+v", got)
	}
}

func TestStore_Save_NoTempFileLeftBehindAfterSuccess(t *testing.T) {
	s := newTestStore(t)

	if err := s.Save([]Playlist{{ID: "pl_1", Name: "test"}}); err != nil {
		t.Fatalf("Save loi: %v", err)
	}

	dir := filepath.Dir(s.filePath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("khong doc duoc thu muc: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Fatalf("con sot file tam sau khi save thanh cong: %s", e.Name())
		}
	}
}

func TestStore_CreatePlaylist_ThenAddTrack(t *testing.T) {
	s := newTestStore(t)

	pl, err := s.CreatePlaylist("Nhac chill")
	if err != nil {
		t.Fatalf("CreatePlaylist loi: %v", err)
	}

	if err := s.AddTrack(pl.ID, Track{ID: "t1", Title: "Song A"}); err != nil {
		t.Fatalf("AddTrack loi: %v", err)
	}

	playlists, _ := s.Load()
	if len(playlists) != 1 || len(playlists[0].Tracks) != 1 {
		t.Fatalf("track khong duoc them dung, got=%+v", playlists)
	}
}

func TestStore_AddTrack_RejectsDuplicate(t *testing.T) {
	s := newTestStore(t)
	pl, _ := s.CreatePlaylist("test")
	s.AddTrack(pl.ID, Track{ID: "t1"})

	err := s.AddTrack(pl.ID, Track{ID: "t1"})
	if err == nil {
		t.Fatal("them trung ID phai tra ve loi")
	}
}

func TestStore_RemoveTrack(t *testing.T) {
	s := newTestStore(t)
	pl, _ := s.CreatePlaylist("test")
	s.AddTrack(pl.ID, Track{ID: "t1"})
	s.AddTrack(pl.ID, Track{ID: "t2"})

	if err := s.RemoveTrack(pl.ID, "t1"); err != nil {
		t.Fatalf("RemoveTrack loi: %v", err)
	}

	playlists, _ := s.Load()
	if len(playlists[0].Tracks) != 1 || playlists[0].Tracks[0].ID != "t2" {
		t.Fatalf("sau khi xoa t1, chi con t2, got=%+v", playlists[0].Tracks)
	}
}

func TestStore_DeletePlaylist(t *testing.T) {
	s := newTestStore(t)
	pl, _ := s.CreatePlaylist("se bi xoa")

	if err := s.DeletePlaylist(pl.ID); err != nil {
		t.Fatalf("DeletePlaylist loi: %v", err)
	}

	playlists, _ := s.Load()
	if len(playlists) != 0 {
		t.Fatalf("playlist phai bi xoa, got=%+v", playlists)
	}
}

func TestStore_ConcurrentAddTrack_NoLostUpdates(t *testing.T) {
	s := newTestStore(t)
	pl, err := s.CreatePlaylist("concurrent test")
	if err != nil {
		t.Fatalf("CreatePlaylist loi: %v", err)
	}

	const n = 50
	var wg sync.WaitGroup
	errs := make(chan error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			track := Track{ID: "track-" + strconv.Itoa(i), Title: "bai " + strconv.Itoa(i)}
			if err := s.AddTrack(pl.ID, track); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("AddTrack that bai trong luc chay dong thoi: %v", err)
	}

	playlists, err := s.Load()
	if err != nil {
		t.Fatalf("Load loi: %v", err)
	}
	if len(playlists) != 1 {
		t.Fatalf("phai chi co 1 playlist, got %d", len(playlists))
	}
	if got := len(playlists[0].Tracks); got != n {
		t.Fatalf("mat du lieu do race: muon %d track, thuc te con %d", n, got)
	}
}
