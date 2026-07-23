package ui

import (
	"som/internal/playlist"
	"som/internal/tui/api"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type LeftPanel struct {
	client       *api.Client
	input        textinput.Model
	spinner      spinner.Model
	tracks       []api.Track
	locals       []LocalFile
	searchCursor int
	searchOffset int
	dlCursor     int
	dlOffset     int
	plCursor     int
	plOffset     int

	loading         bool
	searched        bool
	errMsg          string
	loadingStream   bool
	loadingDownload bool
	width           int
	height          int
	searchOnEnter   bool
	activeTab       SidebarItem
	plStore         *playlist.Store
	playlists       []playlist.Playlist
	activePlaylist  *playlist.Playlist
	plInput         textinput.Model
	showPlInput     bool

	showAddPopup bool
	popupCursor  int

	showDeletePopup   bool
	deletePopupCursor int
	deleteMsg         string
}

func NewLeftPanel(c *api.Client) LeftPanel {
	ti := textinput.New()
	ti.CharLimit = 120
	ti.PromptStyle = InputPromptStyle
	ti.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = StatusMsgStyle

	plInput := textinput.New()
	plInput.CharLimit = 50
	plInput.Prompt = "Tên playlist: "

	p := LeftPanel{
		client:    c,
		input:     ti,
		spinner:   sp,
		activeTab: SideDownloads,
		plInput:   plInput,
	}

	if store, err := playlist.NewStore(); err == nil {
		p.plStore = store
		if pls, err := store.Load(); err == nil {
			p.playlists = pls
		}
	}

	p.scanLocalFiles()
	return p
}

func (p *LeftPanel) SetSize(mainW int, contentH int) {
	p.width = mainW
	p.height = contentH
	p.input.Width = maxInt(mainW-6, 10)
}

func (p LeftPanel) Init() tea.Cmd { return textinput.Blink }

func (p LeftPanel) Update(msg tea.Msg, focused bool) (LeftPanel, tea.Cmd) {
	var cmds []tea.Cmd

	if p.showDeletePopup {
		if k, ok := msg.(tea.KeyMsg); ok {
			switch k.String() {
			case "left", "right", "up", "down", "h", "l":
				if p.deletePopupCursor == 0 {
					p.deletePopupCursor = 1
				} else {
					p.deletePopupCursor = 0
				}
			case "enter":
				if p.deletePopupCursor == 1 {
					if p.activePlaylist != nil && len(p.activePlaylist.Tracks) > 0 && p.plCursor < len(p.activePlaylist.Tracks) {
						trackID := p.activePlaylist.Tracks[p.plCursor].ID
						p.plStore.RemoveTrack(p.activePlaylist.ID, trackID)
						var filtered []playlist.Track
						for _, t := range p.activePlaylist.Tracks {
							if t.ID != trackID {
								filtered = append(filtered, t)
							}
						}
						p.activePlaylist.Tracks = filtered
						for i, pl := range p.playlists {
							if pl.ID == p.activePlaylist.ID {
								p.playlists[i] = *p.activePlaylist
							}
						}
					} else if p.activePlaylist == nil && len(p.playlists) > 0 && p.plCursor < len(p.playlists) {
						plID := p.playlists[p.plCursor].ID
						p.plStore.DeletePlaylist(plID)
						p.playlists = append(p.playlists[:p.plCursor], p.playlists[p.plCursor+1:]...)
					}
				}
				p.showDeletePopup = false
				return p, nil
			case "esc":
				p.showDeletePopup = false
				return p, nil
			}
		}
	}

	if p.showPlInput {
		if k, ok := msg.(tea.KeyMsg); ok {
			switch k.String() {
			case "enter":
				name := strings.TrimSpace(p.plInput.Value())
				if name != "" && p.plStore != nil {
					if pl, err := p.plStore.CreatePlaylist(name); err == nil {
						p.playlists = append(p.playlists, pl)
					}
				}
				p.showPlInput = false
				p.plInput.Blur()
				p.plInput.SetValue("")
				return p, nil
			case "esc":
				p.showPlInput = false
				p.plInput.Blur()
				p.plInput.SetValue("")
				return p, nil
			}
		}
		var plInputCmd tea.Cmd
		p.plInput, plInputCmd = p.plInput.Update(msg)
		cmds = append(cmds, plInputCmd)
		return p, tea.Batch(cmds...)
	}

	if p.showAddPopup {
		if k, ok := msg.(tea.KeyMsg); ok {
			switch k.String() {
			case "up", "k":
				if p.popupCursor > 0 {
					p.popupCursor--
				}
			case "down", "j":
				if p.popupCursor < len(p.playlists)-1 {
					p.popupCursor++
				}
			case "enter":
				var selectedTrack playlist.Track
				if p.activeTab == SideSearch && p.searchCursor < len(p.tracks) {
					t := p.tracks[p.searchCursor]
					selectedTrack = playlist.Track{ID: t.ID, Title: t.Title, Artist: t.Artist, Duration: t.Duration, IsLocal: p.isDownloaded(t)}
				} else if p.activeTab == SideDownloads {
					locals := p.getFilteredLocals()
					if p.dlCursor < len(locals) {
						f := locals[p.dlCursor]
						selectedTrack = playlist.Track{ID: "local:" + f.Path, Title: f.Name, Artist: f.Artist, Duration: f.Duration, IsLocal: true}
					}
				}
				if selectedTrack.ID != "" && p.plStore != nil {
					pl := p.playlists[p.popupCursor]
					if err := p.plStore.AddTrack(pl.ID, selectedTrack); err == nil {
						p.playlists[p.popupCursor].Tracks = append(p.playlists[p.popupCursor].Tracks, selectedTrack)
					}
				}
				p.showAddPopup = false
				return p, nil
			case "esc":
				p.showAddPopup = false
				return p, nil
			}
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if p.input.Focused() {
				q := strings.TrimSpace(p.input.Value())
				if q == "" || p.loading {
					return p, nil
				}
				if p.searchOnEnter {
					p.loading = true
					p.errMsg = ""
					p.tracks = nil
					cmds = append(cmds, p.spinner.Tick, searchCmd(p.client, q))
				}
				return p, tea.Batch(cmds...)
			}
			if !focused {
				break
			}

			if p.activeTab == SideSearch {
				if len(p.tracks) > 0 && p.searchCursor < len(p.tracks) {
					t := p.tracks[p.searchCursor]
					return p, func() tea.Msg { return PlayStartedMsg{Track: t} }
				}
			} else if p.activeTab == SideDownloads {
				locals := p.getFilteredLocals()
				if len(locals) > 0 && p.dlCursor < len(locals) {
					f := locals[p.dlCursor]
					return p, func() tea.Msg { return PlayLocalMsg{Path: f.Path, Title: f.Name} }
				}
			} else if p.activeTab == SidePlaylists {
				if p.activePlaylist != nil && len(p.activePlaylist.Tracks) > 0 && p.plCursor < len(p.activePlaylist.Tracks) {
					plTracks := make([]api.Track, len(p.activePlaylist.Tracks))
					for i, pt := range p.activePlaylist.Tracks {
						plTracks[i] = api.Track{ID: pt.ID, Title: pt.Title, Artist: pt.Artist, Duration: pt.Duration}
					}
					return p, func() tea.Msg { return PlayPlaylistMsg{Tracks: plTracks, Index: p.plCursor} }
				} else if p.activePlaylist == nil && len(p.playlists) > 0 && p.plCursor < len(p.playlists) {
					p.activePlaylist = &p.playlists[p.plCursor]
					p.plCursor = 0
					p.plOffset = 0
				}
			}

		case "backspace", "esc":
			if p.activeTab == SidePlaylists && p.activePlaylist != nil {
				p.activePlaylist = nil
				p.plCursor = 0
				p.plOffset = 0
			}

		case "up", "k":
			if focused && !p.input.Focused() {
				if p.activeTab == SideSearch && p.searchCursor > 0 {
					p.searchCursor--
					if p.searchCursor < p.searchOffset {
						p.searchOffset = p.searchCursor
					}
				} else if p.activeTab == SideDownloads && p.dlCursor > 0 {
					p.dlCursor--
					if p.dlCursor < p.dlOffset {
						p.dlOffset = p.dlCursor
					}
				} else if p.activeTab == SidePlaylists && p.plCursor > 0 {
					p.plCursor--
					if p.plCursor < p.plOffset {
						p.plOffset = p.plCursor
					}
				}
			}

		case "down", "j":
			if focused && !p.input.Focused() {
				items := p.itemCount()
				if p.activeTab == SideSearch {
					if p.searchCursor < items-1 {
						p.searchCursor++
						if p.searchCursor >= p.searchOffset+p.visibleRows() {
							p.searchOffset++
						}
					}
				} else if p.activeTab == SideDownloads {
					if p.dlCursor < items-1 {
						p.dlCursor++
						if p.dlCursor >= p.dlOffset+p.visibleRows() {
							p.dlOffset++
						}
					}
				} else if p.activeTab == SidePlaylists {
					if p.plCursor < items-1 {
						p.plCursor++
						if p.plCursor >= p.plOffset+p.visibleRows() {
							p.plOffset++
						}
					}
				}
			}

		case "n", "c":
			if p.activeTab == SidePlaylists && !p.input.Focused() && p.activePlaylist == nil {
				p.showPlInput = true
				p.plInput.Focus()
				p.plInput.SetValue("")
			}

		case "a":
			if !p.input.Focused() && (p.activeTab == SideSearch || p.activeTab == SideDownloads) {
				if len(p.playlists) == 0 {
					p.errMsg = "Chưa có playlist nào! Hãy sang tab Playlists bấm 'n' để tạo."
				} else {
					p.showAddPopup = true
					p.popupCursor = 0
				}
			}

		case "delete":
			if focused && !p.input.Focused() && p.activeTab == SidePlaylists && p.plStore != nil {
				if p.activePlaylist != nil && len(p.activePlaylist.Tracks) > 0 && p.plCursor < len(p.activePlaylist.Tracks) {
					p.deleteMsg = "Xóa bài hát này khỏi playlist?"
					p.showDeletePopup = true
					p.deletePopupCursor = 0
				} else if p.activePlaylist == nil && len(p.playlists) > 0 && p.plCursor < len(p.playlists) {
					p.deleteMsg = "Xóa playlist này?"
					p.showDeletePopup = true
					p.deletePopupCursor = 0
				}
			}

		case "d":
			if focused && !p.input.Focused() && p.activeTab == SideSearch && len(p.tracks) > 0 && p.searchCursor < len(p.tracks) && !p.loading {
				t := p.tracks[p.searchCursor]
				dir, err := getDownloadDir()
				if err != nil {
					p.errMsg = "Cannot find home directory"
					return p, nil
				}
				p.loadingDownload = true
				return p, tea.Batch(p.spinner.Tick, downloadCmd(p.client, t, dir))
			}

		case "/":
			if !p.input.Focused() {
				p.input.Focus()
				p.input.SetValue("")
			}
			return p, nil
		}

	case SearchResultMsg:
		p.loading = false
		p.searched = true
		p.searchCursor = 0
		p.searchOffset = 0
		if msg.Err != nil {
			p.errMsg = msg.Err.Error()
			p.tracks = nil
		} else {
			p.tracks = msg.Tracks
		}

	case DownloadDoneMsg:
		p.loadingDownload = false
		if msg.Err == nil {
			p.scanLocalFiles()
		}

	case spinner.TickMsg:
		if p.loading || p.loadingDownload || p.loadingStream {
			var cmd tea.Cmd
			p.spinner, cmd = p.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	var inputCmd tea.Cmd
	p.input, inputCmd = p.input.Update(msg)
	cmds = append(cmds, inputCmd)

	if !p.input.Focused() && p.activeTab == SideDownloads {
		localsCount := len(p.getFilteredLocals())
		if p.dlCursor >= localsCount && localsCount > 0 {
			p.dlCursor = localsCount - 1
		}
		if p.dlCursor < 0 {
			p.dlCursor = 0
		}
	}
	return p, tea.Batch(cmds...)
}

// --- HELPER FUNCTIONS ---

func (p LeftPanel) itemCount() int {
	if p.activeTab == SideSearch {
		return len(p.tracks)
	} else if p.activeTab == SideDownloads {
		return len(p.getFilteredLocals())
	} else if p.activeTab == SidePlaylists {
		if p.activePlaylist != nil {
			return len(p.activePlaylist.Tracks)
		}
		return len(p.playlists)
	}
	return 0
}

func (p LeftPanel) visibleRows() int {
	rows := p.height - 12
	if rows < 3 {
		return 3
	}
	return rows
}

func (p LeftPanel) isDownloaded(t api.Track) bool {
	if t.ID != "" {
		for _, f := range p.locals {
			if f.VideoID == t.ID {
				return true
			}
		}
	}
	if t.Title == "" {
		return false
	}
	key := normalizeTrackTitle(t.Title)
	for _, f := range p.locals {
		if normalizeTrackTitle(f.Name) != key {
			continue
		}
		diff := f.Duration - t.Duration
		if diff < 0 {
			diff = -diff
		}
		if diff <= 2 {
			return true
		}
	}
	return false
}

func normalizeTrackTitle(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

func (p LeftPanel) getFilteredLocals() []LocalFile {
	q := strings.ToLower(strings.TrimSpace(p.input.Value()))
	if q == "" {
		return p.locals
	}
	var filtered []LocalFile
	for _, f := range p.locals {
		if strings.Contains(strings.ToLower(f.Name), q) || strings.Contains(strings.ToLower(f.Artist), q) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}
