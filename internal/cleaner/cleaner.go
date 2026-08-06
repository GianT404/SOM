package cleaner

import (
	"regexp"
	"strings"
)

var (
	regexRoundBrackets  = regexp.MustCompile(`\([^)]*\)`)
	regexSquareBrackets = regexp.MustCompile(`\[[^\]]*\]`)
	regexAngleBrackets  = regexp.MustCompile(`【[^】]*】`)

	// Tối ưu từ khóa rác đứng riêng lẻ
	regexTrashKeywords = regexp.MustCompile(
		`(?i)(\b(official|video|lyric|audio|visualizer|live|4k|1080p|720p|hd|prod|cover|remix|mv|m/v|music|amv)\b)`,
	)

	// Xử lý triệt để ft/feat và tên nghệ sĩ đi kèm phía sau
	regexFeaturingTag = regexp.MustCompile(`(?i)\s*\b(ft|feat)\.?\s+[^-\|]+`)

	regexExtraSpaces = regexp.MustCompile(`\s+`)
	// Tách tiêu đề dựa trên các dấu phân cách phổ biến
	regexSeparator = regexp.MustCompile(`\s*\|\s*|\s+-\s+`)
)

var trashKeywords = map[string]bool{
	"official": true, "video": true, "lyric": true, "audio": true,
	"visualizer": true, "live": true, "4k": true, "1080p": true,
	"720p": true, "hd": true, "prod": true, "cover": true,
	"remix": true, "mv": true, "m/v": true, "music": true, "amv": true,
}

func containsTrashKeyword(text string) bool {
	lowerText := strings.ToLower(text)
	for keyword := range trashKeywords {
		if strings.Contains(lowerText, keyword) {
			return true
		}
	}
	return false
}

// CleanYouTubeTitle làm sạch tiêu đề bài hát từ YouTube
func CleanYouTubeTitle(rawTitle string) string {
	result := rawTitle

	// 1. Loại bỏ các cụm từ trong ngoặc nếu chứa từ khóa rác
	result = regexRoundBrackets.ReplaceAllStringFunc(result, func(match string) string {
		if containsTrashKeyword(match) {
			return ""
		}
		return match
	})
	result = regexSquareBrackets.ReplaceAllStringFunc(result, func(match string) string {
		if containsTrashKeyword(match) {
			return ""
		}
		return match
	})
	result = regexAngleBrackets.ReplaceAllStringFunc(result, func(match string) string {
		if containsTrashKeyword(match) {
			return ""
		}
		return match
	})

	// 2. Xóa bỏ phần featuring (Ví dụ: "ft. Snoop Dogg") trước khi băm chuỗi
	result = regexFeaturingTag.ReplaceAllString(result, "")

	// 3. Băm tiêu đề thành các phần riêng biệt dựa trên dấu | hoặc -
	parts := regexSeparator.Split(result, -1)

	var cleanParts []string
	for _, part := range parts {
		// Dọn dẹp từ khóa rác đứng riêng lẻ trong từng phần
		p := regexTrashKeywords.ReplaceAllString(part, "")
		p = strings.TrimSpace(p)
		// Chỉ giữ lại phần nào còn nội dung và không rỗng
		if p != "" {
			cleanParts = append(cleanParts, p)
		}
	}

	// 4. CHIẾN THUẬT LỌC TÊN BÀI HÁT:
	// Giả định phổ biến: Nếu có dạng "Ca sĩ | Tên Bài Hát", ta muốn lấy tên bài hát.
	// Thường tên ca sĩ nổi tiếng (như SƠN TÙNG M-TP) sẽ nằm ở phần đầu tiên.
	if len(cleanParts) >= 2 {
		// Nếu bạn chỉ muốn lấy TÊN BÀI HÁT (bỏ tên nghệ sĩ chính ở phần đầu):
		result = cleanParts[1]
	} else if len(cleanParts) == 1 {
		result = cleanParts[0]
	} else {
		result = ""
	}

	// 5. Dọn dẹp khoảng trắng dư thừa cuối cùng
	result = regexExtraSpaces.ReplaceAllString(result, " ")
	result = strings.Trim(result, " -|,,.")
	result = strings.TrimSpace(result)

	return result
}
