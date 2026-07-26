package ui

import (
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	"som/internal/tui/audio"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type visTickMsg time.Time

var (
	visStylesOut = []lipgloss.Style{
		lipgloss.NewStyle().Foreground(lipgloss.Color("#9D311A")),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#C84328")),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#E8593C")),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#EF9F27")),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")),
	}

	visStylesIn = []lipgloss.Style{
		lipgloss.NewStyle().Foreground(lipgloss.Color("#1B0604")),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#3A0F08")),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#5E190D")),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#9D311A")),
	}

	visStyleGlitch1 = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF"))
	visStyleGlitch2 = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
)

func visTick() tea.Cmd {
	return tea.Tick(33*time.Millisecond, func(t time.Time) tea.Msg {
		return visTickMsg(t)
	})
}

const paletteVisBands = 28

type CommandPalette struct {
	visible   bool
	is3D      bool
	capture   *audio.Capture
	captureOK bool
	amps      []float64
	phase     float64
	width     int
	height    int
	peaks     []float64
	peakHold  []int
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
	m.peaks = make([]float64, paletteVisBands)
	m.peakHold = make([]int, paletteVisBands)
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
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "l", "L":
			m.is3D = !m.is3D
		}
	}
	if _, ok := msg.(visTickMsg); ok {
		//xoayy
		m.phase -= 0.02

		if m.phase <= -2*math.Pi {
			m.phase += 2 * math.Pi
		} else if m.phase >= 2*math.Pi {
			m.phase -= 2 * math.Pi
		}
		if snap := m.capture.Bands(); snap != nil {
			for i, v := range snap {
				if v >= m.amps[i] {
					m.amps[i] = v
				} else {
					m.amps[i] -= 0.06
					if m.amps[i] < v {
						m.amps[i] = v
					}
				}
			}
			for i := range m.amps {
				if m.amps[i] >= m.peaks[i] {
					m.peaks[i] = m.amps[i]
					m.peakHold[i] = 15
				} else {
					if m.peakHold[i] > 0 {
						m.peakHold[i]--
					} else {
						m.peaks[i] -= 0.025
						if m.peaks[i] < m.amps[i] {
							m.peaks[i] = m.amps[i]
						}
					}
				}
			}
		}
		return m, visTick()
	}

	return m, nil
}

