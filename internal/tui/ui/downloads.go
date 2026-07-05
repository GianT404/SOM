package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
			localPath := filepath.Join(dir, e.Name())
			name := strings.TrimSuffix(e.Name(), ".m4a")
			artist := ""
			jsonPath := strings.TrimSuffix(localPath, ".m4a") + ".json"
			if data, err := os.ReadFile(jsonPath); err == nil {
				var meta struct {
					Artist string `json:"artist"`
					Title  string `json:"title"`
				}
				if json.Unmarshal(data, &meta) == nil {
					artist = meta.Artist
					if meta.Title != "" {
						name = meta.Title
					}
				}
			}
			p.locals = append(p.locals, LocalFile{
				Name:   name,
				Path:   localPath,
				Artist: artist,
			})
		}
	}
}

func (p LeftPanel) ViewDownloadsContent(w, h int) string {
	borderColor := lipgloss.Color("#7c7986")
	innerW := w - 4

	// ─── Search box ───────────────────────────
	var searchContent strings.Builder
	inputRow := " " + p.input.View()
	if p.loading {
		inputRow += " " + p.spinner.View()
	}
	searchContent.WriteString(lipgloss.NewStyle().Width(innerW).Render(inputRow))

	if p.errMsg != "" {
		searchContent.WriteString(StatusErrStyle.Render("X " + p.errMsg))
	}

	searchBox := renderBox(w, "Search", searchContent.String(), borderColor)

	// ─── Playlist box ─────────────────────────
	count := len(p.getFilteredLocals())
	listContent := p.renderLocalList(innerW)
	playlistBox := renderBox(w, fmt.Sprintf("Playlist (%d)", count), listContent, borderColor)

	return searchBox + "\n" + playlistBox
}

func (p LeftPanel) renderLocalList(innerW int) string {
	locals := p.getFilteredLocals()
	if len(locals) == 0 {
		if p.input.Focused() && strings.TrimSpace(p.input.Value()) != "" {
			return DimItemStyle.Render(" No matching downloaded files found.")
		}
		return DimItemStyle.Render(" No downloaded files in ~/Music/SOM_Downloads/")
	}
	var b strings.Builder
	vis := p.visibleRows()
	end := p.offset + vis
	if end > len(locals) {
		end = len(locals)
	}

	idxW := 3
	if len(locals) >= 1000 {
		idxW = 4
	}
	artistW := 20
	titleW := innerW - idxW - artistW - 6
	if titleW < 10 {
		titleW = 10
		artistW = innerW - idxW - titleW - 6
		if artistW < 0 {
			artistW = 0
		}
	}

	header := fmt.Sprintf("  %*s  %-*s  %s", idxW, "#", titleW, "Title", "Artist")
	b.WriteString(LocalFileStyle.Width(innerW).Render(header))
	b.WriteString("\n")

	for i := p.offset; i < end; i++ {
		f := locals[i]
		mark := "  "
		if i == p.cursor {
			mark = " "
		}
		idx := fmt.Sprintf("%*d", idxW, i+1)
		title := truncate(f.Name, titleW)
		artist := truncate(f.Artist, artistW)
		line := mark + idx + "  " + fmt.Sprintf("%-*s", titleW, title) + "  " + fmt.Sprintf("%-*s", artistW, artist)

		if i == p.cursor {
			b.WriteString(LocalFileSelectedStyle.Width(innerW).Render(line))
		} else {
			b.WriteString(LocalFileStyle.Width(innerW).Render(line))
		}
		b.WriteString("\n")
	}
	if len(locals) > vis {
		b.WriteString(DimItemStyle.Render(fmt.Sprintf(" %d/%d", p.cursor+1, len(locals))))
		b.WriteString("\n")
	}
	return b.String()
}
