package ui

import (
	"encoding/json"
	"os"

	"som/internal/domain"
	"som/internal/storage"

	tea "charm.land/bubbletea/v2"
)

func saveLocalMetaCmd(p domain.MusicProvider, store *storage.DB, path string, t domain.Track) tea.Cmd {
	return func() tea.Msg {
		info, _ := os.Stat(path)
		fileSize, fileMTime := int64(0), ""
		if info != nil {
			fileSize = info.Size()
			fileMTime = info.ModTime().Format("2006-01-02 15:04:05")
		}

		// Tác vụ mạng chạy ngầm, không block UI
		lr, _ := getCachedLyrics(p, store, t.ID, t.Title, t.Artist, t.Duration)
		lrJSON, _ := json.Marshal(lr)

		err := store.UpsertLocalFileWithMeta(storage.LocalFile{
			Path:      path,
			Name:      t.Title,
			Artist:    t.Artist,
			Duration:  t.Duration,
			VideoID:   t.ID,
			Thumbnail: t.Thumbnail,
			FileSize:  fileSize,
			FileMTime: fileMTime,
		}, &storage.LocalFileMeta{
			Artist:     t.Artist,
			Title:      t.Title,
			VideoID:    t.ID,
			Thumbnail:  t.Thumbnail,
			LyricsJSON: string(lrJSON),
		})

		return MetaSavedMsg{Path: path, Track: t, Err: err}
	}
}
