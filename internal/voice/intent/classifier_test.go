package intent

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestWordFeatures(t *testing.T) {
	got := wordFeatures("Phát bài Numb")
	want := []string{"phát", "bài", "numb", "phát bài", "bài numb"}

	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCharWBFeatures(t *testing.T) {
	got := charWBFeatures("abc")
	want := []string{" ab", "abc", "bc ", " abc", "abc ", " abc "}

	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestPredict(t *testing.T) {
	model := &Model{
		SchemaVersion: 1,
		Classes:       []string{"NEXT", "PLAY"},
		Intercept:     []float64{0, 0},
		Coef:          [][]float64{{-1}, {1}},
		Word: FeatureModel{
			Vocabulary: map[string]int{"play": 0},
			IDF:        []float64{1},
			MinN:       1,
			MaxN:       1,
			Sublinear:  true,
		},
		CharWB: FeatureModel{
			Vocabulary: map[string]int{},
			IDF:        []float64{},
			MinN:       3,
			MaxN:       5,
			Sublinear:  true,
		},
		WordWeight:   1,
		CharWBWeight: 0.7,
	}

	if err := model.Validate(); err != nil {
		t.Fatal(err)
	}

	result := model.Predict("play")
	if result.Intent != "PLAY" {
		t.Fatalf("intent = %q, want PLAY", result.Intent)
	}
	if result.Confidence <= 0.5 || math.IsNaN(result.Confidence) {
		t.Fatalf("confidence = %v, want > 0.5", result.Confidence)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "model.json")
	data := []byte("{\"schema_version\":1,\"classes\":[\"PLAY\"],\"intercept\":[0],\"coef\":[[]],\"word\":{\"vocabulary\":{},\"idf\":[],\"min_n\":1,\"max_n\":2,\"sublinear_tf\":true},\"char_wb\":{\"vocabulary\":{},\"idf\":[],\"min_n\":3,\"max_n\":5,\"sublinear_tf\":true},\"word_weight\":1,\"char_wb_weight\":0.7}")

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	model, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(model.Classes) != 1 || model.Classes[0] != "PLAY" {
		t.Fatalf("classes = %#v", model.Classes)
	}
}
