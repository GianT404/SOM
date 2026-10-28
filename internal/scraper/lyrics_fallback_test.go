package scraper

import (
	"strings"
	"testing"
)

func TestBuildTitleCandidates_AnimePianoTitle(t *testing.T) {
	title := "Crying for Rain (Kawaki wo Ameku) - Domestic na Kanojo OP [Piano] / Minami"
	candidates := buildTitleCandidates(title, "")

	requireCandidate(t, candidates, "Crying for Rain", "Minami")
	requireCandidate(t, candidates, "Kawaki wo Ameku", "Minami")
	requireCandidate(t, candidates, "Crying for Rain (Kawaki wo Ameku)", "Minami")

	// Good hypotheses must survive the LRCLib candidate cap.
	for _, track := range []string{"Crying for Rain", "Kawaki wo Ameku"} {
		index := candidateIndex(candidates, track, "Minami")
		if index >= maxLRCLibCandidates {
			t.Errorf("candidate %q + %q at index %d, beyond max candidate count %d", track, "Minami", index, maxLRCLibCandidates)
		}
	}
}

func TestBuildTitleCandidates_CommonFormats(t *testing.T) {
	tests := []struct {
		name       string
		title      string
		artist     string
		wantTrack  string
		wantArtist string
	}{
		{
			name:       "artist dash song with video noise",
			title:      "Minami - Kawaki wo Ameku (Official Music Video) [1080p]",
			wantTrack:  "Kawaki wo Ameku",
			wantArtist: "Minami",
		},
		{
			name:       "song pipe artist and channel",
			title:      "夜に駆ける | YOASOBI | YOASOBI Official Channel",
			wantTrack:  "夜に駆ける",
			wantArtist: "YOASOBI",
		},
		{
			name:       "Japanese bracket format",
			title:      "【MV】Minami「カワキヲアメク」",
			wantTrack:  "カワキヲアメク",
			wantArtist: "Minami",
		},
		{
			name:       "context suffix removed without dropping title",
			title:      "Crying for Rain (Kawaki wo Ameku) - Domestic na Kanojo OP",
			artist:     "Minami",
			wantTrack:  "Crying for Rain (Kawaki wo Ameku)",
			wantArtist: "Minami",
		},
		{
			name:       "fullwidth parentheses preserve romanized alias",
			title:      "Song Title（Romanized Title） [Official Audio]",
			artist:     "Artist",
			wantTrack:  "Romanized Title",
			wantArtist: "Artist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates := buildTitleCandidates(tt.title, tt.artist)
			requireCandidate(t, candidates, tt.wantTrack, tt.wantArtist)
		})
	}
}

func TestBuildTitleCandidates_RetainsOriginalAsFallback(t *testing.T) {
	rawTitle := "Some Song [Fan Upload] / Random Channel"
	candidates := buildTitleCandidates(rawTitle, "")
	if candidateIndex(candidates, rawTitle, "") < 0 {
		t.Fatalf("raw title should remain available as a fallback candidate")
	}
}

func TestBuildCandidateList_PrioritizesMusicMetadata(t *testing.T) {
	candidates := buildCandidateList(MusicMetadata{
		Track:  "Known Track",
		Artist: "Known Artist",
	}, "Known Track [Official Video]")

	if len(candidates) == 0 {
		t.Fatal("expected metadata candidate")
	}
	if candidates[0].track != "Known Track" || candidates[0].artist != "Known Artist" {
		t.Fatalf("first candidate = (%q, %q), want metadata candidate", candidates[0].track, candidates[0].artist)
	}
}

func requireCandidate(t *testing.T, candidates []candidate, track, artist string) {
	t.Helper()
	if index := candidateIndex(candidates, track, artist); index < 0 {
		var got []string
		for _, c := range candidates {
			got = append(got, c.track+" | "+c.artist+" ("+c.source+")")
		}
		t.Fatalf("candidate (%q, %q) not found; got: %s", track, artist, strings.Join(got, "; "))
	}
}

func candidateIndex(candidates []candidate, track, artist string) int {
	for i, c := range candidates {
		if strings.EqualFold(normalizeSpaces(c.track), normalizeSpaces(track)) &&
			strings.EqualFold(normalizeSpaces(c.artist), normalizeSpaces(artist)) {
			return i
		}
	}
	return -1
}
