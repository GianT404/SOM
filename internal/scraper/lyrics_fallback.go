package scraper

import (
	"context"
	"fmt"
	"strings"

	"dm4a/internal/cleaner"
)

// Lyrics providers module - contains fallback strategy when primary provider fails

// SearchGeniusLyrics - placeholder for future Genius API support
func SearchGeniusLyrics(ctx context.Context, trackName, artistName string) ([]LyricsData, error) {
	// Build search query
	query := trackName
	if artistName != "" && !strings.Contains(strings.ToLower(trackName), strings.ToLower(artistName)) {
		query = artistName + " " + trackName
	}
	
	// Remove common prefixes like "official video", "lyrics", etc
	query = cleanLyricsQuery(query)
	
	// Note: Genius API requires access token for full access
	// This is a fallback that tries to fetch from public endpoint
	return nil, fmt.Errorf("genius: requires API token (not available)")
}

// cleanLyricsQuery removes common non-lyric keywords
func cleanLyricsQuery(query string) string {
	keywords := []string{
		"official video", "music video", "lyrics", "lyric video",
		"ft.", "feat.", "featuring", "\n", "  ",
		"performance", "live",
	}
	
	lowercase := strings.ToLower(query)
	for _, kw := range keywords {
		lowercase = strings.ReplaceAll(lowercase, strings.ToLower(kw), "")
	}
	
	// Trim and restore original case where possible
	return strings.TrimSpace(query)
}

// GetPlainTextLyrics - fallback to return plain text only
func GetPlainTextLyrics(ctx context.Context, trackName, artistName string) ([]LyricsData, error) {
	// This is a minimal fallback - in real scenario would scrape from lyrics sites
	// For now, return structured data showing lyrics not found
	return nil, fmt.Errorf("lyrics: no synced lyrics available for %q", trackName)
}

// TryMultipleLyricProviders - thử nhiều chiến lược khác nhau để tìm kiếm lyric
func TryMultipleLyricProviders(ctx context.Context, trackName, artistName string, durationSec float64) ([]LyricsData, error) {
	// Strategy 1: LRCLib với tiêu đề ban đầu
	if data, err := GetLRCLibLyrics(ctx, trackName, artistName, durationSec); err == nil && len(data) > 0 {
		return data, nil
	}

	// Strategy 2: Làm sạch tiêu đề YouTube (loại bỏ từ khóa rác, ngoặc, v.v.)
	cleanedTitle := cleaner.CleanYouTubeTitle(trackName)
	if cleanedTitle != trackName && cleanedTitle != "" {
		if data, err := GetLRCLibLyrics(ctx, cleanedTitle, artistName, durationSec); err == nil && len(data) > 0 {
			return data, nil
		}
	}

	// Strategy 3: Cố gắng với tiêu đề đã làm sạch + nghệ sĩ đơn giản hóa
	simplifiedArtist := artistName
	if idx := strings.Index(simplifiedArtist, " and "); idx >= 0 {
		simplifiedArtist = simplifiedArtist[:idx]
	}
	if simplifiedArtist != artistName && simplifiedArtist != "" {
		if data, err := GetLRCLibLyrics(ctx, cleanedTitle, simplifiedArtist, durationSec); err == nil && len(data) > 0 {
			return data, nil
		}
	}

	// Không tìm thấy lyric
	return nil, fmt.Errorf("no lyrics found for %q by %q", trackName, artistName)
}
