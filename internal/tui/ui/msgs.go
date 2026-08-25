package ui

import (
	"som/internal/domain"
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
	Path string
	Err  error
}
type StreamStartedMsg struct {
	Track     domain.Track
	Lyrics    domain.LyricsResp
	LyricsErr error
	Err       error
}
type LocalFilesMsg struct {
	Files []LocalFile
}

type LocalFile struct {
	Name     string
	Path     string
	Artist   string
	Duration int
	VideoID  string
}

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
