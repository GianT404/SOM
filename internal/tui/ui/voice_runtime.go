package ui

import (
	"strings"

	"som/internal/tui/voice"
	voicequery "som/internal/voice"
	"som/internal/voice/intent"

	tea "charm.land/bubbletea/v2"
)

func (a *App) SetVoiceEvents(events <-chan voice.Event) {
	a.voiceEvents = events
}

func (a *App) SetVoiceStarter(starter func() (<-chan voice.Event, error)) {
	a.voiceStarter = starter
}

func (a *App) startVoiceCmd() tea.Cmd {
	if a.voiceStarter == nil {
		return nil
	}
	starter := a.voiceStarter
	return func() tea.Msg {
		events, err := starter()
		if err != nil {
			return VoiceStartErrorMsg{Err: err}
		}
		return VoiceStartedMsg{Events: events}
	}
}

func (a *App) SetVoiceIntentModel(model *intent.Model, minConfidence float64) {
	a.voiceIntentModel = model
	if minConfidence > 0 && minConfidence <= 1 {
		a.voiceMinConfidence = minConfidence
	}
}

func (a *App) waitVoiceEvent() tea.Cmd {
	if a.voiceEvents == nil {
		return nil
	}

	events := a.voiceEvents
	return func() tea.Msg {
		event, ok := <-events
		if !ok {
			return VoiceDisconnectedMsg{}
		}

		return VoiceEventMsg{
			Event:      event.Event,
			Transcript: event.Transcript,
			WakeAlias:  event.WakeAlias,
			Command:    event.Command,
			State:      event.State,
		}
	}
}

func (a *App) handleVoiceEventMsg(msg VoiceEventMsg) tea.Cmd {
	cmd := a.handleVoiceEvent(msg)
	if a.voiceEvents == nil {
		return cmd
	}

	wait := a.waitVoiceEvent()
	if cmd == nil {
		return wait
	}
	if wait == nil {
		return cmd
	}

	return tea.Batch(cmd, wait)
}

func (a *App) handleVoiceEvent(msg VoiceEventMsg) tea.Cmd {
	switch strings.ToLower(strings.TrimSpace(msg.Event)) {
	case "wake":
		alias := strings.TrimSpace(msg.WakeAlias)
		if alias == "" {
			alias = "yui"
		}
		a.setStatus(StatusOKStyle.Render("● Voice ready: " + alias))

	case "command":
		commandText := strings.TrimSpace(msg.Command)
		if commandText == "" {
			a.setStatus(StatusErrStyle.Render("X Voice: empty command"))
			return nil
		}
		if a.voiceIntentModel == nil {
			a.setStatus(StatusErrStyle.Render("X Voice: intent model is not configured"))
			return nil
		}

		prediction := a.voiceIntentModel.Predict(commandText)
		query := ""
		if NormalizeVoiceIntent(prediction.Intent) == "SEARCH" {
			query = voicequery.ExtractSearchQuery(commandText)
		}
		return func() tea.Msg {
			return VoiceCommandMsg{
				Command: VoiceCommand{
					Intent:     prediction.Intent,
					Query:      query,
					Transcript: commandText,
					Confidence: prediction.Confidence,
					WakeAlias:  msg.WakeAlias,
				},
			}
		}

	case "error":
		a.setStatus(StatusErrStyle.Render("X Voice: " + strings.TrimSpace(msg.State)))
	}

	return nil
}

func (a *App) handleVoiceDisconnected() {
	if a.player != nil {
		a.player.SetVoiceAEC(false)
		a.player.SetDucking(1.0)
	}
	a.voiceEvents = nil
	a.setStatus(StatusErrStyle.Render("X Voice: process stopped"))
}
