package ui

import "som-tui/internal/tui/api"

type SearchResultMsg struct {
	Tracks []api.Track
	Err    error
}

type PlayStartedMsg struct{ Track api.Track }

type PlayLocalMsg struct {
	Path  string
	Title string
}
type LyricsLoadedMsg struct {
	Lyrics api.LyricsResp
	Err    error
}
type DownloadDoneMsg struct {
	Path string
	Err  error
}
type LocalFilesMsg struct {
	Files []LocalFile
}

type LocalFile struct {
	Name string
	Path string
}

type Pane int

const (
	PaneLeft Pane = iota
	PaneRight
)
