package scraper

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const lrclibBase = "https://lrclib.net"

// lrclibTrack is the shape of a single result from LRCLib.
type lrclibTrack struct {
	ID           int     `json:"id"`
	TrackName    string  `json:"trackName"`
	ArtistName   string  `json:"artistName"`
	AlbumName    string  `json:"albumName"`
	Duration     float64 `json:"duration"`
	Instrumental bool    `json:"instrumental"`
	SyncedLyrics string  `json:"syncedLyrics"`
	PlainLyrics  string  `json:"plainLyrics"`
}

var lrcTimestampRe = regexp.MustCompile(`^\[(\d+):(\d+)(?:[.:](\d+))?\]\s*(.*)$`)
var lrcRegex = regexp.MustCompile(`^\[(\d{2,}):(\d{2})\.(\d{2,3})\](.*)`)

func parseLRC(syncedLyrics string) []LyricLine {
	var lines []LyricLine
	scanner := bufio.NewScanner(strings.NewReader(syncedLyrics))

	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}

		matches := lrcRegex.FindStringSubmatch(text)
		if len(matches) == 5 {
			min, _ := strconv.Atoi(matches[1])
			sec, _ := strconv.Atoi(matches[2])

			msStr := matches[3]
			ms, _ := strconv.Atoi(msStr)
			if len(msStr) == 2 {
				ms *= 10
			}

			start := float64(min)*60 + float64(sec) + float64(ms)/1000.0
			content := strings.TrimSpace(matches[4])

			lines = append(lines, LyricLine{
				Start: start,
				End:   0,
				Text:  content,
			})
		}
	}

	for i := 0; i < len(lines)-1; i++ {
		lines[i].End = lines[i+1].Start

		if lines[i].End-lines[i].Start > 10.0 {
			lines[i].End = lines[i].Start + 5.0
		}
	}

	if len(lines) > 0 {
		lastIdx := len(lines) - 1
		lines[lastIdx].End = lines[lastIdx].Start + 5.0
	}

	return lines
}

var lrclibHTTPClient = &http.Client{Timeout: 6 * time.Second}

// ---- Thuật toán chấm điểm để chọn kết quả LRCLib đúng nhất ----
// Trước đây chỉ chọn theo duration gần nhất -> dễ chọn nhầm bài hoàn toàn
// khác nếu title bị làm sạch sai (VD: yt-dlp trả "NA"/"NA"). Giờ kết hợp
// độ giống tên bài + nghệ sĩ với độ lệch duration, và loại bỏ kết quả
// có độ giống tên quá thấp thay vì chấp nhận đại.

const (
	minNameScore     = 0.35 // dưới ngưỡng này coi như không liên quan, loại bỏ
	minAcceptedScore = 0.45 // điểm tổng tối thiểu để chấp nhận 1 kết quả
)

var nonWordRe = regexp.MustCompile(`[^\p{L}\p{N} ]+`)

// viDiacriticsReplacer bỏ dấu tiếng Việt, vì title YouTube hay bị gõ
// không dấu trong khi LRCLib lưu có dấu (hoặc ngược lại).
var viDiacriticsReplacer = strings.NewReplacer(
	"à", "a", "á", "a", "ạ", "a", "ả", "a", "ã", "a",
	"â", "a", "ầ", "a", "ấ", "a", "ậ", "a", "ẩ", "a", "ẫ", "a",
	"ă", "a", "ằ", "a", "ắ", "a", "ặ", "a", "ẳ", "a", "ẵ", "a",
	"è", "e", "é", "e", "ẹ", "e", "ẻ", "e", "ẽ", "e",
	"ê", "e", "ề", "e", "ế", "e", "ệ", "e", "ể", "e", "ễ", "e",
	"ì", "i", "í", "i", "ị", "i", "ỉ", "i", "ĩ", "i",
	"ò", "o", "ó", "o", "ọ", "o", "ỏ", "o", "õ", "o",
	"ô", "o", "ồ", "o", "ố", "o", "ộ", "o", "ổ", "o", "ỗ", "o",
	"ơ", "o", "ờ", "o", "ớ", "o", "ợ", "o", "ở", "o", "ỡ", "o",
	"ù", "u", "ú", "u", "ụ", "u", "ủ", "u", "ũ", "u",
	"ư", "u", "ừ", "u", "ứ", "u", "ự", "u", "ử", "u", "ữ", "u",
	"ỳ", "y", "ý", "y", "ỵ", "y", "ỷ", "y", "ỹ", "y",
	"đ", "d",
)

