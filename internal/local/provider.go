package local

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"som/internal/domain"
	"som/internal/scraper"
)

type DirectProvider struct {
	Scraper scraper.Scraper
}

func (p *DirectProvider) Search(ctx context.Context, q string) ([]domain.Track, error) {
	start := time.Now()
	results, err := p.Scraper.Search(ctx, q)
	if err != nil {
		log.Printf("search: error for %q after %v: %v", q, time.Since(start), err)
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
	log.Printf("search: %q -> %d result(s) in %v", q, len(tracks), time.Since(start))
	return tracks, nil
}

func (p *DirectProvider) Lyrics(ctx context.Context, id, title, artist string, duration int) (domain.LyricsResp, error) {
	start := time.Now()
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
		log.Printf("lyrics: none for %s (%q) in %v", id, title, time.Since(start))
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

	lineCount := 0
	for _, t := range lr.AllTracks {
		lineCount += len(t.Synced)
	}
	log.Printf("lyrics: %d track(s), %d line(s) for %s in %v", len(lr.AllTracks), lineCount, id, time.Since(start))

	return lr, nil
}

func (p *DirectProvider) ResolveStream(ctx context.Context, id string) (*domain.StreamInfo, error) {
	start := time.Now()
	info, err := p.Scraper.GetStreamInfo(ctx, id)
	if err != nil {
		log.Printf("stream: resolve error for %s after %v: %v", id, time.Since(start), err)
		return nil, err
	}
	log.Printf("stream: url ready for %s in %v", id, time.Since(start))
	return &domain.StreamInfo{URL: info.URL, Headers: info.Headers}, nil
}

func (p *DirectProvider) DownloadOPUS(ctx context.Context, id, title, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", destDir, err)
	}

	start := time.Now()
	tmpFile, err := p.Scraper.DownloadAudio(ctx, id)
	if err != nil {
		log.Printf("download: error for %s after %v: %v", id, time.Since(start), err)
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

	log.Printf("download: %q saved to %s in %v", title, dest, time.Since(start))
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
