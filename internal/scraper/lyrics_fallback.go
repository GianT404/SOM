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

// buildTitleCandidates generates independent search hypotheses. It intentionally
// keeps the raw title as a low-priority fallback instead of trusting one parser
// to understand every YouTube title convention.
func buildTitleCandidates(rawTitle, artist string) []candidate {
	var out []candidate
	byKey := make(map[string]int)

	add := func(track, art, source string, priority int) {
		track = cleanTrackCandidate(track)
		art = normalizeSpaces(art)
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

	raw := normalizeSpaces(rawTitle)
	if raw == "" {
		return nil
	}

	// Explicit suffixes are usually more informative than a generic dash
	// split: "Song / Artist" and "Song | Artist | Channel".
	withoutBracketNoise := stripNoiseParens(raw)
	if trackPart, suffixArtist, ok := splitSuffixArtist(withoutBracketNoise, "/", "／"); ok {
		trackPart = stripTrailingContextSegments(trackPart)
		add(trackPart, suffixArtist, "slash_suffix", 120)
		addParentheticalVariants(trackPart, suffixArtist, add, 112)
		// Some uploaders use the inverse convention; retain it as a fallback.
		add(suffixArtist, trackPart, "slash_reverse", 55)
	}

	if trackPart, suffixArtist := splitPipe(raw); suffixArtist != "" && !isContextNoise(suffixArtist) {
		add(trackPart, suffixArtist, "pipe", 108)
		if artist != "" {
			add(trackPart, artist, "pipe_metadata_artist", 104)
		}
		addParentheticalVariants(trackPart, firstOf(suffixArtist, artist), add, 100)
	}

	// Japanese title convention: Artist「Track」, optionally prefixed by 【MV】.
	if trackPart, parsedArtist := splitJapaneseBracket(raw); trackPart != "" {
		add(trackPart, firstOf(parsedArtist, artist), "japanese_bracket", 110)
		add(trackPart, artist, "japanese_bracket_metadata_artist", 105)
	}

	// Remove only recognizable video/arrangement context. A parenthesized alias
	// that does not contain noise keywords is preserved and also searched alone.
	cleaned := stripTrailingContextSegments(stripNoiseParens(raw))
	cleaned = removeTrashKeywords(cleaned)
	if cleaned != "" {
		add(cleaned, artist, "clean_title", 96)
		addParentheticalVariants(cleaned, artist, add, 94)
	}

	// Dash titles are ambiguous ("Artist - Song", "Song - Artist", "Song -
	// Anime OP"). Try both orientations, preferring the one supported by a known
	// artist or by the conservative legacy length heuristic.
	for _, separator := range []string{" - ", " – ", " — "} {
		parts := strings.SplitN(cleaned, separator, 2)
		if len(parts) != 2 {
			continue
		}
		left := cleanTrackCandidate(stripTrailingContextSegments(parts[0]))
		right := cleanTrackCandidate(stripTrailingContextSegments(parts[1]))
		if left == "" || right == "" {
			continue
		}

		switch {
		case artist != "" && sameTitle(left, artist):
			add(right, artist, "dash_known_artist_left", 116)
		case artist != "" && sameTitle(right, artist):
			add(left, artist, "dash_known_artist_right", 116)
		default:
			if utf8.RuneCountInString(left) <= utf8.RuneCountInString(right) {
				add(right, left, "dash_length_heuristic", 82)
				add(left, right, "dash_reverse", 68)
			} else {
				add(left, right, "dash_length_heuristic", 82)
				add(right, left, "dash_reverse", 68)
			}
		}

		// The supplied artist may come from the uploader rather than the
		// recording metadata, so try both title halves with it at lower priority.
		add(left, artist, "dash_left_with_hint", 76)
		add(right, artist, "dash_right_with_hint", 74)
	}

	// Always retain a cleaned whole-title search and the untouched input for
	// uncommon naming conventions that our heuristics do not recognize.
	add(removeTrashKeywords(stripNoiseParens(raw)), artist, "clean_fallback", 45)
	add(raw, artist, "raw_fallback", 5)

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].priority > out[j].priority
	})
	return out
}

