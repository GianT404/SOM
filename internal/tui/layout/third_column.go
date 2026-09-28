package layout

const (
	thirdColumnLyricRatio      = 0.45
	thirdColumnStatsHeight     = 3
	thirdColumnBoxBorderHeight = 2
	thirdColumnMinInnerWidth   = 1
	thirdColumnMinInnerHeight  = 1
)

type ThirdColumnLayout struct {
	Width  int
	Height int

	SpectrumHeight int
	LyricsHeight   int
	StatsHeight    int

	SpectrumInnerWidth  int
	SpectrumInnerHeight int
	LyricsInnerWidth    int
	LyricsInnerHeight   int
}

func NewThirdColumnLayout(width, height int) ThirdColumnLayout {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}

	statsHeight := thirdColumnStatsHeight

	// Khi container quá thấp, thu nhỏ Session trước
	// để tổng ba section không vượt quá chiều cao.
	if height >= 3 && statsHeight > height-2 {
		statsHeight = height - 2
	}
	if statsHeight < 1 {
		statsHeight = 1
	}

	remainingHeight := height - statsHeight

	lyricsHeight := int(float64(height) * thirdColumnLyricRatio)
	if lyricsHeight < 1 {
		lyricsHeight = 1
	}

	if lyricsHeight > remainingHeight-1 {
		lyricsHeight = remainingHeight - 1
	}
	if lyricsHeight < 1 {
		lyricsHeight = 1
	}

	spectrumHeight := remainingHeight - lyricsHeight
	if spectrumHeight < 1 {
		spectrumHeight = 1
		lyricsHeight = remainingHeight - spectrumHeight
		if lyricsHeight < 1 {
			lyricsHeight = 1
		}
	}

	innerWidth := width - 4
	if innerWidth < thirdColumnMinInnerWidth {
		innerWidth = thirdColumnMinInnerWidth
	}

	spectrumInnerHeight := spectrumHeight - thirdColumnBoxBorderHeight
	if spectrumInnerHeight < thirdColumnMinInnerHeight {
		spectrumInnerHeight = thirdColumnMinInnerHeight
	}

	lyricsInnerHeight := lyricsHeight - thirdColumnBoxBorderHeight
	if lyricsInnerHeight < thirdColumnMinInnerHeight {
		lyricsInnerHeight = thirdColumnMinInnerHeight
	}

	return ThirdColumnLayout{
		Width:  width,
		Height: height,

		SpectrumHeight: spectrumHeight,
		LyricsHeight:   lyricsHeight,
		StatsHeight:    statsHeight,

		SpectrumInnerWidth:  innerWidth,
		SpectrumInnerHeight: spectrumInnerHeight,
		LyricsInnerWidth:    innerWidth,
		LyricsInnerHeight:   lyricsInnerHeight,
	}
}
