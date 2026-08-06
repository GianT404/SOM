package player

import (
	"sync"
	"testing"
)

func newTestPlayer() *Player {
	return &Player{state: Stopped}
}

func TestPlayer_ConcurrentPlayStopSeek_NoRaceNoDeadlock(t *testing.T) {
	p := newTestPlayer()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			switch i % 3 {
			case 0:
				_ = p.Play("fake.opus")
			case 1:
				p.Stop()
			case 2:
				p.SeekBy(5)
			}
		}(i)
	}
	wg.Wait()

	switch p.State() {
	case Stopped, Playing, Paused:
	default:
		t.Fatalf("state không hợp lệ : %v", p.State())
	}
}

func TestPlayer_Stop_IdempotentWhenAlreadyStopped(t *testing.T) {
	p := newTestPlayer()

	p.Stop()
	p.Stop()
	p.Stop()

	if p.State() != Stopped {
		t.Fatalf("state không hợp lệ : %v", p.State())
	}
}

func TestPlayer_PlayFrom_CallsStopLockedBeforeOtoCtxCheck(t *testing.T) {
	p := newTestPlayer()
	p.state = Playing

	err := p.Play("fake.opus")
	if err == nil {
		t.Fatal("otoCtx nil thi playFrom phai tra ve loi")
	}
	if p.State() != Stopped {
		t.Fatalf("state không hợp lệ : %v", p.State())
	}
}
