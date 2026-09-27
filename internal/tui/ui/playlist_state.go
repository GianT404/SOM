package ui

// rebindActivePlaylist nối lại con trỏ sau khi slice playlist thay đổi.
func (p *LeftPanel) rebindActivePlaylist() {
	if p.activePlaylist == nil {
		return
	}

	id := p.activePlaylist.ID
	for i := range p.playlists {
		if p.playlists[i].ID == id {
			p.activePlaylist = &p.playlists[i]
			return
		}
	}

	p.activePlaylist = nil
	p.plCursor = 0
	p.plOffset = 0
}
