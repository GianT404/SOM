package scraper

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

type candidate struct {
	track    string
	artist   string
	source   string
	priority int
}

// Tạo các cặp track/nghệ sĩ theo nhiều quy tắc, không phụ thuộc một mẫu title.
func buildTitleCandidates(rawTitle, artist string) []candidate {
	var out []candidate
	byKey := make(map[string]int)

	add := func(track, art, source string, priority int) {
		track = cleanTrackCandidate(track)
		art = cleanArtistCandidate(art)
		if track == "" {
			return
		}

		key := strings.ToLower(track + "|" + art)
		if index, ok := byKey[key]; ok {
			if priority > out[index].priority {
				out[index].source = source
				out[index].priority = priority
			}
			return
		}

		byKey[key] = len(out)
		out = append(out, candidate{
			track:    track,
			artist:   art,
			source:   source,
			priority: priority,
		})
	}

	addRaw := func(track, art string) {
		track = normalizeSpaces(stripBracketedText(track))
		art = cleanArtistCandidate(art)
		if track == "" {
			return
		}
		key := strings.ToLower(track + "|" + art)
		if index, ok := byKey[key]; ok {
			if out[index].priority < 1 {
				out[index].source = "raw_fallback"
				out[index].priority = 1
			}
			return
		}
		byKey[key] = len(out)
		out = append(out, candidate{
			track: track, artist: art, source: "raw_fallback", priority: 1,
		})
	}

	raw := normalizeSpaces(rawTitle)
	if raw == "" {
		return nil
	}

	// Xóa noise đã biết rồi phân tích phần title còn lại.
	cleaned := stripTrailingContextSegments(cleanTitleNoise(raw))
	searchTitle := cleaned
	artistHint := cleanArtistCandidate(artist)

	// Tách nghệ sĩ ở cuối title, ví dụ "Song / Artist".
	if track, suffixArtist, ok := splitSuffixArtist(searchTitle, "/", "／"); ok {
		track = stripTrailingContextSegments(track)
		add(track, suffixArtist, "slash_suffix", 125)
		searchTitle = track
		if artistHint == "" {
			artistHint = cleanArtistCandidate(suffixArtist)
		}
	}

	// Tách title và nghệ sĩ theo dấu |, bỏ phần kênh ở cuối.
	if track, suffixArtist := splitPipe(searchTitle); suffixArtist != "" && !isContextNoise(suffixArtist) {
		add(track, suffixArtist, "pipe", 115)
		if artistHint == "" {
			artistHint = cleanArtistCandidate(suffixArtist)
		}
		if artist != "" {
			add(track, artist, "pipe_metadata_artist", 108)
		}
		searchTitle = track
	}

	// Nhận diện cú pháp tiếng Nhật: Artist「Track」.
	if track, parsedArtist := splitJapaneseBracket(raw); track != "" {
		add(track, firstOf(parsedArtist, artistHint), "japanese_bracket", 118)
	}

	hasStructure := strings.ContainsAny(searchTitle, "|/／") ||
		strings.Contains(searchTitle, " - ") ||
		strings.Contains(searchTitle, " – ") ||
		strings.Contains(searchTitle, " — ")

	// Thử cả hai chiều khi gặp dấu gạch ngang vì thứ tự không cố định.
	for _, separator := range []string{" - ", " – ", " — "} {
		parts := strings.SplitN(searchTitle, separator, 2)
		if len(parts) != 2 {
			continue
		}

		left := cleanTrackCandidate(stripTrailingContextSegments(parts[0]))
		right := cleanTrackCandidate(stripTrailingContextSegments(parts[1]))
		if left == "" || right == "" {
			continue
		}

		leftArtist := cleanArtistCandidate(left)
		rightArtist := cleanArtistCandidate(right)
		var track, parsedArtist string
		priority := 105
		source := "dash_length_heuristic"

		switch {
		case artistHint != "" && sameTitle(leftArtist, artistHint):
			track, parsedArtist, source = right, artistHint, "dash_known_artist_left"
		case artistHint != "" && sameTitle(rightArtist, artistHint):
			track, parsedArtist, source = left, artistHint, "dash_known_artist_right"
		default:
			if utf8.RuneCountInString(left) <= utf8.RuneCountInString(right) {
				track, parsedArtist = right, leftArtist
			} else {
				track, parsedArtist = left, rightArtist
			}
		}

		add(track, parsedArtist, source, priority)

		// Giữ chiều ngược và gợi ý artist từ client ở mức ưu tiên thấp hơn.
		if sameTitle(track, left) {
			add(right, leftArtist, "dash_reverse", 70)
			add(left, artistHint, "dash_left_with_hint", 65)
			add(right, artistHint, "dash_right_with_hint", 64)
		} else {
			add(left, rightArtist, "dash_reverse", 70)
			add(left, artistHint, "dash_left_with_hint", 65)
			add(right, artistHint, "dash_right_with_hint", 64)
		}
	}

	// Title không có cấu trúc rõ ràng được ưu tiên hơn title còn nhiều thành phần.
	wholePriority := 92
	if hasStructure {
		wholePriority = 35
	}
	add(searchTitle, artistHint, "clean_title", wholePriority)
	if !hasStructure {
	}

	// Luôn giữ một bản nguyên văn ở độ ưu tiên thấp nhất.
	addRaw(raw, artist)

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].priority > out[j].priority
	})
	return out
}

