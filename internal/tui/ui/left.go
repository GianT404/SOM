package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"som/internal/tui/api"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type LeftTab int

const (
	TabSearch LeftTab = iota
	TabDownloads
)

type LeftPanel struct {
	client  *api.Client
	input   textinput.Model
	spinner spinner.Model

	tab      LeftTab
	tracks   []api.Track
	locals   []LocalFile
	cursor   int
	offset   int
	loading  bool
	searched bool
	errMsg   string

	width  int
	height int
}

func NewLeftPanel(c *api.Client) LeftPanel {
	ti := textinput.New()
	ti.Placeholder = "Search"
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

func (p *LeftPanel) scanLocalFiles() {
	p.locals = nil

	dir, err := getDownloadDir()
	if err != nil {
		p.errMsg = "Cannot find home directory"
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		p.errMsg = fmt.Sprintf("Create dir error: %s", err.Error())
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		p.errMsg = fmt.Sprintf("Dir error: %s", err.Error())
		return
	}
	p.errMsg = ""

	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".m4a") {
			p.locals = append(p.locals, LocalFile{
				Name: strings.TrimSuffix(e.Name(), ".m4a"),
				Path: filepath.Join(dir, e.Name()),
			})
		}
	}
}

func (p LeftPanel) Init() tea.Cmd { return textinput.Blink }
func (p *LeftPanel) SelectedTrack() *api.Track {
	if p.tab == TabSearch && len(p.tracks) > 0 && p.cursor < len(p.tracks) {
		t := p.tracks[p.cursor]
		return &t
	}
	return nil
}

