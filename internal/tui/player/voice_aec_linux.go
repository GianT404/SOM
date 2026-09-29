package player

import (
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	voiceAECSinkFL = "echo-cancel-sink:playback_FL"
	voiceAECSinkFR = "echo-cancel-sink:playback_FR"
	voicePCMFL     = "alsa_playback.som:output_FL"
	voicePCMFR     = "alsa_playback.som:output_FR"
)

func routeVoicePlaybackThroughAEC() {
	if runtime.GOOS != "linux" {
		return
	}
	if _, err := exec.LookPath("pw-link"); err != nil {
		return
	}

	for i := 0; i < 20; i++ {
		if routeVoicePlaybackThroughAECOnce() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func routeVoicePlaybackThroughAECOnce() bool {
	links, err := exec.Command("pw-link", "-l").Output()
	if err != nil {
		return false
	}

	graph := string(links)
	if !strings.Contains(graph, voiceAECSinkFL) || !strings.Contains(graph, voiceAECSinkFR) {
		return false
	}

	removeDirectPhysicalLinks(graph, voicePCMFL, "playback_FL")
	removeDirectPhysicalLinks(graph, voicePCMFR, "playback_FR")

	_ = exec.Command("pw-link", voicePCMFL, voiceAECSinkFL).Run()
	_ = exec.Command("pw-link", voicePCMFR, voiceAECSinkFR).Run()

	return true
}

func removeDirectPhysicalLinks(graph, source, channel string) {
	lines := strings.Split(graph, "\n")
	for i := range lines {
		if strings.TrimSpace(lines[i]) != source {
			continue
		}

		for j := i + 1; j < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[j]), "|-> "); j++ {
			target := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[j]), "|->"))
			if strings.HasPrefix(target, "alsa_output.") && strings.HasSuffix(target, ":"+channel) {
				_ = exec.Command("pw-link", "-d", source, target).Run()
			}
		}
	}
}
