package ui

import (
	"testing"

	"som/internal/voice/intent"

	tea "charm.land/bubbletea/v2"
)

func TestHandleVoiceEventClassifiesCommand(t *testing.T) {
	app := &App{
		voiceIntentModel: &intent.Model{
			SchemaVersion: 1,
			Classes:       []string{"PLAY"},
			Intercept:     []float64{0},
			Coef:          [][]float64{{}},
			Word: intent.FeatureModel{
				Vocabulary: map[string]int{},
				IDF:        []float64{},
			},
			CharWB: intent.FeatureModel{
				Vocabulary: map[string]int{},
				IDF:        []float64{},
			},
		},
		voiceMinConfidence: DefaultVoiceMinConfidence,
	}

	cmd := app.handleVoiceEvent(VoiceEventMsg{
		Event:     "command",
		WakeAlias: "yui",
		Command:   "phát nhạc",
	})

	if cmd == nil {
		t.Fatal("handleVoiceEvent returned nil command")
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
		t.Fatalf("transcript = %q, want original command", voiceMsg.Command.Transcript)
	}
	if voiceMsg.Command.WakeAlias != "yui" {
		t.Fatalf("wake alias = %q, want yui", voiceMsg.Command.WakeAlias)
	}
}

func TestHandleVoiceCommandDispatchesNext(t *testing.T) {
	app := &App{voiceMinConfidence: DefaultVoiceMinConfidence}

	cmd := app.handleVoiceCommand(VoiceCommandMsg{
		Command: VoiceCommand{
			Intent:     " next ",
			Transcript: "bài tiếp theo",
			Confidence: 1,
		},
	})
	if cmd == nil {
		t.Fatal("handleVoiceCommand returned nil")
	}

	msg := cmd()
	if _, ok := msg.(PlayNextMsg); !ok {
		t.Fatalf("message type = %T, want PlayNextMsg", msg)
	}

	var _ tea.Msg = msg
}
