package ui

import (
	"math"
	"strings"

	"charm.land/lipgloss/v2"
)

// Hàm chiếu 3D -> 2D
func project3D(x, y, z, cx, cy float64) (float64, float64, float64) {
	fov := 140.0
	distance := 80.0

	zAdj := z + distance
	if zAdj <= 0 {
		zAdj = 0.1
	}

	px := cx + (x * fov / zAdj)
	py := cy - (y * fov / zAdj)

	return px, py, zAdj
}

func draw3DLine(dots, zBuffer [][]float64, x0, y0, z0, x1, y1, z1, cx, cy float64, subW, subH int) {
	px0, py0, pz0 := project3D(x0, y0, z0, cx, cy)
	px1, py1, pz1 := project3D(x1, y1, z1, cx, cy)

	xi0, yi0 := int(math.Round(px0)), int(math.Round(py0))
	xi1, yi1 := int(math.Round(px1)), int(math.Round(py1))

	dx := math.Abs(float64(xi1 - xi0))
	dy := -math.Abs(float64(yi1 - yi0))
	sx, sy := -1, -1
	if xi0 < xi1 {
		sx = 1
	}
	if yi0 < yi1 {
		sy = 1
	}
	err := dx + dy

	dist := math.Hypot(float64(xi1-xi0), float64(yi1-yi0))
	zStep := 0.0
	if dist > 0 {
		zStep = (pz1 - pz0) / dist
	}
	currentZ := pz0

	for {
		if xi0 >= 0 && xi0 < subW && yi0 >= 0 && yi0 < subH {
			if currentZ < zBuffer[xi0][yi0] {
				dots[xi0][yi0] = currentZ
				zBuffer[xi0][yi0] = currentZ
			}
		}
		if xi0 == xi1 && yi0 == yi1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			xi0 += sx
		}
		if e2 <= dx {
			err += dx
			yi0 += sy
		}
		currentZ += zStep
	}
}

type vec3 struct{ x, y, z float64 }

func (m CommandPalette) Render3DVisualizer() string {
	if m.width < 10 || m.height < 5 {
		return DimItemStyle.Render("")
	}

	subW := m.width * 2
	subH := m.height * 4

	dots := make([][]float64, subW)
	zBuffer := make([][]float64, subW)
	for i := range dots {
		dots[i] = make([]float64, subH)
		zBuffer[i] = make([]float64, subH)
		for j := range dots[i] {
			dots[i][j] = -1
			zBuffer[i][j] = 9999.0
		}
	}

	cx := float64(subW) / 2
	cy := float64(subH) / 2

	bass, bassN := 0.0, 0
	for i := 0; i < len(m.amps) && i < 4; i++ {
		bass += m.amps[i]
		bassN++
	}
	if bassN > 0 {
		bass /= float64(bassN)
	}

	minDim := math.Min(cx, cy)
	baseR := minDim * 0.40 * (1 + 0.15*bass)
	maxDeform := minDim * 0.35

	const latitudes = 14
	const longitudes = 28

	points := make([][]vec3, latitudes+1)
	for i := range points {
		points[i] = make([]vec3, longitudes+1)
	}

	rotX := m.phase * 0.6
	rotY := m.phase

	for lat := 0; lat <= latitudes; lat++ {
		phi := math.Pi * float64(lat) / float64(latitudes)

		for lon := 0; lon <= longitudes; lon++ {
			theta := 2 * math.Pi * float64(lon) / float64(longitudes)

			idx := int((math.Abs(math.Sin(theta)) + math.Abs(math.Cos(phi))) * 0.5 * float64(len(m.amps)-1))
			if idx < 0 {
				idx = 0
			}
			if idx >= len(m.amps) {
				idx = len(m.amps) - 1
			}

			amp := m.amps[idx]

			noise := math.Sin(theta*6+m.phase*3) * math.Cos(phi*5-m.phase) * amp * 6.0
			r := baseR + (amp * maxDeform) + noise

			rawX := r * math.Sin(phi) * math.Cos(theta)
			rawY := r * math.Cos(phi)
			rawZ := r * math.Sin(phi) * math.Sin(theta)

			tmpY := rawY*math.Cos(rotX) - rawZ*math.Sin(rotX)
			tmpZ := rawY*math.Sin(rotX) + rawZ*math.Cos(rotX)

			finalX := rawX*math.Cos(rotY) + tmpZ*math.Sin(rotY)
			finalY := tmpY
			finalZ := -rawX*math.Sin(rotY) + tmpZ*math.Cos(rotY)

			points[lat][lon] = vec3{finalX, finalY, finalZ}
		}
	}

	for lat := 0; lat < latitudes; lat++ {
		for lon := 0; lon < longitudes; lon++ {
			p0 := points[lat][lon]
			p1 := points[lat][lon+1]
			p2 := points[lat+1][lon]

			draw3DLine(dots, zBuffer, p0.x, p0.y, p0.z, p1.x, p1.y, p1.z, cx, cy, subW, subH)
			draw3DLine(dots, zBuffer, p0.x, p0.y, p0.z, p2.x, p2.y, p2.z, cx, cy, subW, subH)
		}
	}

	var out strings.Builder
	for row := 0; row < m.height; row++ {
		for col := 0; col < m.width; col++ {
			var mask uint8
			any := false
			sumZ := 0.0
			count := 0

			for dc := 0; dc < 2; dc++ {
				for dr := 0; dr < 4; dr++ {
					val := dots[col*2+dc][row*4+dr]
					if val >= 0 {
						mask |= 1 << brailleBit[dc][dr]
						any = true
						sumZ += val
						count++
					}
				}
			}

			if any {
				avgZ := sumZ / float64(count)
				var finalStyle lipgloss.Style

				if avgZ < 75 {
					finalStyle = visStylesOut[4]
				} else if avgZ < 90 {
					finalStyle = visStylesOut[2]
				} else if avgZ < 105 {
					finalStyle = visStylesIn[3]
				} else {
					finalStyle = visStylesIn[1]
				}

				if bass > 0.85 && avgZ < 85 {
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
