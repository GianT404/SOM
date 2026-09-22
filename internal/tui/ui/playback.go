package ui

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"som/internal/domain"
	"som/internal/storage"
	"som/internal/tui/player"

	tea "charm.land/bubbletea/v2"
)

// PlaybackManager quản dữ liệu hàng đợi, lịch sử và thuật toán chọn bài.
type PlaybackManager struct {
	Player      *player.Player
	Provider    domain.MusicProvider
	NowPlay     *domain.Track
	NextPlay    *domain.Track
	SongStarted bool
	PlayerGen   uint64

	ResolveCancel context.CancelFunc
	Playlist      []domain.Track
	CurrentIdx    int
	Random        bool
	ShuffleHist   []int
	History       []domain.Track
	Queue         []domain.Track
	Store         *storage.DB
}

func NewPlaybackManager() *PlaybackManager {
	return &PlaybackManager{}
}

func (pm *PlaybackManager) SetDependencies(p *player.Player, prov domain.MusicProvider, store *storage.DB) {
	pm.Player = p
	pm.Provider = prov
	pm.Store = store
}

// CancelResolve hủy các luồng tải stream cũ an toàn
func (pm *PlaybackManager) CancelResolve() {
	if pm.ResolveCancel != nil {
		pm.ResolveCancel()
		pm.ResolveCancel = nil
	}
}

// NextTrack trả về (Track, Vị trí trong playlist, Đánh dấu lấy từ Queue)
func (pm *PlaybackManager) NextTrack() (*domain.Track, int, bool) {
	if len(pm.Queue) > 0 {
		t := pm.Queue[0]
		pm.Queue = pm.Queue[1:]
		return &t, -1, true
	}
	if len(pm.Playlist) == 0 {
		return nil, -1, false
	}
	next := pm.CurrentIdx + 1
	if pm.Random {
		next = pm.pickAntiClumpIndex()
	}
	if next >= len(pm.Playlist) {
		return nil, -1, false
	}
	return &pm.Playlist[next], next, false
}

// PrevTrack trả về (Track, Vị trí)
func (pm *PlaybackManager) PrevTrack() (*domain.Track, int) {
	if len(pm.Playlist) == 0 {
		return nil, -1
	}
	if pm.Random && len(pm.History) > 0 {
		prev := pm.History[len(pm.History)-1]
		pm.History = pm.History[:len(pm.History)-1]
		for i, tr := range pm.Playlist {
			if tr.ID == prev.ID {
				return &tr, i
			}
		}
		return &prev, -1
	}
	prevIdx := pm.CurrentIdx - 1
	if prevIdx < 0 {
		return nil, -1
	}
	return &pm.Playlist[prevIdx], prevIdx
}

// RecordHistory ghi nhận lịch sử trước khi chuyển bài
func (pm *PlaybackManager) RecordHistory() {
	if pm.NowPlay != nil && pm.Random {
		pm.History = append(pm.History, *pm.NowPlay)
	}
}

