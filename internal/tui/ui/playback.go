package ui

import (
	"context"
	"math/rand"

	"som/internal/domain"
)

// PlaybackManager quản dữ liệu hàng đợi, lịch sử và thuật toán chọn bài.
type PlaybackManager struct {
	NowPlay       *domain.Track
	NextPlay      *domain.Track
	SongStarted   bool
	PlayerGen     uint64
	ResolveCancel context.CancelFunc

	Playlist    []domain.Track
	CurrentIdx  int
	Random      bool
	ShuffleHist []int
	History     []domain.Track
	Queue       []domain.Track
}

func NewPlaybackManager() *PlaybackManager {
	return &PlaybackManager{}
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
