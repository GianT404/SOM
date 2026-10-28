package scraper

import "testing"

func TestRankLRCLibSearchResults_PrioritizesSyncedAndLimitsResults(t *testing.T) {
	tracks := make([]lrclibTrack, 0, 10)
	for i := 0; i < 10; i++ {
		tracks = append(tracks, lrclibTrack{ID: i + 1, TrackName: "Don't Say Lazy", ArtistName: "Artist", Duration: 180, PlainLyrics: "plain lyrics"})
	}
	tracks[0].SyncedLyrics = "[00:01.00]first line"
	tracks[2].SyncedLyrics = "[00:02.00]second line"
	results := rankLRCLibSearchResults("Don't Say Lazy", tracks, 7)
	if len(results) != 7 {
		t.Fatalf("got %d results, want 7", len(results))
	}
	if !results[0].HasSynced || !results[1].HasSynced {
		t.Fatal("synced lyrics should rank before plain lyrics")
	}
	for i := 2; i < len(results); i++ {
		if results[i].HasSynced {
			t.Fatalf("synced result unexpectedly ranked after plain at index %d", i)
		}
	}
}

func TestRankLRCLibSearchResults_SkipsEntriesWithoutLyrics(t *testing.T) {
	results := rankLRCLibSearchResults("Track", []lrclibTrack{
		{ID: 1, TrackName: "Track"},
		{ID: 2, TrackName: "Track", PlainLyrics: "  "},
		{ID: 3, TrackName: "Track", PlainLyrics: "line"},
	}, 7)
	if len(results) != 1 || results[0].ID != 3 {
		t.Fatalf("results = %#v, want only result 3", results)
	}
}
