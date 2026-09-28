package ui

import (
	"math"
	"testing"
)

func pulseHistory(period, frames int) []float64 {
	history := make([]float64, frames)
	for i := 0; i < frames; i++ {
		if i%period == 0 {
			history[i] = 1
		}
	}
	return history
}

func TestEstimateTempoCorrectsDoubleTime(t *testing.T) {
	bpm, ok := estimateTempo(pulseHistory(12, 120), 100)
	if !ok {
		t.Fatal("expected tempo estimate")
	}

	if math.Abs(bpm-75) > 0.01 {
		t.Fatalf("expected 75 BPM after half-time correction, got %.2f", bpm)
	}
}

func TestEstimateTempoKeeps120BPM(t *testing.T) {
	bpm, ok := estimateTempo(pulseHistory(15, 120), 100)
	if !ok {
		t.Fatal("expected tempo estimate")
	}

	if math.Abs(bpm-120) > 0.01 {
		t.Fatalf("expected 120 BPM, got %.2f", bpm)
	}
}

func TestEstimateTempoKeeps180BPM(t *testing.T) {
	bpm, ok := estimateTempo(pulseHistory(10, 120), 100)
	if !ok {
		t.Fatal("expected tempo estimate")
	}

	if math.Abs(bpm-180) > 0.01 {
		t.Fatalf("expected 180 BPM, got %.2f", bpm)
	}
}

func TestEstimateTempoNeedsEnoughHistory(t *testing.T) {
	bpm, ok := estimateTempo(pulseHistory(12, 30), 100)
	if ok {
		t.Fatal("expected no tempo estimate for short history")
	}

	if bpm != 0 {
		t.Fatalf("expected zero BPM, got %.2f", bpm)
	}
}
