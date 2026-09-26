package ui

import (
	"fmt"
	"math"
	"os"
	"strings"

	"som/internal/domain"

	"charm.land/lipgloss/v2"
)

func renderDashboard(hide bool, vol float64, speedIdx int, presetIdx int, nowPlay *domain.Track) string {
	if hide {
		return ""
	}

	// 1. Volume
	volPercent := int(math.Round(vol * 100))
	volStr := styleHint("VOL", fmt.Sprintf("%d%%", volPercent))

	// 2. Speed
	speedLabel := "1.0x"
	if speedIdx >= 0 && speedIdx < len(playbackSpeeds) {
		speedLabel = playbackSpeeds[speedIdx].Label
	}
	spdStr := styleHint("SPD", speedLabel)

	// 3. Audio Setting (Preset)
	presetName := "Normal"
	if presetIdx >= 0 && presetIdx < len(audioPresets) {
		presetName = audioPresets[presetIdx].Name
	}
	eqStr := styleHint("EQ", presetName)

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
	brStr := styleHint("BR", bitrateStr)

	leftColStyle := lipgloss.NewStyle().Width(14)

	row1 := " " + leftColStyle.Render(volStr) + spdStr
	row2 := " " + leftColStyle.Render(eqStr) + brStr

	return row1 + "\n" + row2
}
