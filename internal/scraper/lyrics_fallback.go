package scraper

import (
	"regexp"
	"strings"
)

type candidate struct {
	track  string
	artist string
}

// buildTitleCandidates sinh ra các biến thể (track, artist) theo thứ tự ưu tiên
// từ cao xuống thấp. Biến thể đầu tiên match LRCLib sẽ được dùng.
func buildTitleCandidates(rawTitle, artist string) []candidate {
	var out []candidate
	seen := map[string]bool{}

	add := func(track, art string) {
		track = normalizeSpaces(track)
		art = normalizeSpaces(art)
		key := strings.ToLower(track + "|" + art)
		if track == "" || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, candidate{track, art})
	}

	// ── Bước 1: thử title gốc ngay (LRCLib đôi khi tìm được dù title dài) ──
	add(rawTitle, artist)

	// ── Bước 2: tách theo các pattern phổ biến ──

	// Pattern: "Artist - Song Title ..."  (dấu gạch ngang có khoảng trắng)
	if track, art := splitDash(rawTitle); track != "" {
		add(track, firstOf(art, artist))
		add(track, artist)
		if art != "" {
			add(art, track) // heuristic có thể đoán ngược, thêm bản đảo để scoring tự chọn
		}
	}

	// Pattern: "Song | Artist" (2 phần) hoặc "Song | Artist | Kênh/Label" (3+ phần)
	if track, art := splitPipe(rawTitle); track != "" {
		add(track, firstOf(art, artist))
		add(track, artist)
		if art != "" {
			add(art, track) // đề phòng đoán ngược thứ tự
		}
	}

	// Pattern: 【MV】Artist「Song」 hoặc Artist「Song」
	if track, art := splitJapaneseBracket(rawTitle); track != "" {
		add(track, firstOf(art, artist))
		add(track, artist)
	}

	// ── Bước 3: loại bỏ noise trong phần track đã tách được ──
	for _, c := range out {
		stripped := stripNoiseParens(c.track)
		if stripped != c.track {
			add(stripped, c.artist)
		}
		noNoise := removeTrashKeywords(stripped)
		if noNoise != stripped {
			add(noNoise, c.artist)
		}
	}

	// ── Bước 4: simplify artist (bỏ "and ...", "feat ...", "x ...") ──
	simpleArtist := simplifyArtist(artist)
	if simpleArtist != artist {
		// Thêm lại các biến thể track tốt nhất với artist đơn giản hơn
		for _, c := range out {
			add(c.track, simpleArtist)
		}
	}

	// ── Bước 5: thử không có artist (search rộng hơn) ──
	for _, c := range out {
		add(c.track, "")
	}

	return out
}

// splitDash tách "Artist - Song" hoặc "Song - Noise" thành (song, artist).
// Chỉ tách ở dấu " - " đầu tiên.
func splitDash(title string) (track, artist string) {
	parts := strings.SplitN(title, " - ", 2)
	if len(parts) != 2 {
		return "", ""
	}
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])
	// Heuristic: nếu phần trái ngắn hơn (thường là tên nghệ sĩ)
	if len(left) <= len(right) {
		return right, left
	}
	return left, right
}

// splitPipe tách title theo dấu "|". Convention phổ biến nhất là
// "Song | Artist" (2 đoạn) hoặc "Song (info) | Artist | Kênh/Label..." (3+ đoạn,
// hay gặp ở video rap/underground VN). Vì vậy: đoạn đầu = track, đoạn 2 = artist,
// các đoạn sau (thường là tên kênh/label) bị bỏ qua.
//
// LƯU Ý: trước đây hàm này dùng heuristic "phần dài hơn là tên bài hát" và chỉ
// SplitN 2 phần, dẫn tới sai hoàn toàn với title 3 đoạn kiểu trên (phần
// "Artist | Kênh" dính lại thành 1 cục dài hơn rồi bị hiểu nhầm là track name).
func splitPipe(title string) (track, artist string) {
	// Loại bỏ ngoặc có noise trước khi tách
	clean := regexNoisyParens.ReplaceAllString(title, "")

	var parts []string
	for _, p := range strings.Split(clean, "|") {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}

	switch len(parts) {
	case 0:
		return "", ""
	case 1:
		return parts[0], ""
	default:
		// 2 phần: "Song | Artist". 3+ phần: lấy 2 phần đầu, bỏ phần còn lại
		// (thường là tên kênh/label, không phải nhạc).
		return parts[0], parts[1]
	}
}

// splitJapaneseBracket tách 【MV】Artist「Song」 hoặc Artist「Song」
var regexJaTitle = regexp.MustCompile(`「([^」]+)」`)
var regexJaBracket = regexp.MustCompile(`【[^】]*】`)

func splitJapaneseBracket(title string) (track, artist string) {
	m := regexJaTitle.FindStringSubmatch(title)
	if m == nil {
		return "", ""
	}
	track = strings.TrimSpace(m[1])
	// Artist: bỏ ngoặc 【...】 và phần 「...」 ra khỏi title
	art := regexJaBracket.ReplaceAllString(title, "")
	art = regexJaTitle.ReplaceAllString(art, "")
	artist = strings.TrimSpace(art)
	return track, artist
}

// regexNoisyParens khớp các cụm ngoặc chứa noise keywords phổ biến.
var regexNoisyParens = regexp.MustCompile(`(?i)\s*[\(\[【][^\)\]】]*(official|video|mv|m\/v|music|audio|lyric|visualizer|live|4k|hd|1080p|720p|prod|cover|remix|amv|piano|arrangement|version|ver\.?)[^\)\]】]*[\)\]】]`)

// stripNoiseParens loại bỏ các cụm ngoặc chứa từ khóa rác.
func stripNoiseParens(s string) string {
	result := regexNoisyParens.ReplaceAllString(s, "")
	return normalizeSpaces(result)
}

// removeTrashKeywords loại bỏ các từ khóa rác đứng riêng lẻ (không trong ngoặc).
var regexTrashWords = regexp.MustCompile(`(?i)\b(official|music video|lyric video|lyrics|visualizer|audio|4k|1080p|720p|hd)\b`)

func removeTrashKeywords(s string) string {
	result := regexTrashWords.ReplaceAllString(s, "")
	return normalizeSpaces(result)
}

// simplifyArtist bỏ phần feat/ft/x/& và các nghệ sĩ phụ.
var regexFeat = regexp.MustCompile(`(?i)\s*(feat\.?|ft\.?|featuring|×)\s+.+`)
var regexAnd = regexp.MustCompile(`(?i)\s*(&|and)\s+.+`)

func simplifyArtist(artist string) string {
	result := regexFeat.ReplaceAllString(artist, "")
	result = regexAnd.ReplaceAllString(result, "")
	return normalizeSpaces(result)
}

// firstOf trả về s nếu không rỗng, ngược lại trả fallback.
func firstOf(s, fallback string) string {
	if strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s)
	}
	return fallback
}

var regexSpaces = regexp.MustCompile(`\s+`)

func normalizeSpaces(s string) string {
	s = regexSpaces.ReplaceAllString(s, " ")
	return strings.Trim(s, " -|,.")
}