func (m CommandPalette) View() string {
	if !m.captureOK {
		return DimItemStyle.Render("visualizer unavailable")
	}

	if m.is3D {
		return m.Render3DVisualizer()
	} else {
		return m.renderVisualizer()
	}
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

	plotBarPoint := func(x, y float64) {
		xi := int(math.Round(x))
		yi := int(math.Round(y))

		if xi < 0 || xi >= subW-1 || yi < 0 || yi >= subH-1 {
			return
		}
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

	bass, bassN := 0.0, 0
	for i := 0; i < len(m.amps) && i < 4; i++ {
		bass += m.amps[i]
		bassN++
	}
	if bassN > 0 {
		bass /= float64(bassN)
	}

	isChaosFrame := bass > 0.85
	baseCx := float64(subW) / 2
	baseCy := float64(subH) / 2
	minDim := math.Min(baseCx, baseCy)

	if isChaosFrame {
		shakeForce := (bass - 0.85) * 100.0

		cx += (rand.Float64() - 0.5) * shakeForce
		cy += (rand.Float64() - 0.5) * shakeForce
	}
	baseR := (minDim * 0.35) + 3.0
	baseR *= (1 + 0.2*bass)
	maxBar := minDim * 0.45

	const circleSteps = 360
	for s := 0; s < circleSteps; s++ {
		theta := 2 * math.Pi * float64(s) / circleSteps
		plotCircle(cx+baseR*math.Cos(theta), cy-baseR*math.Sin(theta))
	}

	n := len(m.amps)
	outerR := baseR + maxBar
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
		peakAmp := m.peaks[i]

		if amp < 0 {
			amp = 0
		} else if amp > 1 {
			amp = 1
		}
		if peakAmp < 0 {
			peakAmp = 0
		} else if peakAmp > 1 {
			peakAmp = 1
		}

		barLen := amp * maxBar
		peakLen := peakAmp * maxBar

		rSteps := int(barLen*3) + 3
		aSteps := int(barWidthPx) + 2
		maxDrop := barWidthPx / 2.0

		jitterX, jitterY := 0.0, 0.0
		if amp > 0.90 && rand.Float64() < 0.50 {
			jitterX = (rand.Float64() - 0.5) * 12.0
			jitterY = (rand.Float64() - 0.5) * 12.0
		}

		for as := 0; as <= aSteps; as++ {
			frac := (float64(as)/float64(aSteps))*2.0 - 1.0

			capReductionAmp := maxDrop * (1.0 - math.Sqrt(1.0-frac*frac))
			capReductionAmp *= (1.0 - amp)
			targetLen := barLen - capReductionAmp
			if targetLen < 0 {
				targetLen = 0
			}

			capReductionPeak := maxDrop * (1.0 - math.Sqrt(1.0-frac*frac))
			capReductionPeak *= (1.0 - peakAmp)
			peakTargetLen := peakLen - capReductionPeak
			if peakTargetLen < 0 {
				peakTargetLen = 0
			}

			theta := theta0 - halfWidthAngle + 2*halfWidthAngle*float64(as)/float64(aSteps)

			if barLen >= 0.001 {
				for rs := 0; rs <= rSteps; rs++ {
					ratio := float64(rs) / float64(rSteps)
					r := baseR + targetLen*ratio

					xPos := cx + r*math.Cos(theta) + jitterX
					yPos := cy - r*math.Sin(theta) + jitterY

					plotBarPoint(xPos, yPos)
				}

				innerTargetLen := targetLen * 0.4
				if innerTargetLen > 0 {
					innerRSteps := int(innerTargetLen*3) + 2
					for rs := 0; rs <= innerRSteps; rs++ {
						ratio := float64(rs) / float64(innerRSteps)

						r := baseR - innerTargetLen*ratio

						xPos := cx + r*math.Cos(theta) + jitterX*0.3
						yPos := cy - r*math.Sin(theta) + jitterY*0.3

						plotBarPoint(xPos, yPos)
					}
				}
			}

			if peakTargetLen > targetLen+0.5 {
				maxParticles := 5.0

				particleCount := int(maxParticles * (1.0 - peakAmp))
				if particleCount < 1 {
					particleCount = 1
				}

				for p := 0; p < particleCount; p++ {
					spread := (1.0 - peakAmp) * 1.0

					scatterR := baseR + peakTargetLen + (rand.Float64()-0.5)*spread
					scatterTheta := theta + (rand.Float64()-0.5)*(spread/baseR)
					pX := int(math.Round(cx + scatterR*math.Cos(scatterTheta) + jitterX))
					pY := int(math.Round(cy - scatterR*math.Sin(scatterTheta) + jitterY))

					if pX >= 0 && pX < subW && pY >= 0 && pY < subH {
						if dots[pX][pY] < 0 || scatterR > dots[pX][pY] {
							dots[pX][pY] = scatterR
						}
					}
				}
			}
		}
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

				var finalStyle lipgloss.Style
				maxInnerBar := maxBar * 0.4

				if avgR < baseR {
					distInwards := baseR - avgR
					norm := 1.0 - (distInwards / maxInnerBar)
					if norm < 0 {
						norm = 0
					}
					if norm > 1 {
						norm = 1
					}

					colorIdx := int(norm * float64(len(visStylesIn)-1))
					finalStyle = visStylesIn[colorIdx]

				} else {
					norm := (avgR - baseR) / maxBar
					if norm > 1 {
						norm = 1
					}

					colorIdx := int(norm * float64(len(visStylesOut)-1))
					finalStyle = visStylesOut[colorIdx]
				}

				//  (Scanline Aberration)
				if isChaosFrame && avgR > baseR {
					scanline := math.Sin(float64(row) * 1.2)
					if scanline > 0.85 {
						finalStyle = visStyleGlitch1
					} else if scanline < -0.85 {
						finalStyle = visStyleGlitch2
					}
				}

				runeStr := string(rune(0x2800 + int(mask)))
				out.WriteString(finalStyle.Render(runeStr))
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
