package ui

import (
	"testing"

	"som/internal/domain"
	"som/internal/storage"
)

func TestRequestDownloadRejectsAlreadyDownloadedTrack(t *testing.T) {
	tests := []struct {
		name   string
		stored storage.LocalFile
		track  domain.Track
	}{
		{
			name: "matching video ID",
			stored: storage.LocalFile{
				Name: "Previously Downloaded",
				VideoID: "video-123",
				Duration: 180,
			},
			track: domain.Track{
				ID: "video-123",
				Title: "Previously Downloaded",
				Duration: 180,
			},
		},
		{
			name: "matching title and duration",
			stored: storage.LocalFile{
				Name: "Same Song",
				VideoID: "another-video",
				Duration: 180,
			},
			track: domain.Track{
				ID: "current-search-result",
				Title: " same song ",
				Duration: 181,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			db, err := storage.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			tt.stored.Path = dir + "/existing.opus"
			if err := db.UpsertLocalFile(tt.stored); err != nil {
				t.Fatal(err)
			}

			// Leave locals empty to verify the storage check catches a previously
			// downloaded track even if the in-memory list has not been refreshed.
			panel := LeftPanel{
				plStore: db,
				downloadDir: dir,
				provider: nil,
			}

			cmd := panel.requestDownload(tt.track)
			if cmd == nil {
				t.Fatal("expected an error message command")
			}

			msg, ok := cmd().(DownloadDoneMsg)
			if !ok {
				t.Fatalf("message type = %T, want DownloadDoneMsg", cmd())
			}
			if msg.Err == nil || msg.Err.Error() != errTrackAlreadyDownloaded.Error() {
				t.Fatalf("error = %v, want %q", msg.Err, errTrackAlreadyDownloaded)
			}
			if panel.loadingDownload {
				t.Fatal("loadingDownload should remain false for a rejected duplicate")
			}
		})
	}
}
