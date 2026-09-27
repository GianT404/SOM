package layout

const (
	tracklistTickWidth  = 4
	tracklistCheckWidth = 2
	tracklistIndexWidth = 3
	tracklistTimeWidth  = 5
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

func NewTracklistLayout(width int, selectMode bool) TracklistLayout {
	if width < 1 {
		width = 1
	}

	tickWidth := 0
	if selectMode {
		tickWidth = tracklistTickWidth
	}

	availableTextWidth := width - tickWidth - tracklistCheckWidth - tracklistIndexWidth - tracklistTimeWidth - tracklistSpacing
	if availableTextWidth < tracklistMinTextW {
		availableTextWidth = tracklistMinTextW
	}

	titleWidth := int(float64(availableTextWidth) * tracklistTitleRatio)
	artistWidth := availableTextWidth - titleWidth

	return TracklistLayout{
		Width:       width,
		TickWidth:   tickWidth,
		CheckWidth:  tracklistCheckWidth,
		IndexWidth:  tracklistIndexWidth,
		TimeWidth:   tracklistTimeWidth,
		TitleWidth:  titleWidth,
		ArtistWidth: artistWidth,
	}
}
