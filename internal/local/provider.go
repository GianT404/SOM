package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"som/internal/domain"
	"som/internal/scraper"
)

// DirectProvider giao tiếp thẳng với core scraper, không qua mạng.
type DirectProvider struct {
	Scraper scraper.Scraper
}

func (p *DirectProvider) Search(ctx context.Context, q string) ([]domain.Track, error) {
	results, err := p.Scraper.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	var tracks []domain.Track
	for _, r := range results {
		tracks = append(tracks, domain.Track{
			ID:        r.ID,
			Title:     r.Title,
			Artist:    r.Artist,
			Duration:  r.Duration,
			Thumbnail: r.Thumbnail,
		})
	}
	return tracks, nil
}

func (p *DirectProvider) Lyrics(ctx context.Context, id, title, artist string, duration int) (domain.LyricsResp, error) {
	var lr domain.LyricsResp
	lr.Title = title
	lr.VideoID = id

	type result struct {
		source string
		data   []scraper.LyricsData
		err    error
	}
	resCh := make(chan result, 2)

	go func() {
		meta, _ := p.Scraper.VideoMetadata(ctx, id)
		if meta.Artist == "" && artist != "" {
			meta.Artist = artist
		}
		lr.Artist = meta.Artist

		data, err := scraper.FetchLyrics(ctx, meta, title, float64(duration))
		resCh <- result{source: "lrclib", data: data, err: err}
	}()

	go func() {
		data, err := p.Scraper.Lyrics(ctx, id)
		resCh <- result{source: "youtube", data: data, err: err}
	}()

	var combined []scraper.LyricsData

	for i := 0; i < 2; i++ {
		res := <-resCh
		if res.err == nil && len(res.data) > 0 {
			combined = append(combined, res.data...)
		}
	}

	if len(combined) == 0 {
		return lr, fmt.Errorf("không tìm thấy lyrics")
	}
	seen := make(map[string]bool)
	var finalData []scraper.LyricsData
	for _, d := range combined {
		if !seen[d.Language] {
			seen[d.Language] = true
			finalData = append(finalData, d)
		}
	}

	for _, d := range finalData {
		lt := domain.LyricsTrack{
			Language:   d.Language,
			TrackName:  d.TrackName,
			ArtistName: d.ArtistName,
		}
		for _, line := range d.Lines {
			lt.Synced = append(lt.Synced, domain.LyricLine{
				Time: line.Start,
				End:  line.End,
				Text: line.Text,
			})
		}
		lr.AllTracks = append(lr.AllTracks, lt)
	}

	if len(lr.AllTracks) > 0 {
		lr.SelectLanguage(0)
	}

	return lr, nil
}

func (p *DirectProvider) ResolveStream(ctx context.Context, id string) (string, error) {
	info, err := p.Scraper.GetStreamInfo(ctx, id)
	if err != nil {
		return "", err
	}
	return info.URL, nil
}

func (p *DirectProvider) DownloadOPUS(ctx context.Context, id, title, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", destDir, err)
	}

	tmpFile, err := p.Scraper.DownloadAudio(ctx, id)
	if err != nil {
		return "", err
	}

	safe := sanitizeLocal(title)
	if safe == "" {
		safe = id
	}
	dest := filepath.Join(destDir, safe+".opus")

	if err := moveFile(tmpFile, dest); err != nil {
		return "", err
	}

	return dest, nil
}

func sanitizeLocal(s string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", `"`, "-", "<", "-", ">", "-", "|", "-")
	return strings.TrimSpace(r.Replace(s))
}
func moveFile(sourcePath, destPath string) error {
	if err := os.Rename(sourcePath, destPath); err == nil {
		return nil
	}

	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("couldn't open source file: %s", err)
	}
	defer inputFile.Close()

	outputFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("couldn't open dest file: %s", err)
	}
	defer outputFile.Close()

	if _, err = io.Copy(outputFile, inputFile); err != nil {
		return err
	}

	inputFile.Close()
	return os.Remove(sourcePath)
}
