package domain

import "context"

// StreamInfo mô tả URL audio để chơi trực tiếp kèm HTTP headers cần thiết
// (User-Agent…) mà ffmpeg phải gửi kèm để CDN không chặn (403).
type StreamInfo struct {
	URL     string
	Headers map[string]string
}

type MusicProvider interface {
	Search(ctx context.Context, q string) ([]Track, error)
	Lyrics(ctx context.Context, id, title, artist string, duration int) (LyricsResp, error)
	ResolveStream(ctx context.Context, id string) (*StreamInfo, error)
	DownloadOPUS(ctx context.Context, id, title, destDir string) (string, error)
}