// Tạo thêm candidate từ tên bài và alias đặt trong ngoặc.
// Tách nghệ sĩ ở hậu tố nếu phần cuối trông giống tên người/nghệ danh.
func splitSuffixArtist(title string, separators ...string) (track, artist string, ok bool) {
	bestIndex, bestSeparator := -1, ""
	for _, separator := range separators {
		if index := strings.LastIndex(title, separator); index > bestIndex {
			bestIndex, bestSeparator = index, separator
		}
	}
	if bestIndex <= 0 {
		return "", "", false
	}

	left := strings.TrimSpace(title[:bestIndex])
	right := normalizeSpaces(title[bestIndex+len(bestSeparator):])
	if left == "" || !isLikelyArtist(right) {
		return "", "", false
	}
	return left, right, true
}

// Loại các cụm hậu tố thường chỉ mô tả video, bản phối hoặc kênh đăng.
func stripTrailingContextSegments(title string) string {
	title = normalizeSpaces(title)
	for title != "" {
		index, separator := lastTitleSeparator(title)
		if index < 0 {
			break
		}
		suffix := strings.TrimSpace(title[index+len(separator):])
		if !isContextOnlySegment(suffix) {
			break
		}
		title = normalizeSpaces(title[:index])
	}
	return title
}

func lastTitleSeparator(title string) (int, string) {
	bestIndex, bestSeparator := -1, ""
	for _, separator := range []string{" - ", " – ", " — ", " | "} {
		if index := strings.LastIndex(title, separator); index > bestIndex {
			bestIndex, bestSeparator = index, separator
		}
	}
	return bestIndex, bestSeparator
}

// Chuẩn hóa title, chỉ xóa cụm nhiễu đã nhận diện được.
func cleanTitleNoise(title string) string {
	// Cắt phần ngôn ngữ phụ đề thường được nối sau dấu phân cách.
	title = regexLyricsLanguageSuffix.ReplaceAllString(title, "")
	title = regexSubLanguageSuffix.ReplaceAllString(title, "")
	title = stripNoiseParens(title)
	title = regexVersionSuffix.ReplaceAllString(title, "")
	title = removeTrashKeywords(title)
	title = strings.ReplaceAll(title, string(rune(34)), "")
	title = strings.NewReplacer("“", "", "”", "", "„", "", "‟", "", "＂", "").Replace(title)
	return normalizeSpaces(title)
}
func cleanTrackCandidate(title string) string {
	return normalizeSpaces(stripTrailingContextSegments(cleanTitleNoise(title)))
}

// Bỏ alias trong ngoặc khỏi artist để không gửi cả tên phiên âm vào LRCLib.
func cleanArtistCandidate(artist string) string {
	return normalizeSpaces(stripBracketedText(artist))
}

func isLikelyArtist(value string) bool {
	value = normalizeSpaces(value)
	if value == "" || isContextNoise(value) || utf8.RuneCountInString(value) > 48 {
		return false
	}
	return len(strings.Fields(value)) <= 5
}

func isContextNoise(value string) bool {
	return regexContextNoise.MatchString(value)
}

// Chỉ loại hẳn một đoạn khi toàn bộ đoạn đó là metadata hoặc hậu tố nhiễu.
func isContextOnlySegment(value string) bool {
	value = normalizeSpaces(value)
	return regexAnimeContextOnly.MatchString(value) ||
		regexMetadataOnly.MatchString(value) ||
		regexVersionOnly.MatchString(value) ||
		regexChannelOnly.MatchString(value)
}

func sameTitle(a, b string) bool {
	return strings.EqualFold(normalizeSpaces(a), normalizeSpaces(b))
}

