package ui

import (
	"fmt"
	"math"
	"os"
	"strings"

	"som/internal/domain"
)

func renderDashboard(hide bool, vol float64, speedIdx int, presetIdx int, nowPlay *domain.Track) string {
	if hide {
		return ""
	}

	sep := StatusOKStyle.Render("  |  ")

	var parts []string

	// 1. Volume
	volPercent := int(math.Round(vol * 100))
	parts = append(parts, LocalFileStyle.Render("VOL:")+" "+StatusOKStyle.Render(fmt.Sprintf("%d%%", volPercent)))

	// 2. Speed
	speedLabel := "1.0x"
	if speedIdx >= 0 && speedIdx < len(playbackSpeeds) {
		speedLabel = playbackSpeeds[speedIdx].Label
	}
	parts = append(parts, LocalFileStyle.Render("SPD:")+" "+StatusOKStyle.Render(speedLabel))

	// 3. Audio Setting (Preset)
	presetName := "Normal"
	if presetIdx >= 0 && presetIdx < len(audioPresets) {
		presetName = audioPresets[presetIdx].Name
	}
	parts = append(parts, LocalFileStyle.Render("EQ:")+" "+StatusOKStyle.Render(presetName))

	// 4. Bitrate
	bitrateStr := "___ kbps"
	if nowPlay != nil && nowPlay.Duration > 0 {
		if strings.HasPrefix(nowPlay.ID, "local:") {
			path := strings.TrimPrefix(nowPlay.ID, "local:")
			if fi, err := os.Stat(path); err == nil {
				kbps := (fi.Size() * 8) / (1000 * int64(nowPlay.Duration))
				bitrateStr = fmt.Sprintf("~%d kbps", kbps)
			}
		} else {
			bitrateStr = "Stream"
		}
	}
	parts = append(parts, LocalFileStyle.Render("BR:")+" "+StatusOKStyle.Render(bitrateStr))

	dashboardText := strings.Join(parts, sep)

	return "  " + dashboardText
}
