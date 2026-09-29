package ui

import "strings"

// VoiceCommand là command cấp application được tạo từ voice/session layer.
// Nó không phụ thuộc STT engine hay intent classifier cụ thể.
type VoiceCommand struct {
	Intent     string
	Query      string
	Transcript string
	Confidence float64
	WakeAlias  string
}

// VoiceCommandMsg đưa một voice command vào event pipeline của TUI.
type VoiceCommandMsg struct {
	Command VoiceCommand
}

// NormalizeVoiceIntent chuẩn hóa intent trước khi dispatch.
func NormalizeVoiceIntent(intent string) string {
	return strings.ToUpper(strings.TrimSpace(intent))
}

const DefaultVoiceMinConfidence = 0.30
