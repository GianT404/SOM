package cleaner
//clean YouTube title by removing common "trash" keywords and patterns that interfere with lyric search
import (
	"regexp"
	"strings"
)

var (
	regexRoundBrackets = regexp.MustCompile(`\([^)]*\)`)
	
	regexSquareBrackets = regexp.MustCompile(`\[[^\]]*\]`)
	
	regexAngleBrackets = regexp.MustCompile(`【[^】]*】`)
	
	regexTrashKeywords = regexp.MustCompile(
		`(?i)(\b(official|video|lyric|audio|visualizer|live|4k|1080p|720p|hd|prod|cover|remix|mv|m/v|music|amv|ft|feat)\b)`,
	)
	
	regexFeaturingTag = regexp.MustCompile(`(?i)\s+(ft|feat)\.\s+[^\[\(\|]*`)
	
	regexExtraSpaces = regexp.MustCompile(`\s+`)
	regexSeparator = regexp.MustCompile(`\s*\|\s*|\s+-\s+`)
	
)

// Danh sách từ khóa rác để kiểm tra trong ngoặc
var trashKeywords = map[string]bool{
	"official":   true,
	"video":      true,
	"lyric":      true,
	"audio":      true,
	"visualizer": true,
	"live":       true,
	"4k":         true,
	"1080p":      true,
	"720p":       true,
	"hd":         true,
	"ft":         true,
	"feat":       true,
	"prod":       true,
	"cover":      true,
	"remix":      true,
	"mv":         true,
	"m/v":        true,
	"music":      true,
	"amv":        true,
}

// containsTrashKeyword kiểm tra xem chuỗi có chứa từ khóa rác không
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
// Loại bỏ từ khóa rác, cụm từ trong ngoặc không cần thiết, và ký tự đặc biệt thừa
func CleanYouTubeTitle(rawTitle string) string {
	result := rawTitle

	// 1. Loại bỏ cụm từ trong ngoặc đơn nếu chứa từ khóa rác
	result = regexRoundBrackets.ReplaceAllStringFunc(result, func(match string) string {
		if containsTrashKeyword(match) {
			return ""
		}
		return match
	})

	// 2. Loại bỏ cụm từ trong ngoặc vuông
	result = regexSquareBrackets.ReplaceAllStringFunc(result, func(match string) string {
		if containsTrashKeyword(match) {
			return ""
		}
		return match
	})

	// 3. Loại bỏ cụm từ trong ngoặc kép góc
	result = regexAngleBrackets.ReplaceAllStringFunc(result, func(match string) string {
		if containsTrashKeyword(match) {
			return ""
		}
		return match
	})

	// 4. Loại bỏ tag featuring
	result = regexFeaturingTag.ReplaceAllString(result, "")

	// 5. Loại bỏ từ khóa rác đứng riêng
	result = regexTrashKeywords.ReplaceAllString(result, "")

	// 6. TUYỆT KỸ CẮT ĐUÔI (Băm nhỏ và bỏ phần dư)
	parts := regexSeparator.Split(result, -1)
	if len(parts) > 2 {
		// Nếu có từ 3 phần trở lên (VD: Bài hát | Ca sĩ | Tên Kênh), chỉ lấy 2 phần đầu
		result = parts[0] + " " + parts[1]
	} else {
		// Nếu có 1 hoặc 2 phần, gom lại bằng khoảng trắng cho sạch dấu phân cách
		result = strings.Join(parts, " ")
	}

	// 7. Dọn dẹp chiến trường: Làm sạch khoảng trắng và ký tự đặc biệt rớt lại
	result = regexExtraSpaces.ReplaceAllString(result, " ")
	result = strings.Trim(result, " -|,")
	result = strings.TrimSpace(result)

	return result
}