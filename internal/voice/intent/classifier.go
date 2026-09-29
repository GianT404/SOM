package intent

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"unicode"
)

const modelSchemaVersion = 1

// Model is a portable representation of the SOM Voice v2 text classifier.
// It mirrors the sklearn FeatureUnion(word TF-IDF + char_wb TF-IDF) followed by
// multinomial LogisticRegression.
type Model struct {
	SchemaVersion int          `json:"schema_version"`
	Classes       []string     `json:"classes"`
	Intercept     []float64    `json:"intercept"`
	Coef          [][]float64  `json:"coef"`
	Word          FeatureModel `json:"word"`
	CharWB        FeatureModel `json:"char_wb"`
	WordWeight    float64      `json:"word_weight"`
	CharWBWeight  float64      `json:"char_wb_weight"`
}

type FeatureModel struct {
	Vocabulary map[string]int `json:"vocabulary"`
	IDF        []float64      `json:"idf"`
	MinN       int            `json:"min_n"`
	MaxN       int            `json:"max_n"`
	Sublinear  bool           `json:"sublinear_tf"`
}

type Result struct {
	Intent     string
	Confidence float64
	Scores     map[string]float64
}

func Load(path string) (*Model, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read intent model: %w", err)
	}

	var model Model
	if err := json.Unmarshal(data, &model); err != nil {
		return nil, fmt.Errorf("decode intent model: %w", err)
	}
	if err := model.Validate(); err != nil {
		return nil, err
	}
	return &model, nil
}

func (m *Model) Validate() error {
	if m == nil {
		return errors.New("intent model is nil")
	}
	if m.SchemaVersion != modelSchemaVersion {
		return fmt.Errorf("unsupported intent model schema version: %d", m.SchemaVersion)
	}
	if len(m.Classes) == 0 {
		return errors.New("intent model has no classes")
	}
	if len(m.Intercept) != len(m.Classes) {
		return fmt.Errorf("intercept length %d does not match classes %d", len(m.Intercept), len(m.Classes))
	}
	if len(m.Coef) != len(m.Classes) {
		return fmt.Errorf("coef rows %d does not match classes %d", len(m.Coef), len(m.Classes))
	}

	wordFeatures := len(m.Word.Vocabulary)
	charFeatures := len(m.CharWB.Vocabulary)
	totalFeatures := wordFeatures + charFeatures

	for i, row := range m.Coef {
		if len(row) != totalFeatures {
			return fmt.Errorf("coef row %d has %d features, want %d", i, len(row), totalFeatures)
		}
	}
	if len(m.Word.IDF) != wordFeatures {
		return fmt.Errorf("word idf length %d does not match vocabulary %d", len(m.Word.IDF), wordFeatures)
	}
	if len(m.CharWB.IDF) != charFeatures {
		return fmt.Errorf("char_wb idf length %d does not match vocabulary %d", len(m.CharWB.IDF), charFeatures)
	}

	if m.WordWeight == 0 {
		m.WordWeight = 1
	}
	if m.CharWBWeight == 0 {
		m.CharWBWeight = 0.7
	}
	return nil
}

func (m *Model) Predict(text string) Result {
	if m == nil || len(m.Classes) == 0 {
		return Result{}
	}

	word := tfidfVector(text, m.Word, wordFeatures)
	char := tfidfVector(text, m.CharWB, charWBFeatures)

	wordN := len(word)
	features := make([]float64, wordN+len(char))
	for i, value := range word {
		features[i] = value * m.WordWeight
	}
	for i, value := range char {
		features[wordN+i] = value * m.CharWBWeight
	}

	logits := make([]float64, len(m.Classes))
	maxLogit := math.Inf(-1)
	for i := range m.Classes {
		score := m.Intercept[i]
		row := m.Coef[i]
		for j, value := range features {
			score += row[j] * value
		}
		logits[i] = score
		if i == 0 || score > maxLogit {
			maxLogit = score
		}
	}

	probs := make([]float64, len(logits))
	var sum float64
	for i, logit := range logits {
		value := math.Exp(logit - maxLogit)
		probs[i] = value
		sum += value
	}
	if sum == 0 || math.IsNaN(sum) {
		return Result{}
	}

	best := 0
	for i := range probs {
		probs[i] /= sum
		if probs[i] > probs[best] {
			best = i
		}
	}

	scores := make(map[string]float64, len(m.Classes))
	for i, class := range m.Classes {
		scores[class] = probs[i]
	}

	return Result{
		Intent:     m.Classes[best],
		Confidence: probs[best],
		Scores:     scores,
	}
}

func wordFeatures(text string) []string {
	words := make([]string, 0)
	var current []rune

	flush := func() {
		if len(current) >= 2 {
			words = append(words, string(current))
		}
		current = current[:0]
	}

	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			current = append(current, r)
			continue
		}
		flush()
	}
	flush()

	ngrams := make([]string, 0, len(words)*2)
	for _, word := range words {
		ngrams = append(ngrams, word)
	}
	for i := 0; i+1 < len(words); i++ {
		ngrams = append(ngrams, words[i]+" "+words[i+1])
	}
	return ngrams
}

func charWBFeatures(text string) []string {
	words := strings.FieldsFunc(strings.ToLower(text), unicode.IsSpace)
	ngrams := make([]string, 0)

	for _, token := range words {
		padded := []rune(" " + token + " ")
		for n := 3; n <= 5; n++ {
			if len(padded) < n {
				break
			}
			for offset := 0; offset+n <= len(padded); offset++ {
				ngrams = append(ngrams, string(padded[offset:offset+n]))
			}
		}
	}
	return ngrams
}

func tfidfVector(text string, feature FeatureModel, tokenizer func(string) []string) []float64 {
	vector := make([]float64, len(feature.Vocabulary))
	if len(vector) == 0 {
		return vector
	}

	counts := make(map[int]int)
	for _, token := range tokenizer(text) {
		index, ok := feature.Vocabulary[token]
		if !ok {
			continue
		}
		counts[index]++
	}

	var norm float64
	for index, count := range counts {
		tf := float64(count)
		if feature.Sublinear && count > 0 {
			tf = 1 + math.Log(tf)
		}
		value := tf * feature.IDF[index]
		vector[index] = value
		norm += value * value
	}

	if norm == 0 {
		return vector
	}

	norm = math.Sqrt(norm)
	for i, value := range vector {
		vector[i] = value / norm
	}
	return vector
}

func DebugTopScores(result Result, limit int) []struct {
	Intent string
	Score  float64
} {
	items := make([]struct {
		Intent string
		Score  float64
	}, 0, len(result.Scores))

	for intentName, score := range result.Scores {
		items = append(items, struct {
			Intent string
			Score  float64
		}{intentName, score})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Score > items[j].Score
	})

	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}
