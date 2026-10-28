package scraper

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
)

// lyricsCandidate carries the origin and confidence of one LRCLib search
// hypothesis. Higher priority candidates are attempted first.
type lyricsCandidate struct {
	track    string
	artist   string
	source   string
	priority int
}

func buildCandidateList(meta MusicMetadata, rawTitle string) []lyricsCandidate {
	var out []lyricsCandidate
	byKey := make(map[string]int)

	add := func(track, artist, source string, priority int) {
		track = normalizeSpaces(track)
		artist = normalizeSpaces(artist)
		if track == "" {
			return
		}

		key := strings.ToLower(track + "|" + artist)
		if index, ok := byKey[key]; ok {
			if priority > out[index].priority {
				out[index].priority = priority
				out[index].source = source
			}
			return
		}
		byKey[key] = len(out)
		out = append(out, lyricsCandidate{
			track:    track,
			artist:   artist,
			source:   source,
			priority: priority,
		})
	}

	if meta.Track != "" {
		add(meta.Track, meta.Artist, "metadata", 130)
	}
	for _, parsed := range buildTitleCandidates(rawTitle, meta.Artist) {
		add(parsed.track, parsed.artist, "title:"+parsed.source, parsed.priority)
	}

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].priority > out[j].priority
	})
	return out
}

// FetchLyrics evaluates high-confidence hypotheses in small parallel batches.
// It ranks successful candidates by parser priority rather than returning the
// first network response, while keeping total concurrency and request time bounded.
const (
	maxLRCLibCandidates = 8
	lyricsCandidateBatchSize = 3
)

func FetchLyrics(ctx context.Context, meta MusicMetadata, rawTitle string, durationSec float64) ([]LyricsData, error) {
	candidates := buildCandidateList(meta, rawTitle)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("lrclib: no usable title candidates (metadata and title are empty)")
	}
	if len(candidates) > maxLRCLibCandidates {
		candidates = candidates[:maxLRCLibCandidates]
	}

	type candResult struct {
		candidate lyricsCandidate
		index     int
		data      []LyricsData
		err       error
	}

	var lastErr error
	for start := 0; start < len(candidates); start += lyricsCandidateBatchSize {
		end := start + lyricsCandidateBatchSize
		if end > len(candidates) {
			end = len(candidates)
		}
		batch := candidates[start:end]
		results := make(chan candResult, len(batch))

		for i, candidate := range batch {
			go func(candidate lyricsCandidate, index int) {
				data, err := lookupLRCLibCandidate(ctx, candidate.track, candidate.artist, durationSec)
				results <- candResult{
					candidate: candidate,
					index:     index,
					data:      data,
					err:       err,
				}
			}(candidate, i)
		}

		var best *candResult
		for range batch {
			select {
			case result := <-results:
				if result.err != nil {
					lastErr = result.err
					continue
				}
				if len(result.data) == 0 {
					continue
				}
				if best == nil ||
					result.candidate.priority > best.candidate.priority ||
					(result.candidate.priority == best.candidate.priority && result.index < best.index) {
					copy := result
					best = &copy
				}
			case <-ctx.Done():
				return nil, fmt.Errorf("lrclib: lookup canceled or timed out for %q: %w", rawTitle, ctx.Err())
			}
		}

		if best != nil {
			log.Printf(
				"lrclib: matched via %s candidate track=%q artist=%q priority=%d",
				best.candidate.source,
				best.candidate.track,
				best.candidate.artist,
				best.candidate.priority,
			)
			return best.data, nil
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("lrclib: no confident match for %q (%w)", rawTitle, lastErr)
	}
	return nil, fmt.Errorf("lrclib: no confident match for %q", rawTitle)
}
