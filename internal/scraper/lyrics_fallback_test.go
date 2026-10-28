package scraper

import (
	"strings"
	"testing"
)

func TestBuildTitleCandidates_RemovesBracketContents(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{"square brackets", "Song [Official MV]", "Song"},
		{"curly braces", "Song {fan upload}", "Song"},
		{"parentheses", "Song (romanized title)", "Song"},
		{"Japanese square brackets", "Song 【live version】", "Song"},
		{"fullwidth brackets", "Song ［metadata］ ｛extra｝ （alias）", "Song"},
		{"nested brackets", "Song ([inner] {metadata})", "Song"},
		{"quoted track words are preserved", "Don't say \"lazy\"", "Don't say lazy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates := buildTitleCandidates(tt.title, "")
			requireCandidate(t, candidates, tt.want, "")
			for _, candidate := range candidates {
				for _, blocked := range []string{"Official MV", "fan upload", "romanized title", "metadata", "extra", "alias", "inner"} {
					if strings.Contains(strings.ToLower(candidate.track), strings.ToLower(blocked)) {
						t.Errorf("candidate %q still contains bracket content %q", candidate.track, blocked)
					}
				}
			}
		})
	}
}

func TestBuildTitleCandidates_AnimePianoTitle(t *testing.T) {
	title := "Crying for Rain (Kawaki wo Ameku) - Domestic na Kanojo OP [Piano] / Minami"
	candidates := buildTitleCandidates(title, "")
	requireCandidate(t, candidates, "Crying for Rain", "Minami")
	assertNoCandidateContains(t, candidates, "Kawaki wo Ameku")
}

func TestBuildTitleCandidates_CoverCreditSuffix(t *testing.T) {
	title := "Minami (美波) - Crying for Rain (カワキヲアメク) Live Band Cover by AARCADE"
	candidates := buildTitleCandidates(title, "")
	requireCandidate(t, candidates, "Crying for Rain", "Minami")
	assertNoCandidateContains(t, candidates, "カワキヲアメク")
	assertNoCandidateContains(t, candidates, "美波")
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
			name: "artist dash song with video noise",
			title: "Minami - Kawaki wo Ameku (Official Music Video) [1080p]",
			wantTrack: "Kawaki wo Ameku",
			wantArtist: "Minami",
		},
		{
			name: "song pipe artist and channel",
			title: "夜に駆ける | YOASOBI | YOASOBI Official Channel",
			wantTrack: "夜に駆ける",
			wantArtist: "YOASOBI",
		},
		{
			name: "Japanese bracket format",
			title: "【MV】Minami「カワキヲアメク」",
			wantTrack: "カワキヲアメク",
			wantArtist: "Minami",
		},
		{
			name: "anime context removed",
			title: "Crying for Rain (Kawaki wo Ameku) - Domestic na Kanojo OP",
			artist: "Minami",
			wantTrack: "Crying for Rain",
			wantArtist: "Minami",
		},
		{
			name: "fullwidth parentheses removed",
			title: "Song Title（Romanized Title） [Official Audio]",
			artist: "Artist",
			wantTrack: "Song Title",
			wantArtist: "Artist",
		},
		{
			name: "version suffix removed",
			title: "Song Title Piano Cover by Channel Name",
			artist: "Artist",
			wantTrack: "Song Title",
			wantArtist: "Artist",
		},
		{
			name: "noise word inside plain title preserved",
			title: "Cover Me in Rain",
			wantTrack: "Cover Me in Rain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidates := buildTitleCandidates(tt.title, tt.artist)
			requireCandidate(t, candidates, tt.wantTrack, tt.wantArtist)
		})
	}
}

func TestBuildTitleCandidates_FallbackDoesNotRestoreBracketContents(t *testing.T) {
	rawTitle := "Some Song [Fan Upload] / Random Channel"
	candidates := buildTitleCandidates(rawTitle, "")
	requireCandidate(t, candidates, "Some Song / Random Channel", "")
	assertNoCandidateContains(t, candidates, "Fan Upload")
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

func assertNoCandidateContains(t *testing.T, candidates []candidate, value string) {
	t.Helper()
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(candidate.track+" "+candidate.artist), strings.ToLower(value)) {
			t.Errorf("candidate %+v still contains %q", candidate, value)
		}
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
