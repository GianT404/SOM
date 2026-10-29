package ui

import (
	"som/internal/domain"

	"charm.land/lipgloss/v2"
)

func (p LeftPanel) ViewQueueContent(w, h int, queue []domain.Track, playingID string) string {
	if len(queue) == 0 {
		return lipgloss.Place(
			w,
			h,
			lipgloss.Center,
			lipgloss.Center,
			DimItemStyle.Render("No tracks in queue.")+"\n"+DimItemStyle.Render("Use ':' -> 'Add to queue'"),
		)
	}

	innerW := w - 4

	listContent := p.renderQueueList(innerW, queue, playingID)

	return listContent
}

func (p LeftPanel) renderQueueList(innerW int, queue []domain.Track, playingID string) string {
	return renderSharedTrackList(
		innerW,
		queue,
		p.qCursor,
		p.qOffset,
		p.visibleRows()+1,
		func(i int, t domain.Track) (string, string, string, int, bool, bool, int) {
			return t.Title, t.Artist, t.ID, t.Duration, false, t.ID == playingID && playingID != "", i + 1
		},
		false, false, false, nil, nil,
	)
}
