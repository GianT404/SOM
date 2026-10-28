package ui

import (
	"strings"
	"testing"

	"som/internal/domain"

	tea "charm.land/bubbletea/v2"
)

func TestLyricsSearchModalEnterPreviewsBeforeApplying(t *testing.T) {
	modal := NewLyricsSearchModal(nil, domain.Track{ID: "video-1", Title: "Current track"}, 90, 30)
	modal.stage = lyricsSearchResults
	modal.results = []domain.LyricsSearchResult{{
		TrackName:  "Candidate track",
		ArtistName: "Candidate artist",
		Lyrics: domain.LyricsTrack{
			Language: "lrclib",
			Synced: []domain.LyricLine{
				{Text: "Preview this lyric first"},
			},
		},
	}}

	updated, cmd := modal.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := updated.(*LyricsSearchModal)
	if got.stage != lyricsSearchPreview {
		t.Fatalf("stage after first Enter = %v, want preview", got.stage)
	}
	if cmd != nil {
		t.Fatal("first Enter must not emit an apply message")
	}
	if !strings.Contains(got.View(), "Preview this lyric first") {
		t.Fatal("preview view does not contain the selected lyrics")
	}

	_, cmd = got.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("second Enter should emit an apply message")
	}
	rawMsg := cmd()
	msg, ok := rawMsg.(ManualLyricsChosenMsg)
	if !ok {
		t.Fatalf("second Enter message = %T, want ManualLyricsChosenMsg", rawMsg)
	}
	if msg.VideoID != "video-1" || msg.Result.TrackName != "Candidate track" {
		t.Fatalf("apply message = %#v, unexpected target", msg)
	}
}

func TestLyricsSearchModalEscapeFromPreviewReturnsToResults(t *testing.T) {
	modal := NewLyricsSearchModal(nil, domain.Track{ID: "video-1", Title: "Current track"}, 90, 30)
	modal.stage = lyricsSearchPreview
	modal.preview = 0
	modal.cursor = 0
	modal.results = []domain.LyricsSearchResult{{TrackName: "Candidate"}}

	updated, cmd := modal.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	got := updated.(*LyricsSearchModal)
	if got.stage != lyricsSearchResults {
		t.Fatalf("stage after Esc = %v, want results", got.stage)
	}
	if got.preview != -1 {
		t.Fatalf("preview index after Esc = %d, want -1", got.preview)
	}
	if cmd != nil {
		t.Fatal("Esc back to results should not emit a command")
	}
}
