package ui

import (
	"som/internal/domain"
	"som/internal/storage"
)

type SearchResultMsg struct {
	Tracks []domain.Track
	Err    error
}

type SuggestDebounceMsg struct {
	Query string
}

type SuggestionsMsg struct {
	Query string
	Items []string
	Err   error
}

type PlayStartedMsg struct{ Track domain.Track }

type PlayLocalMsg struct {
	Path  string
	Title string
}
type DownloadDoneMsg struct {
	Path  string
	Err   error
	Track domain.Track
}
type StreamResolvedMsg struct {
	Lyrics    domain.LyricsResp
	LyricsErr error
	Err       error
	Gen       uint64
}
type LocalFilesMsg struct {
	Files []LocalFile
}

type LocalFile struct {
	Name      string
	Path      string
	Artist    string
	Duration  int
	VideoID   string
	Thumbnail string
	FileSize  int64
	FileMTime string
	CreatedAt string
}

type CloseModalMsg struct{}

type Pane int

const (
	PaneLeft Pane = iota
	PaneRight
)

type PlayPlaylistMsg struct {
	Tracks []domain.Track
	Index  int
}

type PlayQueueMsg struct {
	Index int
}

type RemoveFromQueueMsg struct {
	Index int
}

type InitMoveSessionMsg struct {
	TargetPlIdx int    // >= 0: Chọn playlist có sẵn
	NewPlName   string // TargetPlIdx == -1: Tạo playlist mới
}

type ExecuteMoveMsg struct{}

type ExecuteCmdOptionMsg struct {
	Option string
}

type ExecuteRemovePlMsg struct {
	PlIdx int
	Track storage.PlaylistTrack
}

type PlayNextMsg struct{}
type PlayPrevMsg struct{}
type PlayTrackAtMsg struct {
	Index int
	Track domain.Track
}
type TogglePauseMsg struct{}
type PlaybackStateChangedMsg struct {
	State int
}
type PlaybackErrorMsg struct {
	Err error
}
type TrackChangedMsg struct {
	Track       domain.Track
	IsLocal     bool
	Gen         uint64
	PlaylistPos int
	PlaylistLen int
	IsRandom    bool
}

type LocalLyricsLoadedMsg struct {
	Lyrics domain.LyricsResp
}

type PlaybackTickMsg struct{}
type CloseAllModalsMsg struct{}

type SetPlaylistMsg struct {
	Tracks []domain.Track
}
type EnqueueTrackMsg struct {
	Track domain.Track
}
type ToggleRandomMsg struct{}
type QueueChangedMsg struct {
	Queue []domain.Track
}

type OpenSettingsMsg struct{}
type OpenHelpMsg struct{}
type ApplySettingMsg struct {
	Index int
	Value bool
}
