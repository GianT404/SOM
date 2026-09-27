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


func TestLeftPanelIgnoresStaleTrackChange(t *testing.T) {
	p := LeftPanel{
		tracks:   []domain.Track{{ID: "current"}},
		trackGen: 5,
	}
	p.searchCursor = 0

	p, _ = p.Update(TrackChangedMsg{
		Track: domain.Track{ID: "old"},
		Gen:   4,
	}, false, nil)

	if p.trackGen != 5 {
		t.Fatalf("track generation=%d, want 5", p.trackGen)
	}
	if p.searchCursor != 0 {
		t.Fatalf("search cursor=%d, want 0", p.searchCursor)
	}
}

func TestLeftPanelIgnoresStaleStreamResult(t *testing.T) {
	p := LeftPanel{
		activeTab: SideSearch,
		trackGen:  5,
		loadingStream: true,
	}

	p, _ = p.Update(StreamResolvedMsg{Gen: 4}, false, nil)
	if !p.loadingStream {
		t.Fatal("stale stream result should not clear loading state")
	}

	p, _ = p.Update(StreamResolvedMsg{Gen: 5}, false, nil)
	if p.loadingStream {
		t.Fatal("current stream result should clear loading state")
	}
}