func (pm *PlaybackManager) pickAntiClumpIndex() int {
	n := len(pm.Playlist)
	if n <= 1 {
		return 0
	}
	histCap := n / 2
	if histCap > 8 {
		histCap = 8
	}

	recent := make(map[int]bool, len(pm.ShuffleHist))
	start := len(pm.ShuffleHist) - histCap
	if start < 0 {
		start = 0
	}
	for _, idx := range pm.ShuffleHist[start:] {
		recent[idx] = true
	}

	curArtist := ""
	if pm.CurrentIdx >= 0 && pm.CurrentIdx < n {
		curArtist = pm.Playlist[pm.CurrentIdx].Artist
	}

	var freshDiffArtist, freshSameArtist, usedDiffArtist []int
	for i, t := range pm.Playlist {
		if i == pm.CurrentIdx {
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
		for i := range pm.Playlist {
			if i != pm.CurrentIdx {
				pool = append(pool, i)
			}
		}
	}

	picked := pool[rand.Intn(len(pool))]
	pm.ShuffleHist = append(pm.ShuffleHist, picked)
	if len(pm.ShuffleHist) > histCap*2 {
		pm.ShuffleHist = pm.ShuffleHist[len(pm.ShuffleHist)-histCap:]
	}

	return picked
}

func (pm *PlaybackManager) Update(msg tea.Msg) (*PlaybackManager, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case SetPlaylistMsg:
		pm.Playlist = msg.Tracks
		pm.ShuffleHist = nil
		pm.History = nil

	case EnqueueTrackMsg:
		pm.Queue = append(pm.Queue, msg.Track)
		cmds = append(cmds, func() tea.Msg { return QueueChangedMsg{Queue: pm.Queue} })

	case ToggleRandomMsg:
		pm.Random = !pm.Random
		pm.ShuffleHist = nil
		pm.History = nil
		if pm.NowPlay != nil {
			// Báo cho UI biết để cập nhật biểu tượng Random [r]
			cmds = append(cmds, func() tea.Msg {
				return TrackChangedMsg{
					Track:       *pm.NowPlay,
					IsLocal:     strings.HasPrefix(pm.NowPlay.ID, "local:"),
					Gen:         pm.PlayerGen,
					PlaylistPos: pm.CurrentIdx,
					PlaylistLen: len(pm.Playlist),
					IsRandom:    pm.Random,
				}
			})
		}

	case PlayQueueMsg:
		if msg.Index >= 0 && msg.Index < len(pm.Queue) {
			t := pm.Queue[msg.Index]
			pm.Queue = append(pm.Queue[:msg.Index], pm.Queue[msg.Index+1:]...)
			cmds = append(cmds, pm.playTrackCmd(-1, t))
			cmds = append(cmds, func() tea.Msg { return QueueChangedMsg{Queue: pm.Queue} })
		}

	case RemoveFromQueueMsg:
		if msg.Index >= 0 && msg.Index < len(pm.Queue) {
			pm.Queue = append(pm.Queue[:msg.Index], pm.Queue[msg.Index+1:]...)
			cmds = append(cmds, func() tea.Msg { return QueueChangedMsg{Queue: pm.Queue} })
		}

	case DeleteDoneMsg:
		if pm.NowPlay != nil && pm.NowPlay.ID == "local:"+msg.Path {
			pm.Player.Stop()
			pm.NowPlay = nil
			pm.SongStarted = false
			pm.NextPlay = nil
			cmds = append(cmds, func() tea.Msg { return TrackChangedMsg{Track: domain.Track{}, IsLocal: true} })
		}
		var newPl []domain.Track
		for _, t := range pm.Playlist {
			if t.ID != "local:"+msg.Path {
				newPl = append(newPl, t)
			}
		}
		pm.Playlist = newPl
		var newQueue []domain.Track
		for _, t := range pm.Queue {
			if t.ID != "local:"+msg.Path {
				newQueue = append(newQueue, t)
			}
		}
		pm.Queue = newQueue
		cmds = append(cmds, func() tea.Msg { return QueueChangedMsg{Queue: pm.Queue} })

	case RenameDoneMsg:
		if msg.Err != nil {
			break
		}
		for i := range pm.Playlist {
			if pm.Playlist[i].ID == "local:"+msg.OldPath {
				pm.Playlist[i].ID = "local:" + msg.NewPath
				pm.Playlist[i].Title = msg.NewTitle
			}
		}
		for i := range pm.Queue {
			if pm.Queue[i].ID == "local:"+msg.OldPath {
				pm.Queue[i].ID = "local:" + msg.NewPath
				pm.Queue[i].Title = msg.NewTitle
			}
		}
		if pm.NowPlay != nil && pm.NowPlay.ID == "local:"+msg.OldPath {
			pm.NowPlay.ID = "local:" + msg.NewPath
			pm.NowPlay.Title = msg.NewTitle
			cmds = append(cmds, func() tea.Msg {
				return TrackChangedMsg{Track: *pm.NowPlay, IsLocal: true, Gen: pm.PlayerGen, PlaylistPos: pm.CurrentIdx, PlaylistLen: len(pm.Playlist), IsRandom: pm.Random}
			})
		}
		cmds = append(cmds, func() tea.Msg { return QueueChangedMsg{Queue: pm.Queue} })

	case PlayNextMsg:
		t, idx, isQueue := pm.NextTrack()
		if t == nil {
			return pm, nil
		}
		if isQueue {
			cmds = append(cmds, pm.playTrackCmd(-1, *t))
		} else {
			cmds = append(cmds, pm.playTrackCmd(idx, *t))
		}

	case PlayPrevMsg:
		t, idx := pm.PrevTrack()
		if t != nil {
			cmds = append(cmds, pm.playTrackCmd(idx, *t))
		}

	case PlayTrackAtMsg:
		cmds = append(cmds, pm.playTrackCmd(msg.Index, msg.Track))

	case PlaybackTickMsg:
		if !pm.SongStarted || pm.NowPlay == nil {
			return pm, nil
		}

		// Xử lý khi bài hát KẾT THÚC
		if pm.Player.State() == player.Stopped {
			playErr := pm.Player.PlaybackError()
			pm.NowPlay = nil

			if playErr != nil {
				cmds = append(cmds, func() tea.Msg { return PlaybackErrorMsg{Err: playErr} })
				return pm, tea.Batch(cmds...)
			}

			// Nếu đã Pre-decode sẵn từ buffer (Gapless)
			if pm.Player.PlayFromBuffer() {
				if pm.NextPlay != nil {
					t := *pm.NextPlay
					pm.NextPlay = nil
					pm.NowPlay = &t
					// Dọn dẹp hàng đợi / playlist
					if len(pm.Queue) > 0 && pm.Queue[0].ID == t.ID {
						pm.Queue = pm.Queue[1:]
					} else {
						for i, tr := range pm.Playlist {
							if tr.ID == t.ID {
								pm.CurrentIdx = i
								break
							}
						}
					}

					pm.PlayerGen = pm.Player.Generation()
					pm.SongStarted = true

					//  UI cập nhật
					cmds = append(cmds, func() tea.Msg {
						return TrackChangedMsg{
							Track:       t,
							IsLocal:     true,
							Gen:         pm.PlayerGen,
							PlaylistPos: pm.CurrentIdx,
							PlaylistLen: len(pm.Playlist),
							IsRandom:    pm.Random,
						}
					})
				}
			} else {
				cmds = append(cmds, func() tea.Msg { return PlayNextMsg{} })
			}
		} else if pm.Player.State() == player.Playing {
			pos := pm.Player.Position()
			dur := time.Duration(pm.NowPlay.Duration) * time.Second
			if remaining := dur - pos; remaining > 0 && remaining <= 3*time.Second {
				pm.triggerPreDecodeNext()
			}
		}
	case TogglePauseMsg:
		pm.Player.TogglePause()
		cmds = append(cmds, func() tea.Msg { return PlaybackStateChangedMsg{State: int(pm.Player.State())} })
	}

	return pm, tea.Batch(cmds...)
}

