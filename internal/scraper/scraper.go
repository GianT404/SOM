package scraper

import (
	"context"
)

// SearchResult represents a single video search result.
type SearchResult struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Thumbnail string `json:"thumbnail"`
	Duration  int    `json:"duration"`
	Uploader  string `json:"uploader"`
}

// LyricLine represents a single line of lyrics with timing info.
type LyricLine struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// StreamInfo holds the direct audio URL and video title for streaming.
type StreamInfo struct {
	URL   string // Direct m4a/aac audio URL
	Title string // Video title (for Content-Disposition filename)
}

// LyricsData maps a subtitle language to its parsed lyrics.
type LyricsData struct {
	Language string      `json:"language"`
	Lines    []LyricLine `json:"lines"`
}

// Scraper abstracts the video/audio data source.
// Implementations can wrap yt-dlp, invidious, or any other backend.
type Scraper interface {
	// Search returns up to 7 video results for the given keyword.
	Search(ctx context.Context, keyword string) ([]SearchResult, error)

	// GetStreamInfo returns the direct audio URL and title for the given video ID.
	GetStreamInfo(ctx context.Context, videoID string) (*StreamInfo, error)

	// DownloadAudio downloads audio to a temporary file and returns its path.
	// This ensures the moov atom is correctly written for m4a files,
	// bypassing ExoPlayer UnrecognizedInputFormatException issues.
	DownloadAudio(ctx context.Context, videoID string) (string, error)

	// VideoTitle returns the title of the video.
	VideoTitle(ctx context.Context, videoID string) (string, error)

	// Lyrics returns all parsed subtitle lines mapped by language for the given video ID.
	Lyrics(ctx context.Context, videoID string) ([]LyricsData, error)
}
