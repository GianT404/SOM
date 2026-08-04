package domain

import "context"

type MusicProvider interface {
	Search(ctx context.Context, q string) ([]Track, error)
	Lyrics(ctx context.Context, id, title, artist string, duration int) (LyricsResp, error)
	ResolveStream(ctx context.Context, id string) (string, error)
	DownloadOPUS(ctx context.Context, id, title, destDir string) (string, error)
}
