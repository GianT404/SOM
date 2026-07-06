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

// DirectYoutubeScraper provides an alternative way to fetch subtitles bypassing yt-dlp 429 errors.
func GetDirectSubtitles(ctx context.Context, videoID string, preferredLang string) ([]LyricsData, error) {
	client := youtube.Client{}

	// Fetch video info using Innertube API
	video, err := client.GetVideoContext(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("could not get video info: %w", err)
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Get caption tracks from the video metadata
	// `kkdai/youtube` parses the CaptionTracks automatically for the main langs
	if len(video.CaptionTracks) > 0 {
		// Order tracks: preferred language first, then everything else
		tracks := video.CaptionTracks
		if preferredLang != "" {
			tracks = reorderTracks(video.CaptionTracks, preferredLang)
		}
		for _, track := range tracks {
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

				return []LyricsData{{
					Language: langCode,
					Lines:    mappedLines,
				}}, nil
			}
		}
	}

	return nil, fmt.Errorf("no subtitles found or all fetches failed")
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

	// Only write MP4/AAC audio to the .m4a cache path. WebM/Opus bytes with a
	// .m4a extension fail in WebKit's audio element.
	formats.Sort()
	var bestFormat *youtube.Format
	for i := range formats {
		if formats[i].ItagNo == 140 { // Standard 128kbps AAC
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

// reorderTracks moves tracks matching preferredLang to the front, so they are
// tried first when iterating. Tracks that match the preferred language and are
// not auto-generated (if detectable) get the highest priority.
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
