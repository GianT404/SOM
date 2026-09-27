package ui

const (
	layoutSeparatorHeight = 1
	layoutStatusHeight    = 1
	layoutProgressHeight  = 3
	layoutMinMainWidth    = 10
	layoutMinMainHeight   = 5
	layoutThirdColRatio   = 0.30
	layoutThirdColMinW    = 25
	layoutTracklistGap    = 1
)

type UILayout struct {
	Width  int
	Height int

	SidebarWidth int

	HeaderHeight    int
	SeparatorHeight int
	StatusHeight    int
	ProgressHeight  int
	HintHeight      int

	MainWidth      int
	MainViewHeight int

	TracklistWidth   int
	ThirdColumnWidth int
}

func NewUILayout(width, height, sidebarWidth, headerHeight int, hideHint bool) UILayout {
	hintHeight := 0
	if !hideHint {
		hintHeight = 1
	}

	mainWidth := width - sidebarWidth
	if mainWidth < layoutMinMainWidth {
		mainWidth = layoutMinMainWidth
	}

	thirdColumnWidth := int(float64(mainWidth) * layoutThirdColRatio)
	maxThirdColumnWidth := mainWidth - layoutTracklistGap - 1
	if maxThirdColumnWidth < 1 {
		maxThirdColumnWidth = 1
	}
	if thirdColumnWidth < layoutThirdColMinW && maxThirdColumnWidth >= layoutThirdColMinW {
		thirdColumnWidth = layoutThirdColMinW
	}
	if thirdColumnWidth > maxThirdColumnWidth {
		thirdColumnWidth = maxThirdColumnWidth
	}

	tracklistWidth := mainWidth - thirdColumnWidth - layoutTracklistGap

	mainViewHeight := height -
		headerHeight -
		layoutSeparatorHeight -
		layoutStatusHeight -
		layoutProgressHeight -
		hintHeight
	if mainViewHeight < layoutMinMainHeight {
		mainViewHeight = layoutMinMainHeight
	}

	return UILayout{
		Width:            width,
		Height:           height,
		SidebarWidth:     sidebarWidth,
		HeaderHeight:     headerHeight,
		SeparatorHeight:  layoutSeparatorHeight,
		StatusHeight:     layoutStatusHeight,
		ProgressHeight:   layoutProgressHeight,
		HintHeight:       hintHeight,
		MainWidth:        mainWidth,
		MainViewHeight:   mainViewHeight,
		TracklistWidth:   tracklistWidth,
		ThirdColumnWidth: thirdColumnWidth,
	}
}

// ProgressBarTop returns the first terminal row occupied by the progress bar.
func (l UILayout) ProgressBarTop() int {
	return l.HeaderHeight + l.SeparatorHeight + l.MainViewHeight + l.StatusHeight
}
