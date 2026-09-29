package ui

import "testing"

func TestNormalizeVoiceIntent(t *testing.T) {
	cases := map[string]string{
		"":          "",
		" play ":    "PLAY",
		"pause":     "PAUSE",
		"Next":      "NEXT",
		"previous ": "PREVIOUS",
	}

	for input, want := range cases {
		if got := NormalizeVoiceIntent(input); got != want {
			t.Fatalf("NormalizeVoiceIntent(%q) = %q, want %q", input, got, want)
		}
	}
}
