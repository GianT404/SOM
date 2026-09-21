package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	"som/internal/domain"

	tea "charm.land/bubbletea/v2"
)

func (a *App) cancelResolve() {
	if a.playback.ResolveCancel != nil {
		a.playback.ResolveCancel()
		a.playback.ResolveCancel = nil
	}
}

func (a *App) playTrackAt(idx int, t domain.Track) tea.Cmd {
	if a.playback.NowPlay != nil && a.playback.Random {
		a.playback.History = append(a.playback.History, *a.playback.NowPlay)
	}
	a.playback.RecordHistory()
	a.playback.CurrentIdx = idx
	a.playback.NowPlay = &t
	a.playback.SongStarted = false
	a.playback.NextPlay = nil
	a.syncPlaylistState()
	a.left.FocusTrack(t.ID)
	if strings.HasPrefix(t.ID, "local:") {
		a.playback.CancelResolve()
		a.left.loadingStream = false
		path := strings.TrimPrefix(t.ID, "local:")
		if err := a.player.Play(path); err != nil {
			a.setStatus(StatusErrStyle.Render("X " + err.Error()))
			return a.playNext()
		}
		a.playback.PlayerGen = a.player.Generation()
		a.playback.SongStarted = true
		a.right.SetTrack(&t)
		a.setStatus(StatusOKStyle.Render(">  " + t.Title))
		if a.avrcp != nil {
			a.avrcp.UpdateMetadata(t.ID, t.Title, t.Artist, "", t.Thumbnail, int64(t.Duration)*1_000_000)
			a.avrcp.UpdatePlaybackStatus("Playing")
		}
		if a.left.plStore != nil {
			if lyricsJSON, err := a.left.plStore.GetLocalFileLyrics(path); err == nil && lyricsJSON != "" {
				var lr domain.LyricsResp
				if json.Unmarshal([]byte(lyricsJSON), &lr) == nil {
					a.right.SetLyrics(lr)
					return nil
				}
			}
		}
		a.right.SetLyrics(domain.LyricsResp{Plain: "(No lyrics available)"})
		return nil
	}

	a.playback.CancelResolve()
	ctx, cancel := context.WithCancel(context.Background())
	a.playback.ResolveCancel = cancel
	gen := a.player.Generation()
	a.playback.PlayerGen = gen
	return func() tea.Msg {
		streamInfo, err := a.provider.ResolveStream(ctx, t.ID)
		if ctx.Err() != nil || gen != a.player.Generation() {
			return nil
		}
		if err != nil || streamInfo == nil || streamInfo.URL == "" {
			return StreamStartedMsg{Err: fmt.Errorf("lỗi lấy link CDN: %v", err)}
		}
		if err := a.player.PlayWithHeaders(streamInfo.URL, streamInfo.Headers); err != nil {
			return StreamStartedMsg{Err: err}
		}
		lr, lyricsErr := getCachedLyrics(a.provider, t.ID, t.Title, t.Artist, t.Duration)
		return StreamStartedMsg{
			Track:     t,
			Lyrics:    lr,
			LyricsErr: lyricsErr,
			Gen:       gen,
		}
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

func (a *App) playNext() tea.Cmd {
	t, idx, isQueue := a.playback.NextTrack()
	if t == nil {
		return nil
	}
	if isQueue {
		if a.left.qCursor >= len(a.playback.Queue) {
			a.left.qCursor = maxInt(len(a.playback.Queue)-1, 0)
		}
		a.focusTrackAndSwitchTab(*t)
		a.setStatus(StatusOKStyle.Render(fmt.Sprintf("> Playing from queue: %s", t.Title)))
	}
	return a.playTrackAt(idx, *t)
}

func (a *App) playPrev() tea.Cmd {
	t, idx := a.playback.PrevTrack()
	if t == nil {
		return nil
	}
	return a.playTrackAt(idx, *t)
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

// loadLyricsForTrack reads lyrics from SQLite for local files or sets placeholder.
func (a *App) loadLyricsForTrack(t domain.Track) {
	if strings.HasPrefix(t.ID, "local:") {
		path := strings.TrimPrefix(t.ID, "local:")
		if a.left.plStore != nil {
			if lyricsJSON, err := a.left.plStore.GetLocalFileLyrics(path); err == nil && lyricsJSON != "" {
				var lr domain.LyricsResp
				if json.Unmarshal([]byte(lyricsJSON), &lr) == nil {
					a.right.SetLyrics(lr)
					return
				}
			}
		}
	}
	a.right.SetLyrics(domain.LyricsResp{Plain: "(No lyrics available)"})
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
