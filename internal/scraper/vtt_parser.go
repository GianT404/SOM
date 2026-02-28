package scraper

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	// Matches VTT/SRT timestamp lines: "00:01:23.456 --> 00:01:25.789"
	reTimestamp = regexp.MustCompile(
		`(\d{1,2}:)?(\d{2}):(\d{2})[.,](\d{3})\s*-->\s*(\d{1,2}:)?(\d{2}):(\d{2})[.,](\d{3})`,
	)

	// Matches HTML-like tags: <c>, </c>, <b>, <00:01:23.456>, etc.
	reHTMLTag = regexp.MustCompile(`<[^>]+>`)
)

// ParseVTT parses WebVTT or SRT text into a slice of LyricLines.
// It deduplicates consecutive identical text lines and strips markup.
func ParseVTT(raw string) []LyricLine {
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")

	var result []LyricLine
	var lastText string

	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// Try to match a timestamp line.
		match := reTimestamp.FindStringSubmatch(line)
		if match == nil {
			i++
			continue
		}

		startSec := parseTimeParts(match[1], match[2], match[3], match[4])
		endSec := parseTimeParts(match[5], match[6], match[7], match[8])
		i++

		// Collect text lines until the next blank line or timestamp.
		var textParts []string
		for i < len(lines) {
			tl := strings.TrimSpace(lines[i])
			if tl == "" {
				i++
				break
			}
			if reTimestamp.MatchString(tl) {
				break
			}
			// Strip HTML tags and VTT cue tags.
			cleaned := reHTMLTag.ReplaceAllString(tl, "")
			cleaned = strings.TrimSpace(cleaned)
			if cleaned != "" {
				textParts = append(textParts, cleaned)
			}
			i++
		}

		text := strings.Join(textParts, " ")
		if text == "" || text == lastText {
			continue // skip empty or duplicate consecutive lines
		}

		lastText = text
		result = append(result, LyricLine{
			Start: startSec,
			End:   endSec,
			Text:  text,
		})
	}

	return result
}

// parseTimeParts converts captured timestamp groups into seconds as float64.
// hours may be empty string (e.g., "01:" or "").
func parseTimeParts(hours, minutes, seconds, millis string) float64 {
	h := 0
	if hours != "" {
		hours = strings.TrimSuffix(hours, ":")
		h, _ = strconv.Atoi(hours)
	}
	m, _ := strconv.Atoi(minutes)
	s, _ := strconv.Atoi(seconds)
	ms, _ := strconv.Atoi(millis)

	return float64(h)*3600 + float64(m)*60 + float64(s) + float64(ms)/1000.0
}
