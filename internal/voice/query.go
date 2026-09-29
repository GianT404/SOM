package voice

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

func ExtractSearchQuery(transcript string) string {
	original := strings.Fields(strings.TrimSpace(transcript))
	if len(original) == 0 {
		return ""
	}

	folded := strings.Fields(foldForMatch(transcript))
	if len(folded) != len(original) {
		return strings.Join(original, " ")
	}

	prefixes := [][]string{
		{"tim", "kiem"},
		{"tim", "bai", "hat"},
		{"tim", "bai"},
		{"search", "for"},
		{"find", "song"},
		{"search"},
		{"find"},
		{"sop", "bai"},
		{"sop"},
	}

	for _, prefix := range prefixes {
		if len(folded) < len(prefix) {
			continue
		}
		match := true
		for i := range prefix {
			if folded[i] != prefix[i] {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		return cleanSearchTail(original[len(prefix):], folded[len(prefix):])
	}

	switch folded[0] {
	case "sop", "tim", "search", "find":
		return cleanSearchTail(original[1:], folded[1:])
	}

	return strings.Join(original, " ")
}

func cleanSearchTail(original, folded []string) string {
	offset := 0
	if len(folded) >= 2 && folded[0] == "bai" && folded[1] == "hat" {
		offset = 2
	} else if len(folded) >= 1 && (folded[0] == "bai" || folded[0] == "song") {
		offset = 1
	}
	if offset >= len(original) {
		return ""
	}
	return strings.TrimSpace(strings.Join(original[offset:], " "))
}

func foldForMatch(text string) string {
	text = strings.ToLower(text)
	text = strings.NewReplacer(
		"đ", "d",
		"ð", "d",
	).Replace(text)

	decomposed := norm.NFD.String(text)
	var b strings.Builder
	b.Grow(len(text))

	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}

	result := strings.TrimSpace(strings.Join(strings.Fields(b.String()), " "))
	if !utf8.ValidString(result) {
		return ""
	}
	return result
}
