package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
				Name:     name,
				Path:     localPath,
				Artist:   artist,
				Duration: getFileDuration(localPath),
			})
		}
	}
}

func getFileDuration(path string) int {
	cmd := exec.Command("ffprobe", "-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0
	}
	sec, err := strconv.ParseFloat(strings.TrimSpace(out.String()), 64)
	if err != nil {
		return 0
	}
	return int(sec)
}

func (p LeftPanel) ViewDownloadsContent(w, h int) string {
	innerW := w - 4

	inputFocused := p.input.Focused()
	searchBorder := lipgloss.Color("#7c7986")
	contentBorder := lipgloss.Color("#7c7986")
	if inputFocused {
		searchBorder = lipgloss.Color("#e8593c")
	} else {
		contentBorder = lipgloss.Color("#e8593c")
	}

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

	searchBox := renderBox(w, "Search", searchContent.String(), searchBorder)

	// ─── Playlist box ─────────────────────────
	count := len(p.getFilteredLocals())
	listContent := p.renderLocalList(innerW)
	playlistBox := renderBox(w, fmt.Sprintf("Playlist (%d)", count), listContent, contentBorder)

	return searchBox + "\n" + playlistBox
}

func (p LeftPanel) renderLocalList(innerW int) string {
	locals := p.getFilteredLocals()
	if len(locals) == 0 {
		if p.input.Focused() && strings.TrimSpace(p.input.Value()) != "" {
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

	idxW := 3
	if len(locals) >= 1000 {
		idxW = 4
	}
	artistW := 27
	titleW := innerW - idxW - artistW - 6
	if titleW < 10 {
		titleW = 10
		artistW = innerW - idxW - titleW - 6
		if artistW < 0 {
			artistW = 0
		}
	}

	header := fmt.Sprintf("  %*s  %-*s  %s", idxW, "#", titleW, "Title", "Artist")
	b.WriteString(DimItemStyle.Width(innerW).Render(header))
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