// normalizeForMatch chuẩn hoá chuỗi để so sánh: lowercase, bỏ dấu, bỏ ký tự đặc biệt
func normalizeForMatch(s string) string {
	s = strings.ToLower(s)
	s = viDiacriticsReplacer.Replace(s)
	s = nonWordRe.ReplaceAllString(s, " ")
	return normalizeSpaces(s)
}

func tokenSet(s string) map[string]bool {
	set := make(map[string]bool)
	for _, w := range strings.Fields(normalizeForMatch(s)) {
		if len(w) > 1 { // bỏ token 1 ký tự (thường là noise)
			set[w] = true
		}
	}
	return set
}

// jaccardSimilarity đo độ giống nhau giữa 2 chuỗi theo tập từ (0..1)
func jaccardSimilarity(a, b string) float64 {
	setA, setB := tokenSet(a), tokenSet(b)
	if len(setA) == 0 && len(setB) == 0 {
		return 0
	}
	inter := 0
	for w := range setA {
		if setB[w] {
			inter++
		}
	}
	union := len(setA) + len(setB) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// nameScore kết hợp độ giống track name (trọng số cao) và artist name
func nameScore(queryTrack, queryArtist, candTrack, candArtist string) float64 {
	trackSim := jaccardSimilarity(queryTrack, candTrack)
	// Nếu track name khớp gần như tuyệt đối theo substring, ưu tiên mạnh
	if qn, cn := normalizeForMatch(queryTrack), normalizeForMatch(candTrack); qn != "" &&
		(strings.Contains(cn, qn) || strings.Contains(qn, cn)) {
		trackSim = math.Max(trackSim, 0.8)
	}

	if queryArtist == "" {
		return trackSim
	}
	artistSim := jaccardSimilarity(queryArtist, candArtist)
	return trackSim*0.75 + artistSim*0.25
}

// durationScore trả 1.0 nếu duration khớp hoàn toàn, giảm dần khi lệch xa
func durationScore(diffSec float64) float64 {
	const tolerance = 8.0 // giây, lệch trong khoảng này coi gần như khớp
	if diffSec <= tolerance {
		return 1.0
	}
	score := 1.0 - (diffSec-tolerance)/30.0
	if score < 0 {
		return 0
	}
	return score
}

// lookupLRCLibCandidate tra 1 candidate (track, artist) cụ thể trên LRCLib.
// Thử /api/get (exact) trước, fail thì fallback /api/search + chấm điểm.
// Đây là building block cho FetchLyrics (lyrics_pipeline.go), không gọi trực
// tiếp từ ngoài package nữa.
func lookupLRCLibCandidate(ctx context.Context, trackName, artistName string, durationSec float64) ([]LyricsData, error) {
	// Cap total time spent across both strategies (get + search) so this
	// secondary feature can't stall the parent request for too long. The
	// caller (handler) also enforces its own overall deadline; this is a
	// tighter inner bound so LRCLib alone never eats the whole budget.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// ---- Strategy 1: exact match via /api/get ----
	if durationSec > 0 && trackName != "" {
		params := url.Values{
			"track_name":  {trackName},
			"artist_name": {artistName},
			"duration":    {fmt.Sprintf("%.0f", durationSec)},
		}
		reqURL := lrclibBase + "/api/get?" + params.Encode()
		if t, err := fetchLRCLibGet(ctx, reqURL); err == nil && t != nil {
			// vẫn chấm điểm để tránh trường hợp server match nhầm bản cover/remix
			if nameScore(trackName, artistName, t.TrackName, t.ArtistName) >= minNameScore {
				return lrclibTrackToLyricsData(*t), nil
			}
		}
	}

	if trackName == "" {
		return nil, fmt.Errorf("lrclib: empty track name")
	}

	// ---- Strategy 2: search fallback (trả về tối đa ~20 kết quả) ----
	query := trackName
	if artistName != "" {
		query = artistName + " " + trackName
	}
	params := url.Values{"q": {query}}
	reqURL := lrclibBase + "/api/search?" + params.Encode()

	tracks, err := fetchLRCLibSearch(ctx, reqURL)
	if err != nil || len(tracks) == 0 {
		return nil, fmt.Errorf("lrclib: no results for %q", trackName)
	}

	// Chấm điểm từng kết quả: kết hợp độ giống tên (chính) + độ lệch duration (phụ)
	var best *lrclibTrack
	var bestScore float64
	for i := range tracks {
		t := &tracks[i]
		ns := nameScore(trackName, artistName, t.TrackName, t.ArtistName)
		if ns < minNameScore {
			continue // tên không liên quan, loại ngay dù duration có khớp
		}

		score := ns
		if durationSec > 0 {
			ds := durationScore(absDiff(t.Duration, durationSec))
			score = ns*0.7 + ds*0.3
		}

		if best == nil || score > bestScore {
			best = t
			bestScore = score
		}
	}

	if best == nil || bestScore < minAcceptedScore {
		return nil, fmt.Errorf("lrclib: no confident match for %q by %q", trackName, artistName)
	}

	return lrclibTrackToLyricsData(*best), nil
}

func absDiff(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}

// fetchLRCLibGet does a GET and decodes a single lrclibTrack (raw, chưa convert)
// để caller có thể chấm điểm trước khi chấp nhận.
func fetchLRCLibGet(ctx context.Context, reqURL string) (*lrclibTrack, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	req.Header.Set("Lrclib-Client", "SOM/1.0 (https://github.com/GianT404/SOM)")

	resp, err := lrclibHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // signal "not found"
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lrclib: HTTP %d from %s", resp.StatusCode, reqURL)
	}

	var t lrclibTrack
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return nil, fmt.Errorf("lrclib: decode error: %w", err)
	}

	return &t, nil
}