func (pm *PlaybackManager) playTrackCmd(idx int, t domain.Track) tea.Cmd {
	pm.CancelResolve()

	pm.NowPlay = &t
	pm.CurrentIdx = idx
	pm.SongStarted = false
	pm.NextPlay = nil

	if pm.Random {
		pm.History = append(pm.History, t)
		pm.RecordHistory()
	}

	trackChangedMsg := TrackChangedMsg{
		Track:       t,
		IsLocal:     strings.HasPrefix(t.ID, "local:"),
		Gen:         pm.PlayerGen,
		PlaylistPos: pm.CurrentIdx,
		PlaylistLen: len(pm.Playlist),
		IsRandom:    pm.Random,
	}

	if trackChangedMsg.IsLocal {
		path := strings.TrimPrefix(t.ID, "local:")
		if err := pm.Player.Play(path); err != nil {
			return func() tea.Msg { return PlaybackErrorMsg{Err: err} }
		}
		pm.PlayerGen = pm.Player.Generation()
		pm.SongStarted = true

		trackChangedMsg.Gen = pm.PlayerGen
		return func() tea.Msg { return trackChangedMsg }
	}

	// Logic Stream
	ctx, cancel := context.WithCancel(context.Background())
	pm.ResolveCancel = cancel
	gen := pm.Player.Generation()
	pm.PlayerGen = gen

	trackChangedMsg.Gen = gen

	// Luồng ngầm: Đi lấy stream URL
	resolveStreamCmd := func() tea.Msg {
		streamInfo, err := pm.Provider.ResolveStream(ctx, t.ID)
		if ctx.Err() != nil || gen != pm.Player.Generation() {
			return nil
		}
		if err != nil || streamInfo == nil || streamInfo.URL == "" {
			return PlaybackErrorMsg{Err: fmt.Errorf("lỗi lấy link: %v", err)}
		}

		if err := pm.Player.PlayWithHeaders(streamInfo.URL, streamInfo.Headers); err != nil {
			return PlaybackErrorMsg{Err: err}
		}

		lr, lyricsErr := getCachedLyrics(pm.Provider, pm.Store, t.ID, t.Title, t.Artist, t.Duration)
		return StreamResolvedMsg{
			Lyrics:    lr,
			LyricsErr: lyricsErr,
			Gen:       gen,
		}
	}

	// Batch: Ném TrackChangedMsg NGAY LẬP TỨC để UI đổi chữ,
	// đồng thời chạy resolveStreamCmd ở background.
	return tea.Batch(
		func() tea.Msg { return trackChangedMsg },
		resolveStreamCmd,
	)
}

func (pm *PlaybackManager) triggerPreDecodeNext() {
	if pm.NextPlay != nil {
		return
	}
	var next domain.Track
	if len(pm.Queue) > 0 {
		next = pm.Queue[0]
	} else if len(pm.Playlist) > 0 {
		idx := pm.CurrentIdx + 1
		if pm.Random {
			idx = pm.pickAntiClumpIndex()
		}
		if idx >= len(pm.Playlist) {
			return
		}
		next = pm.Playlist[idx]
	} else {
		return
	}

	if !strings.HasPrefix(next.ID, "local:") {
		return
	}
	path := strings.TrimPrefix(next.ID, "local:")
	pm.NextPlay = &next
	pm.Player.PreDecodeNext(path, nil)
}
