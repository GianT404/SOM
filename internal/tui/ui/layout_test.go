package ui

import "testing"

func TestUILayoutReservesTerminalHeight(t *testing.T) {
	tests := []struct {
		name      string
		width     int
		height    int
		sidebar   int
		header    int
		hideHint  bool
	}{
		{name: "normal", width: 120, height: 40, sidebar: 18, header: 2},
		{name: "hidden hint", width: 120, height: 40, sidebar: 18, header: 2, hideHint: true},
		{name: "narrow", width: 20, height: 30, sidebar: 18, header: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout := NewUILayout(tt.width, tt.height, tt.sidebar, tt.header, tt.hideHint)

			reserved := layout.HeaderHeight +
				layout.SeparatorHeight +
				layout.MainViewHeight +
				layout.StatusHeight +
				layout.ProgressHeight +
				layout.HintHeight
			if reserved != tt.height && tt.height > 0 {
				t.Fatalf("vertical layout reserves %d rows, want %d", reserved, tt.height)
			}

			if layout.MainWidth < layoutMinMainWidth {
				t.Fatalf("main width %d below minimum %d", layout.MainWidth, layoutMinMainWidth)
			}
			if layout.MainViewHeight < layoutMinMainHeight {
				t.Fatalf("main height %d below minimum %d", layout.MainViewHeight, layoutMinMainHeight)
			}

			if layout.TracklistWidth < 1 || layout.ThirdColumnWidth < 1 {
				t.Fatalf("invalid columns: tracklist=%d third=%d", layout.TracklistWidth, layout.ThirdColumnWidth)
			}
			if available := layout.MainWidth - layoutTracklistGap - 1; available >= layoutThirdColMinW &&
				layout.ThirdColumnWidth < layoutThirdColMinW {
				t.Fatalf("third column %d below preferred minimum %d when space is available",
					layout.ThirdColumnWidth, layoutThirdColMinW)
			}

			if got := layout.TracklistWidth + layout.ThirdColumnWidth + layoutTracklistGap; got != layout.MainWidth {
				t.Fatalf("horizontal layout uses %d columns, want %d", got, layout.MainWidth)
			}

			wantProgressTop := layout.HeaderHeight +
				layout.SeparatorHeight +
				layout.MainViewHeight +
				layout.StatusHeight
			if got := layout.ProgressBarTop(); got != wantProgressTop {
				t.Fatalf("progress bar top=%d, want %d", got, wantProgressTop)
			}
		})
	}
}

func TestThirdColumnLayoutKeepsSectionsInsideContainer(t *testing.T) {
	for _, height := range []int{3, 5, 10, 20, 40} {
		layout := NewThirdColumnLayout(60, height)

		if got := layout.SpectrumHeight + layout.LyricsHeight + layout.StatsHeight; got != layout.Height {
			t.Fatalf("height=%d sections use %d rows, want %d", height, got, layout.Height)
		}
		if layout.SpectrumHeight < 1 || layout.LyricsHeight < 1 || layout.StatsHeight < 1 {
			t.Fatalf("height=%d has invalid section heights: spectrum=%d lyrics=%d stats=%d",
				height, layout.SpectrumHeight, layout.LyricsHeight, layout.StatsHeight)
		}
		if layout.SpectrumInnerWidth < 1 || layout.LyricsInnerWidth < 1 {
			t.Fatalf("height=%d has invalid inner widths: spectrum=%d lyrics=%d",
				height, layout.SpectrumInnerWidth, layout.LyricsInnerWidth)
		}
		if layout.SpectrumInnerHeight < 1 || layout.LyricsInnerHeight < 1 {
			t.Fatalf("height=%d has invalid inner heights: spectrum=%d lyrics=%d",
				height, layout.SpectrumInnerHeight, layout.LyricsInnerHeight)
		}
	}
}
