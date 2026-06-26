// internal/ui/msgs.go
// All custom Bubble Tea Msg types used across screens.
package ui

import "github.com/GianT404/SOM/tui/internal/api"

// ─── Search ─────────────────────────────────────────────────────────────────

type SearchResultMsg struct {
	Tracks []api.Track
	Err    error
}

// ─── Playback ────────────────────────────────────────────────────────────────

type PlayStartedMsg struct{ Track api.Track }
type PlayErrorMsg struct{ Err error }
type PlayToggledMsg struct{}

// ─── Lyrics ──────────────────────────────────────────────────────────────────

type LyricsLoadedMsg struct {
	Lyrics api.LyricsResp
	Err    error
}

// ─── Download ────────────────────────────────────────────────────────────────

type DownloadDoneMsg struct {
	Path string
	Err  error
}

type DownloadProgressMsg struct {
	Done  int64
	Total int64 // -1 if unknown
}

// ─── Screen navigation ───────────────────────────────────────────────────────

type Screen int

const (
	ScreenSearch Screen = iota
	ScreenLyrics
)
