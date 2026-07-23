package ui

import (
	"som/internal/tui/api"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type LeftPanel struct {
	client  *api.Client
	input   textinput.Model
	spinner spinner.Model
	tracks  []api.Track
	locals  []LocalFile

	// TÁCH BIỆT CURSOR VÀ OFFSET CHO 2 TAB
	searchCursor int
	searchOffset int
	dlCursor     int
	dlOffset     int

	loading         bool
	searched        bool
	errMsg          string
	loadingStream   bool
	loadingDownload bool
	width           int
	height          int
	searchOnEnter   bool
	activeTab       SidebarItem // Biến theo dõi tab hiện tại
}

func NewLeftPanel(c *api.Client) LeftPanel {
	ti := textinput.New()
	ti.CharLimit = 120
	ti.PromptStyle = InputPromptStyle
	ti.Focus()
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = StatusMsgStyle
	p := LeftPanel{client: c, input: ti, spinner: sp, activeTab: SideDownloads}
	p.scanLocalFiles()
	return p
}

func (p *LeftPanel) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.input.Width = maxInt(w-6, 10)
}

func (p LeftPanel) Init() tea.Cmd { return textinput.Blink }

func (p LeftPanel) Update(msg tea.Msg, focused bool) (LeftPanel, tea.Cmd) {
	var cmds []tea.Cmd
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
			// Xử lý enter theo tab hiện tại
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
				}
			}
		case "down", "j":
			if focused && !p.input.Focused() {
				items := p.itemCount()
				if p.activeTab == SideSearch {
					if p.searchCursor < items-1 {
						p.searchCursor++
						vis := p.visibleRows()
						if p.searchCursor >= p.searchOffset+vis {
							p.searchOffset++
						}
					}
				} else if p.activeTab == SideDownloads {
					if p.dlCursor < items-1 {
						p.dlCursor++
						vis := p.visibleRows()
						if p.dlCursor >= p.dlOffset+vis {
							p.dlOffset++
						}
					}
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

func (p LeftPanel) itemCount() int {
	if p.activeTab == SideSearch {
		return len(p.tracks)
	}
	return len(p.getFilteredLocals())
}

// --- GIỮ NGUYÊN CÁC HÀM BÊN DƯỚI ---
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