func addParentheticalVariants(title, artist string, add func(string, string, string, int), priority int) {
	for _, match := range regexTitleAliases.FindAllStringSubmatch(title, -1) {
		if len(match) < 2 {
			continue
		}
		alias := match[1]
		if alias == "" && len(match) > 2 {
			alias = match[2]
		}
		alias = normalizeSpaces(alias)
		if alias == "" || isContextNoise(alias) {
			continue
		}
		base := regexTitleAliases.ReplaceAllString(title, "")
		base = cleanTrackCandidate(base)
		if base != "" {
			add(base, artist, "title_without_alias", priority)
		}
		add(alias, artist, "parenthetical_alias", priority-4)
	}
}

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

func isLikelyArtist(value string) bool {
	value = normalizeSpaces(value)
	if value == "" || isContextNoise(value) || utf8.RuneCountInString(value) > 48 {
		return false
	}
	return len(strings.Fields(value)) <= 5
}

func stripTrailingContextSegments(title string) string {
	title = normalizeSpaces(title)
	for title != "" {
		index, separator := lastTitleSeparator(title)
		if index < 0 {
			break
		}
		suffix := strings.TrimSpace(title[index+len(separator):])
		if !isContextNoise(suffix) {
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

func isContextNoise(value string) bool {
	return regexContextNoise.MatchString(value)
}

func cleanTrackCandidate(value string) string {
	value = stripNoiseParens(value)
	value = removeTrashKeywords(value)
	return normalizeSpaces(value)
}

func sameTitle(a, b string) bool {
	return strings.EqualFold(normalizeSpaces(a), normalizeSpaces(b))
}

// splitPipe supports "Song | Artist" and "Song | Artist | Channel/Label".
func splitPipe(title string) (track, artist string) {
	clean := regexNoisyParens.ReplaceAllString(title, "")
	var parts []string
	for _, part := range strings.Split(clean, "|") {
		if part = strings.TrimSpace(part); part != "" {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

// splitJapaneseBracket handles 【MV】Artist「Track」 and Artist「Track」.
var (
	regexJaTitle      = regexp.MustCompile(`「([^」]+)」`)
	regexJaBracket    = regexp.MustCompile(`【[^】]*】`)
	regexTitleAliases = regexp.MustCompile(`\(([^()]+)\)|（([^（）]+)）`)
	regexNoisyParens  = regexp.MustCompile(`(?i)\s*[\(\[【][^\)\]】]*(official|video|mv|m/v|music|audio|lyric|visualizer|live|4k|hd|1080p|720p|prod|cover|remix|amv|piano|arrangement|version|ver\.?|instrumental|karaoke|creditless)[^\)\]】]*[\)\]】]`)
	regexTrashWords   = regexp.MustCompile(`(?i)\b(official music video|official video|official audio|lyric video|music video|full version|visualizer|lyrics?|official|audio|4k|1080p|720p|hd|creditless)\b`)
	regexContextNoise = regexp.MustCompile(`(?i)\b(?:op|ed)\s*\d*\b|\b(?:opening|ending)(?:\s+(?:theme|song|credits?))?\b|\b(?:official|lyrics?|visualizer|piano|instrumental|karaoke|cover|remix|arrangement|1080p|720p|4k|amv|creditless)\b`)
	regexSpaces       = regexp.MustCompile(`\s+`)
)

func splitJapaneseBracket(title string) (track, artist string) {
	match := regexJaTitle.FindStringSubmatch(title)
	if match == nil {
		return "", ""
	}
	track = strings.TrimSpace(match[1])
	artist = regexJaBracket.ReplaceAllString(title, "")
	artist = regexJaTitle.ReplaceAllString(artist, "")
	artist = strings.TrimSpace(artist)
	artist = cleanTrackCandidate(artist)
	return track, artist
}

func stripNoiseParens(value string) string {
	return normalizeSpaces(regexNoisyParens.ReplaceAllString(value, ""))
}

func removeTrashKeywords(value string) string {
	return normalizeSpaces(regexTrashWords.ReplaceAllString(value, ""))
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
