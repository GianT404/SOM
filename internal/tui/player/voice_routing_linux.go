package player

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// PrepareVoicePlaybackRouting configures PipeWire ALSA before Oto creates its
// playback context. PIPEWIRE_NODE is the native PipeWire ALSA mechanism for
// selecting a specific playback node.
func PrepareVoicePlaybackRouting() {
	if runtime.GOOS != "linux" {
		return
	}
	if _, err := exec.LookPath("pw-cli"); err != nil {
		return
	}

	graph, err := exec.Command("pw-cli", "ls", "Node").Output()
	if err != nil {
		return
	}

	if !strings.Contains(string(graph), `node.name = "echo-cancel-sink"`) {
		return
	}

	// Respect an explicit user setting. Voice AEC is opt-in through --voice,
	// but should not silently override a caller that deliberately selected
	// another PipeWire node for SOM.
	if value := strings.TrimSpace(os.Getenv("PIPEWIRE_NODE")); value != "" {
		return
	}

	_ = os.Setenv("PIPEWIRE_NODE", "echo-cancel-sink")
}
