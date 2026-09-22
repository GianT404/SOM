package ui

import (
	"math/rand"
	"strings"

	"som/internal/domain"
)

func (a *App) cancelResolve() {
	if a.playback.ResolveCancel != nil {
		a.playback.ResolveCancel()
		a.playback.ResolveCancel = nil
	}
}

func (a *App) focusTrackAndSwitchTab(t domain.Track) {
	if strings.HasPrefix(t.ID, "local:") {
		a.activeContext = SideDownloads
		a.sidebarActive = SideDownloads
		a.left.activeTab = SideDownloads
	} else {
		a.activeContext = SideSearch
		a.sidebarActive = SideSearch
		a.left.activeTab = SideSearch
	}
	a.left.FocusTrack(t.ID)
}

// triggerPreDecodeNext pre-decodes the next track for gapless playback.
// Only works for local files (no headers needed).
func (a *App) triggerPreDecodeNext() {
	if a.playback.NextPlay != nil {
		return
	}

	var next domain.Track
	if len(a.playback.Queue) > 0 {
		next = a.playback.Queue[0]
	} else if len(a.playback.Playlist) > 0 {
		idx := a.playback.CurrentIdx + 1
		if a.playback.Random {
			idx = a.playback.pickAntiClumpIndex()
		}
		if idx >= len(a.playback.Playlist) {
			return
		}
		next = a.playback.Playlist[idx]
	} else {
		return
	}

	if !strings.HasPrefix(next.ID, "local:") {
		return
	}
	path := strings.TrimPrefix(next.ID, "local:")

	a.playback.NextPlay = &next
	a.player.PreDecodeNext(path, nil)
}

func (a *App) pickAntiClumpIndex() int {
	n := len(a.playback.Playlist)
	if n <= 1 {
		return 0
	}
	histCap := n / 2
	if histCap > 8 {
		histCap = 8
	}

	recent := make(map[int]bool, len(a.playback.ShuffleHist))
	start := len(a.playback.ShuffleHist) - histCap
	if start < 0 {
		start = 0
	}
	for _, idx := range a.playback.ShuffleHist[start:] {
		recent[idx] = true
	}

	curArtist := ""
	if a.playback.CurrentIdx >= 0 && a.playback.CurrentIdx < n {
		curArtist = a.playback.Playlist[a.playback.CurrentIdx].Artist
	}

	var freshDiffArtist, freshSameArtist, usedDiffArtist []int
	for i, t := range a.playback.Playlist {
		if i == a.playback.CurrentIdx {
			continue
		}
		diffArtist := curArtist == "" || t.Artist != curArtist
		if recent[i] {
			if diffArtist {
				usedDiffArtist = append(usedDiffArtist, i)
			}
			continue
		}
		if diffArtist {
			freshDiffArtist = append(freshDiffArtist, i)
		} else {
			freshSameArtist = append(freshSameArtist, i)
		}
	}

	pool := freshDiffArtist
	if len(pool) == 0 {
		pool = freshSameArtist
	}
	if len(pool) == 0 {
		pool = usedDiffArtist
	}
	if len(pool) == 0 {
		for i := range a.playback.Playlist {
			if i != a.playback.CurrentIdx {
				pool = append(pool, i)
			}
		}
	}

	picked := pool[rand.Intn(len(pool))]

	a.playback.ShuffleHist = append(a.playback.ShuffleHist, picked)
	if len(a.playback.ShuffleHist) > histCap*2 {
		a.playback.ShuffleHist = a.playback.ShuffleHist[len(a.playback.ShuffleHist)-histCap:]
	}

	return picked
}

func (a *App) syncPlaylistState() {
	if a.playback.Playlist != nil {
		a.right.SetPlaylistState(a.playback.CurrentIdx, len(a.playback.Playlist), a.playback.Random)
	}
}