// Tách "Song | Artist", bỏ các phần sau thường là kênh hoặc label.
func splitPipe(title string) (track, artist string) {
	var parts []string
	for _, part := range strings.Split(title, "|") {
		if part = strings.TrimSpace(part); part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) < 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

// Tách title dạng Artist「Track」, có thể kèm 【MV】 ở đầu.
func splitJapaneseBracket(title string) (track, artist string) {
	match := regexJaTitle.FindStringSubmatch(title)
	if match == nil {
		return "", ""
	}
	track = strings.TrimSpace(match[1])
	artist = regexJaBracket.ReplaceAllString(title, "")
	artist = regexJaTitle.ReplaceAllString(artist, "")
	artist = cleanArtistCandidate(artist)
	return track, artist
}

func stripNoiseParens(title string) string {
	return stripBracketedText(title)
}

// Xóa nội dung nằm trong các cặp ngoặc phổ biến.
func stripBracketedText(value string) string {
	pairs := map[rune]rune{
		'[': ']', '{': '}', '(': ')',
		'【': '】', '（': '）', '［': '］', '｛': '｝',
		'〔': '〕', '〖': '〗',
	}
	var output strings.Builder
	var closing []rune

	for _, char := range value {
		if len(closing) > 0 {
			if end, ok := pairs[char]; ok {
				closing = append(closing, end)
				continue
			}
			if char == closing[len(closing)-1] {
				closing = closing[:len(closing)-1]
			}
			continue
		}

		if end, ok := pairs[char]; ok {
			closing = append(closing, end)
			continue
		}
		output.WriteRune(char)
	}
	return normalizeSpaces(output.String())
}

func removeTrashKeywords(title string) string {
	return normalizeSpaces(regexTrashWords.ReplaceAllString(title, ""))
}

func firstOf(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func normalizeSpaces(value string) string {
	value = regexSpaces.ReplaceAllString(value, " ")
	return strings.Trim(value, " -|,.")
}

// Nhận diện metadata và hậu tố cần bỏ khỏi title.
var (
	regexJaTitle = regexp.MustCompile(`「([^」]+)」`)
	regexJaBracket = regexp.MustCompile(`【[^】]*】`)
	regexTrashWords = regexp.MustCompile(`(?i)\b(official music video|official video|official audio|lyric video|music video|full version|visualizer|lyrics?|official|audio|4k|1080p|720p|hd|creditless)\b`)
	regexVersionSuffix = regexp.MustCompile(`(?i)\s+(live\s+band\s+cover|live\s+cover|piano\s+cover|guitar\s+cover|acoustic\s+cover|cover(\s+version)?|instrumental(\s+version)?|karaoke(\s+version)?|remix|nightcore|slowed(\s*(\+|and)?\s*reverb)?|sped\s*up|speed\s*up|bass\s+boosted|8d(\s+audio)?|lo[\s-]?fi|orchestral\s+version)(\s+by\s+.+)?\s*$`)
	regexContextNoise = regexp.MustCompile(`(?i)\b(op|ed)\s*\d*\b|\b(opening|ending)(\s+(theme|song|credits?))?\b|\b(official|lyrics?|visualizer|piano|instrumental|karaoke|cover|remix|arrangement|1080p|720p|4k|amv|creditless|nightcore|slowed|sped\s*up|channel|topic)\b`)
	regexLyricsLanguageSuffix = regexp.MustCompile(`(?i)\s+(~|\||｜|-|–|—)\s*(lyrics?|subtitles?)\b.*$`)
	regexSubLanguageSuffix = regexp.MustCompile(`(?i)\s*/\s*sub(titles?)?\b.*$`)
	regexAnimeContextOnly = regexp.MustCompile(`(?i)^(.*\s+)?(op|ed)\s*\d*$`)
	regexMetadataOnly = regexp.MustCompile(`(?i)^(official\s+)?(music\s+)?(video|mv|m/v|audio|lyrics?|lyric\s+video|visualizer|1080p|720p|4k|hd|creditless)$`)
	regexVersionOnly = regexp.MustCompile(`(?i)^(live\s+band\s+cover|live\s+cover|piano\s+cover|guitar\s+cover|acoustic\s+cover|cover(\s+version)?|instrumental(\s+version)?|karaoke(\s+version)?|remix|nightcore|slowed(\s*(\+|and)?\s*reverb)?|sped\s*up|speed\s*up|bass\s+boosted|8d(\s+audio)?|lo[\s-]?fi|orchestral\s+version)(\s+by\s+.+)?$`)
	regexChannelOnly = regexp.MustCompile(`(?i)^(.*\s+)?(official\s+)?(youtube\s+)?channel$`)
	regexSpaces = regexp.MustCompile(`\s+`)
)
