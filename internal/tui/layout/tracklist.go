package layout

const (
	tracklistTickWidth  = 4
	tracklistCheckWidth = 8
	tracklistIndexWidth = 3
	tracklistTimeWidth  = 6
	tracklistSpacing    = 8
	tracklistMinTextW   = 10
	tracklistTitleRatio = 0.70
)

type TracklistLayout struct {
	Width       int
	TickWidth   int
	CheckWidth  int
	IndexWidth  int
	TimeWidth   int
	TitleWidth  int
	ArtistWidth int
}

func NewTracklistLayout(width int, selectMode, showCheckColumn bool) TracklistLayout {
	if width < 1 {
		width = 1
	}

	tickWidth := 0
	if selectMode {
		tickWidth = tracklistTickWidth
	}

	checkWidth := 0
	if showCheckColumn {
		checkWidth = tracklistCheckWidth
	}

	availableTextWidth := width - tickWidth - checkWidth - tracklistIndexWidth - tracklistTimeWidth - tracklistSpacing
	if availableTextWidth < tracklistMinTextW {
		availableTextWidth = tracklistMinTextW
	}

	// Khi không có cột Saved, giữ nguyên Title và dành phần trống cho Artist.
	titleBudgetWidth := availableTextWidth
	if !showCheckColumn {
		titleBudgetWidth -= tracklistCheckWidth
		if titleBudgetWidth < tracklistMinTextW {
			titleBudgetWidth = tracklistMinTextW
		}
	}

	titleWidth := int(float64(titleBudgetWidth) * tracklistTitleRatio)
	artistWidth := availableTextWidth - titleWidth

	return TracklistLayout{
		Width:       width,
		TickWidth:   tickWidth,
		CheckWidth:  checkWidth,
		IndexWidth:  tracklistIndexWidth,
		TimeWidth:   tracklistTimeWidth,
		TitleWidth:  titleWidth,
		ArtistWidth: artistWidth,
	}
}
