package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kkdai/youtube/v2"
)

const maxCaptionLanguages = 10

func GetDirectSubtitles(ctx context.Context, videoID string, preferredLang string) ([]LyricsData, error) {
	client := youtube.Client{}

	video, err := client.GetVideoContext(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("could not get video info: %w", err)
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	if len(video.CaptionTracks) == 0 {
		return nil, fmt.Errorf("no caption tracks available")
	}

	tracks := video.CaptionTracks
	if preferredLang != "" {
		tracks = reorderTracks(video.CaptionTracks, preferredLang)
	}

	var results []LyricsData
	seenLang := make(map[string]bool)

	for _, track := range tracks {
		if len(results) >= maxCaptionLanguages {
			break
		}
		baseUrl := track.BaseURL
		langCode := track.LanguageCode
		if baseUrl == "" || langCode == "" || seenLang[langCode] {
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

		mappedLines := ParseVTT(string(bodyBytes))
		if len(mappedLines) == 0 {
			continue
		}

		seenLang[langCode] = true
		results = append(results, LyricsData{
			Language: langCode,
			Lines:    mappedLines,
		})
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no subtitles found or all fetches failed")
	}

	return results, nil
}

func GetDirectAudio(ctx context.Context, videoID string, destPath string) error {
	client := youtube.Client{}
	video, err := client.GetVideoContext(ctx, videoID)
	if err != nil {
		return err
	}

	formats := video.Formats.WithAudioChannels()
	if len(formats) == 0 {
		return fmt.Errorf("no audio formats found")
	}

	formats.Sort()
	var bestFormat *youtube.Format
	for i := range formats {
		if formats[i].ItagNo == 140 {
			bestFormat = &formats[i]
			break
		}
	}
	if bestFormat == nil {
		for i := range formats {
			if strings.Contains(formats[i].MimeType, "audio/mp4") {
				bestFormat = &formats[i]
				break
			}
		}
	}
	if bestFormat == nil {
		return fmt.Errorf("no m4a audio format found")
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
func reorderTracks(tracks []youtube.CaptionTrack, preferredLang string) []youtube.CaptionTrack {
	var preferred, rest []youtube.CaptionTrack
	for _, t := range tracks {
		if t.LanguageCode == preferredLang || strings.HasPrefix(t.LanguageCode, preferredLang) {
			preferred = append(preferred, t)
		} else {
			rest = append(rest, t)
		}
	}
	return append(preferred, rest...)
}
