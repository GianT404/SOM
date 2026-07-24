package ui

import (
	"log"
	"math"
	"strings"
	"time"

	"som/internal/tui/audio"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type visTickMsg time.Time

func visTick() tea.Cmd {
	return tea.Tick(33*time.Millisecond, func(t time.Time) tea.Msg {
		return visTickMsg(t)
	})
}

const paletteVisBands = 28

type CommandPalette struct {
	visible bool

	capture   *audio.Capture
	captureOK bool
	amps      []float64
	phase     float64
	width     int
	height    int
}

func NewCommandPalette() CommandPalette {
	return CommandPalette{
		capture: audio.New(),
		amps:    make([]float64, paletteVisBands),
	}
}

func (m *CommandPalette) Open() tea.Cmd {
	m.visible = true
	m.amps = make([]float64, paletteVisBands)
	m.phase = 0

	if err := m.capture.Start(paletteVisBands); err == nil {
		m.captureOK = true
	} else {
		m.captureOK = false
		log.Printf("visualizer: capture unavailable: %v", err)
	}

	return visTick()
}

func (m *CommandPalette) Close() {
	m.visible = false
	m.capture.Stop()
	m.captureOK = false
}

func (m CommandPalette) Visible() bool { return m.visible }

func (m CommandPalette) Update(msg tea.Msg) (CommandPalette, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
	}

	if !m.visible {
		return m, nil
	}

	if _, ok := msg.(visTickMsg); ok {
		if snap := m.capture.Bands(); snap != nil {
			m.amps = snap
		}
		return m, visTick()
	}

	return m, nil
}

func (m CommandPalette) View() string {
	if !m.captureOK {
		return DimItemStyle.Render("")
	}
	return m.renderVisualizer()
}

var brailleBit = [2][4]uint8{
	{0, 1, 2, 6},
	{3, 4, 5, 7},
}

func (m CommandPalette) renderVisualizer() string {
	if m.width < 10 || m.height < 5 {
		return DimItemStyle.Render("")
	}

	subW := m.width * 2
	subH := m.height * 4

	dots := make([][]float64, subW)
	for i := range dots {
		dots[i] = make([]float64, subH)
		for j := range dots[i] {
			dots[i][j] = -1
		}
	}

	cx := float64(subW) / 2
	cy := float64(subH) / 2

	plotCircle := func(x, y float64) {
		xi := int(math.Round(x))
		yi := int(math.Round(y))
		r := math.Hypot(x-cx, y-cy)

		for dx := 0; dx <= 1; dx++ {
			for dy := 0; dy <= 1; dy++ {
				nx, ny := xi+dx, yi+dy
				if nx >= 0 && nx < subW && ny >= 0 && ny < subH {
					if dots[nx][ny] < 0 || r > dots[nx][ny] {
						dots[nx][ny] = r
					}
				}
			}
		}
	}

	// render bar
	plotBarPoint := func(x, y float64) {
		xi := int(math.Round(x))
		yi := int(math.Round(y))
		if xi < 0 || xi >= subW || yi < 0 || yi >= subH {
			return
		}
		r := math.Hypot(x-cx, y-cy)
		if dots[xi][yi] < 0 || r > dots[xi][yi] {
			dots[xi][yi] = r
		}
	}

	bass, bassN := 0.0, 0
	for i := 0; i < len(m.amps) && i < 4; i++ {
		bass += m.amps[i]
		bassN++
	}
	if bassN > 0 {
		bass /= float64(bassN)
	}

	minDim := math.Min(cx, cy)

	baseR := (minDim * 0.35) + 3.0
	baseR *= (1 + 0.2*bass)
	maxBar := minDim * 0.45

	const circleSteps = 120
	for s := 0; s < circleSteps; s++ {
		theta := 2 * math.Pi * float64(s) / circleSteps
		plotCircle(cx+baseR*math.Cos(theta), cy-baseR*math.Sin(theta))
	}

	n := len(m.amps)
	outerR := baseR + maxBar // (biên độ = 1)
	circumference := 2 * math.Pi * outerR
	slotPx := circumference / float64(n)

	barWidthPx := slotPx + 0.5
	if barWidthPx < 1 {
		barWidthPx = 1
	}
	halfWidthAngle := (barWidthPx / 2) / outerR

	for i := 0; i < n; i++ {
		theta0 := 2*math.Pi*float64(i)/float64(n) + m.phase

		amp := m.amps[i]
		if amp < 0 {
			amp = 0
		} else if amp > 1 {
			amp = 1
		}

		barLen := amp * maxBar
		if barLen < 0.001 {
			continue
		}

		rSteps := int(barLen*3) + 3
		aSteps := int(barWidthPx) + 2

		maxDrop := barWidthPx / 2.0

		for as := 0; as <= aSteps; as++ {
			frac := (float64(as)/float64(aSteps))*2.0 - 1.0

			capReduction := maxDrop * (1.0 - math.Sqrt(1.0-frac*frac))

			capReduction *= (1.0 - amp)

			targetLen := barLen - capReduction
			if targetLen < 0 {
				targetLen = 0
			}

			theta := theta0 - halfWidthAngle + 2*halfWidthAngle*float64(as)/float64(aSteps)

			for rs := 0; rs <= rSteps; rs++ {
				ratio := float64(rs) / float64(rSteps)
				r := baseR + targetLen*ratio
				plotBarPoint(cx+r*math.Cos(theta), cy-r*math.Sin(theta))
			}
		}
	}

	palette := []lipgloss.Color{
		"#9D311A",
		"#C84328",
		"#E8593C",
		"#EF9F27",
		"#FFD700",
	}

	var out strings.Builder
	for row := 0; row < m.height; row++ {
		for col := 0; col < m.width; col++ {
			var mask uint8
			any := false
			sumR := 0.0
			count := 0

			for dc := 0; dc < 2; dc++ {
				for dr := 0; dr < 4; dr++ {
					val := dots[col*2+dc][row*4+dr]
					if val >= 0 {
						mask |= 1 << brailleBit[dc][dr]
						any = true
						sumR += val
						count++
					}
				}
			}

			if any {
				avgR := sumR / float64(count)
				var colorIdx int

				if avgR <= baseR {
					colorIdx = 0
				} else {
					norm := (avgR - baseR) / maxBar
					if norm > 1 {
						norm = 1
					}
					colorIdx = int(norm * float64(len(palette)-1))
					if colorIdx > len(palette)-1 {
						colorIdx = len(palette) - 1
					}
				}

				runeStr := string(rune(0x2800 + int(mask)))
				out.WriteString(lipgloss.NewStyle().Foreground(palette[colorIdx]).Render(runeStr))
			} else {
				out.WriteByte(' ')
			}
		}
		if row < m.height-1 {
			out.WriteByte('\n')
		}
	}

	return out.String()
}