// fetchLRCLibSearch does a GET and decodes an array of lrclibTrack.
func fetchLRCLibSearch(ctx context.Context, reqURL string) ([]lrclibTrack, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	req.Header.Set("Lrclib-Client", "SOM/1.0 (https://github.com/GianT404/SOM)")

	resp, err := lrclibHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lrclib search: HTTP %d", resp.StatusCode)
	}

	var tracks []lrclibTrack
	if err := json.NewDecoder(resp.Body).Decode(&tracks); err != nil {
		return nil, fmt.Errorf("lrclib search: decode error: %w", err)
	}
	return tracks, nil
}

// lrclibTrackToLyricsData converts a lrclibTrack into our internal format.
// We return only one language "lrclib" (synced) or "" (plain).
func lrclibTrackToLyricsData(t lrclibTrack) []LyricsData {
	var result []LyricsData

	if t.SyncedLyrics != "" {
		lines := parseLRC(t.SyncedLyrics)
		if len(lines) > 0 {
			result = append(result, LyricsData{
				Language:   "lrclib",
				Lines:      lines,
				TrackName:  t.TrackName,
				ArtistName: t.ArtistName,
			})
		}
	}

	// If no synced lyrics but there are plain lyrics, serve them as a single block.
	if len(result) == 0 && t.PlainLyrics != "" {
		var lines []LyricLine
		for _, l := range strings.Split(t.PlainLyrics, "\n") {
			l = strings.TrimSpace(l)
			if l != "" {
				lines = append(lines, LyricLine{Text: l})
			}
		}
		if len(lines) > 0 {
			result = append(result, LyricsData{
				Language:   "lrclib",
				Lines:      lines,
				TrackName:  t.TrackName,
				ArtistName: t.ArtistName,
			})
		}
	}

	return result
}
