package layout

const (
	separatorHeight = 1
	statusHeight    = 1
	progressHeight  = 3
	minMainWidth    = 10
	minMainHeight   = 5
	thirdColRatio   = 0.30
	thirdColMinW    = 25
	tracklistGap    = 1
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
	if mainWidth < minMainWidth {
		mainWidth = minMainWidth
	}

	thirdColumnWidth := int(float64(mainWidth) * thirdColRatio)
	maxThirdColumnWidth := mainWidth - tracklistGap - 1
	if maxThirdColumnWidth < 1 {
		maxThirdColumnWidth = 1
	}
	if thirdColumnWidth < thirdColMinW && maxThirdColumnWidth >= thirdColMinW {
		thirdColumnWidth = thirdColMinW
	}
	if thirdColumnWidth > maxThirdColumnWidth {
		thirdColumnWidth = maxThirdColumnWidth
	}

	tracklistWidth := mainWidth - thirdColumnWidth - tracklistGap

	mainViewHeight := height -
		headerHeight -
		separatorHeight -
		statusHeight -
		progressHeight -
		hintHeight
	if mainViewHeight < minMainHeight {
		mainViewHeight = minMainHeight
	}

	return UILayout{
		Width:            width,
		Height:           height,
		SidebarWidth:     sidebarWidth,
		HeaderHeight:     headerHeight,
		SeparatorHeight:  separatorHeight,
		StatusHeight:     statusHeight,
		ProgressHeight:   progressHeight,
		HintHeight:       hintHeight,
		MainWidth:        mainWidth,
		MainViewHeight:   mainViewHeight,
		TracklistWidth:   tracklistWidth,
		ThirdColumnWidth: thirdColumnWidth,
	}
}

func (l UILayout) ProgressBarTop() int {
	return l.HeaderHeight + l.SeparatorHeight + l.MainViewHeight + l.StatusHeight
}
