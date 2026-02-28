package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/kkdai/youtube/v2"
)

// DirectYoutubeScraper provides an alternative way to fetch subtitles bypassing yt-dlp 429 errors.
func GetDirectSubtitles(ctx context.Context, videoID string) ([]LyricsData, error) {
	client := youtube.Client{}

	// Fetch video info using Innertube API
	video, err := client.GetVideoContext(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("could not get video info: %w", err)
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	var results []LyricsData

	// Get caption tracks from the video metadata
	// `kkdai/youtube` parses the CaptionTracks automatically for the main langs
	if len(video.CaptionTracks) > 0 {
		for _, track := range video.CaptionTracks {
			baseUrl := track.BaseURL
			if baseUrl == "" {
				continue
			}

			// Add vtt format query
			vttUrl := baseUrl + "&fmt=vtt"

			req, err := http.NewRequestWithContext(ctx, "GET", vttUrl, nil)
			if err != nil {
				continue
			}

			// Mimic browser
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
			req.Header.Set("Referer", "https://www.youtube.com/")
			req.Header.Set("Accept-Language", "vi-VN,vi;q=0.9,en-US;q=0.8,en;q=0.7")

			resp, err := httpClient.Do(req)
			if err != nil || resp.StatusCode != 200 {
				if resp != nil {
					resp.Body.Close()
				}
				continue
			}

			bodyBytes, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				continue
			}

			vttContent := string(bodyBytes)
			mappedLines := ParseVTT(vttContent)

			if len(mappedLines) > 0 {
				langCode := track.LanguageCode
				// Determine if it's auto-generated, youtube API often adds kind=asr
				// however, kkdai doesn't export Kind, so we will just use the returned language.

				results = append(results, LyricsData{
					Language: langCode,
					Lines:    mappedLines,
				})
			}
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no subtitles found or all fetches failed")
	}

	return results, nil
}

// GetDirectAudio fetches the best audio stream using kkdai/youtube/v2 directly to bypass yt-dlp.
func GetDirectAudio(ctx context.Context, videoID string, destPath string) error {
	client := youtube.Client{}
	video, err := client.GetVideoContext(ctx, videoID)
	if err != nil {
		return err
	}

	formats := video.Formats.WithAudioChannels() // only get formats with audio
	if len(formats) == 0 {
		return fmt.Errorf("no audio formats found")
	}

	// Try to find m4a format 140 or highest quality first
	formats.Sort()
	var bestFormat *youtube.Format
	for i := range formats {
		if formats[i].ItagNo == 140 { // Standard 128kbps AAC
			bestFormat = &formats[i]
			break
		}
	}
	if bestFormat == nil {
		bestFormat = &formats[0] // fallback to any
	}

	stream, _, err := client.GetStreamContext(ctx, video, bestFormat)
	if err != nil {
		return err
	}
	defer stream.Close()

	file, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, stream)
	if err != nil {
		return err
	}

	return nil
}
