package scraper

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// lyricsCandidate là 1 phương án (track, artist) để thử tìm trên LRCLib,
// kèm nguồn gốc (để log/debug biết candidate nào match).
type lyricsCandidate struct {
	track  string
	artist string
	source string // "metadata" | "title"
}

// buildCandidateList gộp candidate từ 2 nguồn thành 1 danh sách theo thứ tự ưu tiên:
//  1. YouTube Music metadata (nếu có) - độ tin cậy cao nhất, thử trước
//  2. Parse từ raw title (splitDash/splitPipe/...) - fallback bắt buộc, vì rất
//     nhiều video (nhạc rap/indie tự upload) không có metadata YouTube Music.
//
// Đây là điểm khác với flow "metadata -> LRCLib, fail thì thôi": coi title-parsing
// là 1 phần của cùng pipeline chứ không phải nhánh riêng, để không bỏ sót các
// video thiếu metadata (rất phổ biến trong thực tế).
func buildCandidateList(meta MusicMetadata, rawTitle string) []lyricsCandidate {
	var out []lyricsCandidate
	seen := map[string]bool{}

	add := func(track, artist, source string) {
		track = normalizeSpaces(track)
		artist = normalizeSpaces(artist)
		key := strings.ToLower(track + "|" + artist)
		if track == "" || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, lyricsCandidate{track, artist, source})
	}

	if meta.Track != "" {
		add(meta.Track, meta.Artist, "metadata")
	}

	for _, c := range buildTitleCandidates(rawTitle, meta.Artist) {
		add(c.track, c.artist, "title")
	}

	return out
}

// FetchLyrics là entrypoint DUY NHẤT để lấy lyrics từ LRCLib.
//
// Pipeline: build danh sách candidate (metadata trước, title-parsing sau) ->
// với từng candidate: /api/get (exact) rồi fallback /api/search, chấm điểm và
// validate confidence (xem lookupLRCLibCandidate) -> trả kết quả tin cậy đầu
// tiên tìm được, hoặc lỗi nếu không candidate nào đủ tin cậy.
func FetchLyrics(ctx context.Context, meta MusicMetadata, rawTitle string, durationSec float64) ([]LyricsData, error) {
	candidates := buildCandidateList(meta, rawTitle)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("lrclib: không có candidate nào (thiếu cả metadata và title)")
	}

	var lastErr error
	for _, c := range candidates {
		data, err := lookupLRCLibCandidate(ctx, c.track, c.artist, durationSec)
		if err == nil && len(data) > 0 {
			log.Printf("lrclib: matched via %s candidate track=%q artist=%q", c.source, c.track, c.artist)
			return data, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("lrclib: không tìm thấy lyrics đủ tin cậy cho %q (%w)", rawTitle, lastErr)
}
