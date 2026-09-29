package player

import "testing"

func TestPlayerDuckingPreservesUserVolume(t *testing.T) {
	p := &Player{
		volume:     1.0,
		duckFactor: 1.0,
	}

	p.SetDucking(0.25)
	if got := p.duckFactor; got != 0.25 {
		t.Fatalf("duck factor = %v, want 0.25", got)
	}
	if got := p.effectiveVolumeLocked(); got != 0.25 {
		t.Fatalf("effective volume = %v, want 0.25", got)
	}

	p.SetVolume(0.8)
	if got := p.volume; got != 0.8 {
		t.Fatalf("user volume = %v, want 0.8", got)
	}
	if got := p.effectiveVolumeLocked(); got != 0.2 {
		t.Fatalf("effective volume = %v, want 0.2", got)
	}

	p.SetDucking(1.0)
	if got := p.effectiveVolumeLocked(); got != 0.8 {
		t.Fatalf("restored effective volume = %v, want 0.8", got)
	}
}
