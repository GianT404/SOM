package ui

import (
	"testing"

	"som/internal/domain"
)

func TestRightPanelIgnoresStaleTrackChange(t *testing.T) {
	r := RightPanel{
		nowPlay: &domain.Track{ID: "current", Title: "Current"},
		trackGen: 5,
	}

	r, _ = r.Update(TrackChangedMsg{
		Track: domain.Track{ID: "old", Title: "Old"},
		Gen:   4,
	}, false)

	if r.nowPlay == nil || r.nowPlay.ID != "current" {
		t.Fatalf("now playing=%v, want current", r.nowPlay)
	}
	if r.trackGen != 5 {
		t.Fatalf("track generation=%d, want 5", r.trackGen)
	}
}

func TestRightPanelIgnoresStaleLyrics(t *testing.T) {
	r := RightPanel{trackGen: 5}

	r, _ = r.Update(StreamResolvedMsg{
		Gen: 4,
		Lyrics: domain.LyricsResp{Plain: "stale lyrics"},
	}, false)
	if r.loaded {
		t.Fatal("stale stream result should not load lyrics")
	}

	r, _ = r.Update(LocalLyricsLoadedMsg{
		Gen:    4,
		Lyrics: domain.LyricsResp{Plain: "stale local lyrics"},
	}, false)
	if r.loaded {
		t.Fatal("stale local lyrics should not load")
	}
}

func TestRightPanelAcceptsCurrentGenerationLyrics(t *testing.T) {
	r := RightPanel{trackGen: 5}

	r, _ = r.Update(StreamResolvedMsg{
		Gen:    5,
		Lyrics: domain.LyricsResp{Plain: "current lyrics"},
	}, false)
	if !r.loaded {
		t.Fatal("current stream result should load lyrics")
	}
	if r.lyrics.Plain != "current lyrics" {
		t.Fatalf("lyrics=%q, want current lyrics", r.lyrics.Plain)
	}
}
