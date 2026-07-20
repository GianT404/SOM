package ui

import (
	"som/internal/tui/api"
	"time"
)

type SearchResultMsg struct {
	Tracks []api.Track
	Err    error
}

type PlayStartedMsg struct{ Track api.Track }

type PlayLocalMsg struct {
	Path  string
	Title string
}
type DownloadDoneMsg struct {
	Path string
	Err  error
}
type StreamStartedMsg struct {
	Track     api.Track
	PlayedAt  time.Time
	Lyrics    api.LyricsResp
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
	// VideoID is the YouTube video ID this file was downloaded from,
	// read back from the accompanying .json metadata (empty if unknown,
	// e.g. for files predating this field).
	VideoID string
}

type Pane int

const (
	PaneLeft Pane = iota
	PaneRight
)