func (p LeftPanel) Update(msg tea.Msg, focused bool) (LeftPanel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !focused {
			break
		}
		switch msg.String() {
		case "tab":
			if p.tab == TabSearch {
				p.tab = TabDownloads
				p.scanLocalFiles()
			} else {
				p.tab = TabSearch
			}
			p.cursor = 0
			p.offset = 0
			p.input.Blur()
			p.input.SetValue("")

		case "enter":
			if p.input.Focused() {
				if p.tab == TabSearch {
					q := strings.TrimSpace(p.input.Value())
					if q == "" || p.loading {
						break
					}
					p.loading = true
					p.errMsg = ""
					p.tracks = nil
					cmds = append(cmds, p.spinner.Tick, searchCmd(p.client, q))
				} else if p.tab == TabDownloads {
					p.input.Blur()
				}
			} else if p.tab == TabSearch && len(p.tracks) > 0 {
				t := p.tracks[p.cursor]
				return p, func() tea.Msg { return PlayStartedMsg{Track: t} }
			} else if p.tab == TabDownloads {
				locals := p.getFilteredLocals()
				if len(locals) > 0 && p.cursor < len(locals) {
					f := locals[p.cursor]
					return p, func() tea.Msg { return PlayLocalMsg{Path: f.Path, Title: f.Name} }
				}
			}

		case "up", "k":
			if p.cursor > 0 {
				p.cursor--
				if p.cursor < p.offset {
					p.offset = p.cursor
				}
			}

		case "down", "j":
			items := p.itemCount()
			if p.cursor < items-1 {
				p.cursor++
				vis := p.visibleRows()
				if p.cursor >= p.offset+vis {
					p.offset++
				}
			}

		case "d":
			if p.tab == TabSearch && len(p.tracks) > 0 && !p.loading {
				t := p.tracks[p.cursor]
				dir, err := getDownloadDir()
				if err != nil {
					p.errMsg = "Cannot find home directory"
					return p, nil
				}
				return p, downloadCmd(p.client, t, dir)
			}

		case "/":
			p.input.Focus()
			p.input.SetValue("")
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
		if msg.Err == nil {
			p.scanLocalFiles()
		}

	case spinner.TickMsg:
		if p.loading {
			var cmd tea.Cmd
			p.spinner, cmd = p.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	var inputCmd tea.Cmd
	p.input, inputCmd = p.input.Update(msg)
	cmds = append(cmds, inputCmd)
	if p.tab == TabDownloads {
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

func (p LeftPanel) View(focused bool) string {
	var b strings.Builder

	// Search Input
	b.WriteString(p.input.View())
	if p.loading {
		b.WriteString(" " + p.spinner.View())
	}
	b.WriteString("\n\n")

	if p.errMsg != "" {
		b.WriteString(StatusErrStyle.Render("X " + p.errMsg))
	} else {
		switch p.tab {
		case TabSearch:
			innerW := p.width - 4
			if innerW < 10 {
				innerW = 10
			}
			b.WriteString(p.renderSearchList(innerW))
		case TabDownloads:
			innerW := p.width - 4
			if innerW < 10 {
				innerW = 10
			}
			b.WriteString(p.renderLocalList(innerW))
		}
	}
	style := PanelStyle.Copy().BorderTop(false)
	borderColor := colorBorder

	var tabSearchRendered, tabDLRendered string
	if p.tab == TabSearch {
		tabSearchRendered = SelectedItemStyle.Render(" Search ")
		tabDLRendered = DimItemStyle.Render(" Downloaded ")
	} else {
		tabSearchRendered = DimItemStyle.Render(" Search ")
		tabDLRendered = LocalFileSelectedStyle.Render(" Downloaded ")
	}

	if focused {
		style = PanelFocusedStyle.Copy().BorderTop(false)
		borderColor = colorBorderF
	}
	contentBox := style.
		Width(p.width - 2).
		Height(p.height - 2).
		Render(b.String())
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	wSearch := lipgloss.Width(tabSearchRendered)
	wDL := lipgloss.Width(tabDLRendered)

	leftTicks := "─"
	middleTicks := "──"
	rightLineCount := p.width - lipgloss.Width(leftTicks) - wSearch - lipgloss.Width(middleTicks) - wDL - 2
	if rightLineCount < 0 {
		rightLineCount = 0
	}

	topBorder := borderStyle.Render("╭"+leftTicks) +
		tabSearchRendered +
		borderStyle.Render(middleTicks) +
		tabDLRendered +
		borderStyle.Render(strings.Repeat("─", rightLineCount)+"╮")

	return lipgloss.JoinVertical(lipgloss.Left, topBorder, contentBox)
}

func (p LeftPanel) renderSearchList(innerW int) string {
	if !p.searched {
		return SubtitleStyle.Render(" Type to search…") + "\n"
	}
	if len(p.tracks) == 0 {
		return DimItemStyle.Render(" No results.") + "\n"
	}
	var b strings.Builder
	vis := p.visibleRows()
	end := p.offset + vis
	if end > len(p.tracks) {
		end = len(p.tracks)
	}
	titleW := innerW - 12
	if titleW < 10 {
		titleW = 10
	}

	for i := p.offset; i < end; i++ {
		t := p.tracks[i]
		mark := "  "
		if i == p.cursor {
			mark = ""
		}
		safeTitle := truncate(t.Title, titleW)
		titleBlock := lipgloss.NewStyle().Width(titleW).Render(safeTitle)
		durationBlock := FormatDuration(t.Duration)

		line := mark + titleBlock + " " + durationBlock

		if i == p.cursor {
			b.WriteString(SelectedItemStyle.Width(innerW).Render(line))
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}
	if len(p.tracks) > vis {
		b.WriteString(DimItemStyle.Render(fmt.Sprintf(" %d/%d", p.cursor+1, len(p.tracks))))
		b.WriteString("\n")
	}
	return b.String()
}

func (p LeftPanel) renderLocalList(innerW int) string {
	locals := p.getFilteredLocals()
	if len(locals) == 0 {
		if strings.TrimSpace(p.input.Value()) != "" {
			return DimItemStyle.Render(" No matching downloaded files found.") + "\n"
		}
		return DimItemStyle.Render(" No downloaded files in ~/Music/SOM_Downloads/") + "\n"
	}
	var b strings.Builder
	vis := p.visibleRows()
	end := p.offset + vis
	if end > len(locals) {
		end = len(locals)
	}
	for i := p.offset; i < end; i++ {
		f := locals[i]
		mark := "  "
		if i == p.cursor {
			mark = ""
		}
		line := mark + truncate(f.Name, innerW-4)
		if i == p.cursor {
			b.WriteString(LocalFileSelectedStyle.Width(innerW).Render(line))
		} else {
			b.WriteString(LocalFileStyle.Render(line))
		}
		b.WriteString("\n")
	}
	if len(locals) > vis {
		b.WriteString(DimItemStyle.Render(fmt.Sprintf(" %d/%d", p.cursor+1, len(locals))))
		b.WriteString("\n")
	}
	return b.String()
}

func (p LeftPanel) renderPanel(content string, focused bool) string {
	style := PanelStyle
	if focused {
		style = PanelFocusedStyle
	}
	return style.
		Width(p.width - 2).
		Height(p.height - 2).
		Render(content)
}

func (p LeftPanel) itemCount() int {
	if p.tab == TabSearch {
		return len(p.tracks)
	}
	return len(p.getFilteredLocals())
}

func (p LeftPanel) visibleRows() int {
	rows := p.height - 10
	if rows < 3 {
		return 3
	}
	return rows
}

// ─── Shared commands ─────────────────────────────────────────────────────────

func searchCmd(c *api.Client, q string) tea.Cmd {
	return func() tea.Msg {
		tracks, err := c.Search(q)
		return SearchResultMsg{Tracks: tracks, Err: err}
	}
}
func getDownloadDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, "Music", "SOM_Downloads"), nil
}

func downloadCmd(c *api.Client, t api.Track, destDir string) tea.Cmd {
	return func() tea.Msg {
		path, err := c.DownloadM4A(t.ID, t.Title, destDir)

		if err == nil {
			lr, errLyr := c.Lyrics(t.ID, t.Title, t.Artist, t.Duration)
			if errLyr == nil {
				jsonPath := strings.TrimSuffix(path, ".m4a") + ".json"
				data, _ := json.MarshalIndent(lr, "", "  ")
				_ = os.WriteFile(jsonPath, data, 0644)
			}
		}
		return DownloadDoneMsg{Path: path, Err: err}
	}
}

func (p LeftPanel) getFilteredLocals() []LocalFile {
	if p.tab != TabDownloads {
		return p.locals
	}
	q := strings.ToLower(strings.TrimSpace(p.input.Value()))
	if q == "" {
		return p.locals
	}
	var filtered []LocalFile
	for _, f := range p.locals {
		if strings.Contains(strings.ToLower(f.Name), q) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}
