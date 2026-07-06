package ui

import (
	"strings"

	"som/internal/tui/api"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type LeftPanel struct {
	client  *api.Client
	input   textinput.Model
	spinner spinner.Model

	tracks   []api.Track
	locals   []LocalFile
	cursor   int
	offset   int
	loading  bool
	searched bool
	errMsg   string

	loadingStream   bool
	loadingDownload bool

	width         int
	height        int
	searchOnEnter bool
}

func NewLeftPanel(c *api.Client) LeftPanel {
	ti := textinput.New()
	ti.CharLimit = 120
	ti.PromptStyle = InputPromptStyle
	ti.Focus()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = StatusMsgStyle

	p := LeftPanel{client: c, input: ti, spinner: sp}
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
					break
				}
				if p.searchOnEnter {
					p.loading = true
					p.errMsg = ""
					p.tracks = nil
					cmds = append(cmds, p.spinner.Tick, searchCmd(p.client, q))
				} else {
					p.input.Blur()
				}
			} else if len(p.tracks) > 0 && p.cursor < len(p.tracks) {
				t := p.tracks[p.cursor]
				return p, func() tea.Msg { return PlayStartedMsg{Track: t} }
			} else {
				locals := p.getFilteredLocals()
				if len(locals) > 0 && p.cursor < len(locals) {
					f := locals[p.cursor]
					return p, func() tea.Msg { return PlayLocalMsg{Path: f.Path, Title: f.Name} }
				}
			}

		case "up", "k":
			if !p.input.Focused() && p.cursor > 0 {
				p.cursor--
				if p.cursor < p.offset {
					p.offset = p.cursor
				}
			}

		case "down", "j":
			if !p.input.Focused() {
				items := p.itemCount()
				if p.cursor < items-1 {
					p.cursor++
					vis := p.visibleRows()
					if p.cursor >= p.offset+vis {
						p.offset++
					}
				}
			}

		case "d":
			if !p.input.Focused() && len(p.tracks) > 0 && p.cursor < len(p.tracks) && !p.loading {
				t := p.tracks[p.cursor]
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
		p.cursor = 0
		p.offset = 0
		p.input.Blur()
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

	if !p.input.Focused() && len(p.tracks) == 0 {
		localsCount := len(p.getFilteredLocals())
		if p.cursor >= localsCount && localsCount > 0 {
			p.cursor = localsCount - 1
		}
		if p.cursor < 0 {
			p.cursor = 0
		}
	}

	return p, tea.Batch(cmds...)
}

// ─── Helpers ───────────────────────────────────────────────────────────

func (p LeftPanel) itemCount() int {
	if len(p.tracks) > 0 {
		return len(p.tracks)
	}
	return len(p.getFilteredLocals())
}

func (p LeftPanel) visibleRows() int {
	rows := p.height - 12
	if rows < 3 {
		return 3
	}
	return rows
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
