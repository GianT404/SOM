package ui

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
)

func TestRenderStatusLineShowsLoadingMoreAnimation(t *testing.T) {
	a := &App{
		sidebarActive: SideSearch,
		left: LeftPanel{
			loadingMore: true,
			spinner:     spinner.New(),
		},
		statusMsg: "Previous operation completed",
		statusAt:  time.Now(),
	}

	got := a.renderStatusLine()
	if !strings.Contains(got, "Loading more...") {
		t.Fatalf("status line = %q, want loading-more status", got)
	}
	if strings.Contains(got, "Previous operation completed") {
		t.Fatalf("previous status should not override loading animation: %q", got)
	}
}
