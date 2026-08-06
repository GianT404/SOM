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
// chạy SONG SONG tối đa maxLRCLibCandidates candidate (mỗi candidate thử
// /api/get rồi /api/search, chấm điểm và validate confidence) -> trả kết quả
// tin cậy đầu tiên tìm được, hoặc lỗi nếu không candidate nào đủ tin cậy.
//
// Song song + giới hạn số candidate giúp không cháy toàn bộ budget HTTP của
// request vào LRCLib (trước đây thử tuần tự hàng chục candidate, mỗi candidate
// 2 call HTTP).
const maxLRCLibCandidates = 5

func FetchLyrics(ctx context.Context, meta MusicMetadata, rawTitle string, durationSec float64) ([]LyricsData, error) {
	candidates := buildCandidateList(meta, rawTitle)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("lrclib: không có candidate nào (thiếu cả metadata và title)")
	}
	if len(candidates) > maxLRCLibCandidates {
		candidates = candidates[:maxLRCLibCandidates]
	}

	// Cancel các request LRCLib còn dang dở ngay khi tìm thấy kết quả đầu tiên.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type candResult struct {
		cand lyricsCandidate
		data []LyricsData
		err  error
	}
	results := make(chan candResult, len(candidates))
	for _, c := range candidates {
		go func(c lyricsCandidate) {
			data, err := lookupLRCLibCandidate(ctx, c.track, c.artist, durationSec)
			results <- candResult{cand: c, data: data, err: err}
		}(c)
	}

	var lastErr error
	for i := 0; i < len(candidates); i++ {
		r := <-results
		if r.err == nil && len(r.data) > 0 {
			log.Printf("lrclib: matched via %s candidate track=%q artist=%q", r.cand.source, r.cand.track, r.cand.artist)
			return r.data, nil
		}
		if r.err != nil {
			lastErr = r.err
		}
	}

	return nil, fmt.Errorf("lrclib: không tìm thấy lyrics đủ tin cậy cho %q (%w)", rawTitle, lastErr)
}
