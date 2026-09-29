package ui

import (
	"testing"

	"som/internal/tui/player"
	"som/internal/voice/intent"
)



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

func TestHandleVoiceEventCommand(t *testing.T) {
	model := &intent.Model{
		SchemaVersion: 1,
		Classes:       []string{"PLAY"},
		Intercept:     []float64{1},
		Coef:          [][]float64{{}},
		Word:          intent.FeatureModel{Vocabulary: map[string]int{}, IDF: []float64{}},
		CharWB:        intent.FeatureModel{Vocabulary: map[string]int{}, IDF: []float64{}},
	}
	a := &App{
		voiceIntentModel:   model,
		voiceMinConfidence: DefaultVoiceMinConfidence,
	}

	cmd := a.handleVoiceEvent(VoiceEventMsg{
		Event:     "command",
		WakeAlias: "yui",
		Command:   "phát nhạc",
	})
	if cmd == nil {
		t.Fatal("expected command msg")
	}

	msg := cmd()
	voiceMsg, ok := msg.(VoiceCommandMsg)
	if !ok {
		t.Fatalf("message type = %T, want VoiceCommandMsg", msg)
	}
	if voiceMsg.Command.Intent != "PLAY" {
		t.Fatalf("intent = %q, want PLAY", voiceMsg.Command.Intent)
	}
	if voiceMsg.Command.Transcript != "phát nhạc" {
		t.Fatalf("transcript = %q", voiceMsg.Command.Transcript)
	}
	if voiceMsg.Command.WakeAlias != "yui" {
		t.Fatalf("wake alias = %q", voiceMsg.Command.WakeAlias)
	}
}
func TestHandleVoiceCommandPlaySelectedDownload(t *testing.T) {
	a := &App{
		player: &player.Player{},
		playback: &PlaybackManager{},
		left: LeftPanel{
			activeTab: SideDownloads,
			locals: []LocalFile{{
				Name: "track.opus",
				Path: "/music/track.opus",
			}},
		},
	}

	cmd := a.handleVoiceCommand(VoiceCommandMsg{
		Command: VoiceCommand{
			Intent:     "PLAY",
			Transcript: "phát nhạc",
			Confidence: 1,
		},
	})
	if cmd == nil {
		t.Fatal("expected play command")
	}

	msg := cmd()
	play, ok := msg.(PlayLocalMsg)
	if !ok {
		t.Fatalf("message type = %T, want PlayLocalMsg", msg)
	}
	if play.Path != "/music/track.opus" || play.Title != "track.opus" {
		t.Fatalf("play = %#v", play)
	}
}

