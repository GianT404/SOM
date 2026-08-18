package ui

import (
	"som/internal/domain"
	"som/internal/playlist"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type LeftPanel struct {
	provider     domain.MusicProvider
	input        textinput.Model
	spinner      spinner.Model
	tracks       []domain.Track
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

	inputSearch   string
	inputDownload string
	inputPlaylist string

	animTick          int
	showDeletePopup   bool
	deletePopupCursor int
	deleteMsg         string

	suggestions   []string
	suggestCursor int
	suggestOffset int
	suggestFocus  bool
}

const suggestMaxShow = 5

func NewLeftPanel(prov domain.MusicProvider) LeftPanel {
	ti := textinput.New()
	ti.CharLimit = 120
	ti.PromptStyle = InputPromptStyle
	ti.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = StatusMsgStyle

	plInput := textinput.New()
	plInput.CharLimit = 50
	plInput.Prompt = ""

	panel := LeftPanel{
		provider:  prov,
		input:     ti,
		spinner:   sp,
		activeTab: SideDownloads,
		plInput:   plInput,
	}

	if store, err := playlist.NewStore(); err == nil {
		panel.plStore = store
		if pls, err := store.Load(); err == nil {
			panel.playlists = pls
		}
	}

	panel.scanLocalFiles()
	return panel
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

						if p.plCursor >= len(p.activePlaylist.Tracks) {
							p.plCursor = len(p.activePlaylist.Tracks) - 1
						}
						if p.plCursor < 0 {
							p.plCursor = 0
						}
						if p.plOffset > p.plCursor {
							p.plOffset = p.plCursor
						}

					} else if p.activePlaylist == nil && len(p.playlists) > 0 && p.plCursor < len(p.playlists) {
						plID := p.playlists[p.plCursor].ID
						p.plStore.DeletePlaylist(plID)
						p.playlists = append(p.playlists[:p.plCursor], p.playlists[p.plCursor+1:]...)

						if p.plCursor >= len(p.playlists) {
							p.plCursor = len(p.playlists) - 1
						}
						if p.plCursor < 0 {
							p.plCursor = 0
						}
						if p.plOffset > p.plCursor {
							p.plOffset = p.plCursor
						}
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

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if p.input.Focused() && p.suggestFocus && len(p.suggestions) > 0 {
				// Chọn gợi ý đang highlight trong danh sách gợi ý.
				q := strings.TrimSpace(p.suggestions[p.suggestCursor])
				p.input.SetValue(q)
				p.input.CursorEnd()
				p.suggestions = nil
				p.suggestFocus = false
				p.suggestCursor = 0
				if q == "" || p.loading {
					return p, nil
				}
				if p.searchOnEnter {
					p.loading = true
					p.errMsg = ""
					p.tracks = nil
					cmds = append(cmds, p.spinner.Tick, searchCmd(p.provider, q))
				}
				return p, tea.Batch(cmds...)
			}
			if p.input.Focused() {
				q := strings.TrimSpace(p.input.Value())
				if q == "" || p.loading {
					return p, nil
				}
				p.suggestions = nil
				p.suggestFocus = false
				p.suggestCursor = 0
				if p.searchOnEnter {
					p.loading = true
					p.errMsg = ""
					p.tracks = nil
					cmds = append(cmds, p.spinner.Tick, searchCmd(p.provider, q))
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
					plTracks := make([]domain.Track, len(p.activePlaylist.Tracks))
					for i, pt := range p.activePlaylist.Tracks {
						plTracks[i] = domain.Track{ID: pt.ID, Title: pt.Title, Artist: pt.Artist, Duration: pt.Duration}
					}
					return p, func() tea.Msg { return PlayPlaylistMsg{Tracks: plTracks, Index: p.plCursor} }
				} else if p.activePlaylist == nil && len(p.playlists) > 0 && p.plCursor < len(p.playlists) {
					p.activePlaylist = &p.playlists[p.plCursor]
					p.plCursor = 0
					p.plOffset = 0
				}
			}

		case "backspace", "esc":
			if p.suggestFocus {
				p.suggestFocus = false
				p.suggestCursor = 0
				p.suggestOffset = 0
				p.input.Focus()
				break
			}
			if len(p.suggestions) > 0 && p.input.Focused() {
				p.suggestions = nil
				p.suggestCursor = 0
				p.suggestOffset = 0
				break
			}
			if p.activeTab == SidePlaylists && p.activePlaylist != nil {
				p.activePlaylist = nil
				p.plCursor = 0
				p.plOffset = 0
			}

		case "up":
			if p.suggestFocus {
				if p.suggestCursor > 0 {
					p.suggestCursor--
					if p.suggestCursor < p.suggestOffset {
						p.suggestOffset--
					}
				} else {
					p.suggestFocus = false
					p.input.Focus()
				}
				break
			}
			if p.input.Focused() {
				break
			}
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

		case "down":
			if p.suggestFocus {
				if p.suggestCursor < len(p.suggestions)-1 {
					p.suggestCursor++
					if p.suggestCursor >= p.suggestOffset+suggestMaxShow {
						p.suggestOffset++
					}
				} else {
					// Hết danh sách gợi ý → chuyển focus xuống kết quả.
					p.suggestFocus = false
					p.suggestions = nil
					p.suggestCursor = 0
					p.suggestOffset = 0
					p.input.Blur()
				}
				return p, nil
			}
			if p.input.Focused() {
				if len(p.suggestions) > 0 && p.activeTab == SideSearch {
					p.suggestFocus = true
					p.suggestCursor = 0
				} else if p.activeTab == SideSearch || p.activeTab == SideDownloads {
					p.input.Blur()
				}
				return p, nil
			}
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
						if p.dlCursor >= p.dlOffset+p.visibleRows()+1 {
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

		case "delete":
			if focused && !p.input.Focused() && p.activeTab == SidePlaylists && p.plStore != nil {
				if p.activePlaylist != nil && len(p.activePlaylist.Tracks) > 0 && p.plCursor < len(p.activePlaylist.Tracks) {
					p.deleteMsg = "Delete track from playlist?"
					p.showDeletePopup = true
					p.deletePopupCursor = 0
				} else if p.activePlaylist == nil && len(p.playlists) > 0 && p.plCursor < len(p.playlists) {
					p.deleteMsg = "Delete playlist?"
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
				return p, tea.Batch(p.spinner.Tick, downloadCmd(p.provider, t, dir))
			}

		case "/":
			if p.activeTab == SidePlaylists {
				if !p.input.Focused() && p.activePlaylist == nil {
					p.showPlInput = true
					p.plInput.Focus()
					p.plInput.SetValue("")
				}
			} else if !p.input.Focused() {
				p.input.Focus()
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

	case SuggestDebounceMsg:
		if p.activeTab != SideSearch || !p.input.Focused() {
			break
		}
		cur := strings.TrimSpace(p.input.Value())
		if cur != "" && cur == msg.Query {
			cmds = append(cmds, suggestCmd(cur))
		}

	case SuggestionsMsg:
		if p.activeTab != SideSearch || !p.input.Focused() {
			break
		}
		cur := strings.TrimSpace(p.input.Value())
		if cur != msg.Query {
			break // kết quả đã cũ, bỏ qua
		}
		if msg.Err != nil {
			p.suggestions = nil
			p.suggestCursor = 0
			p.suggestOffset = 0
			break
		}
		p.suggestions = msg.Items
		if p.suggestCursor >= len(p.suggestions) {
			p.suggestCursor = 0
		}
		if p.suggestOffset > len(p.suggestions)-suggestMaxShow {
			p.suggestOffset = len(p.suggestions) - suggestMaxShow
		}
		if p.suggestOffset < 0 {
			p.suggestOffset = 0
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
	oldVal := strings.TrimSpace(p.input.Value())
	p.input, inputCmd = p.input.Update(msg)
	cmds = append(cmds, inputCmd)

	if p.activeTab == SideSearch && p.input.Focused() && !p.showDeletePopup {
		newVal := strings.TrimSpace(p.input.Value())
		if newVal != oldVal {
			p.suggestFocus = false
			if newVal == "" {
				p.suggestions = nil
				p.suggestCursor = 0
			} else {
				cmds = append(cmds, suggestDebounceCmd(newVal))
			}
		}
	} else if len(p.suggestions) > 0 && !p.input.Focused() {
		p.suggestions = nil
		p.suggestCursor = 0
		p.suggestFocus = false
	}

	if p.activeTab == SideDownloads {
		localsCount := len(p.getFilteredLocals())
		if p.dlOffset > 0 && p.dlOffset >= localsCount {
			p.dlOffset = maxInt(localsCount-1, 0)
		}
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
	rows := p.height - 7
	if rows < 3 {
		return 3
	}
	return rows
}

func (p LeftPanel) isDownloaded(t domain.Track) bool {
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
	tokens := strings.Fields(q)
	var filtered []LocalFile
	for _, f := range p.locals {
		if localMatches(f, tokens) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

func localMatches(f LocalFile, tokens []string) bool {
	hay := strings.ToLower(f.Name + " " + f.Artist)
	for _, tok := range tokens {
		if !strings.Contains(hay, tok) {
			return false
		}
	}
	return true
}

func saveInputForTab(p *LeftPanel, tab SidebarItem) {
	switch tab {
	case SideSearch:
		p.inputSearch = p.input.Value()
	case SideDownloads:
		p.inputDownload = p.input.Value()
	case SidePlaylists:
		p.inputPlaylist = p.input.Value()
	}
}

func loadInputForTab(p *LeftPanel, tab SidebarItem) {
	switch tab {
	case SideSearch:
		p.input.SetValue(p.inputSearch)
	case SideDownloads:
		p.input.SetValue(p.inputDownload)
	case SidePlaylists:
		p.input.SetValue(p.inputPlaylist)
	}
}
