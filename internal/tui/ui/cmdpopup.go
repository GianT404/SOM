package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var cmdOptions = []string{
	"Rename title",
}

func (a *App) updateCmdPopup(k tea.KeyMsg) tea.Cmd {
	if a.renameActive {
		switch k.String() {
		case "enter":
			a.applyRenameTitle(strings.TrimSpace(a.renameInput.Value()))
			a.renameActive = false
			a.renameInput.Blur()
			a.renameInput.SetValue("")
			a.showCmdPopup = false
			return nil
		case "esc":
			a.renameActive = false
			a.renameInput.Blur()
			a.renameInput.SetValue("")
			return nil
		}
		var cmd tea.Cmd
		a.renameInput, cmd = a.renameInput.Update(k)
		return cmd
	}

	switch k.String() {
	case "esc", ":":
		a.showCmdPopup = false
	case "up", "k":
		if a.cmdCursor > 0 {
			a.cmdCursor--
		}
	case "down", "j":
		if a.cmdCursor < len(cmdOptions)-1 {
			a.cmdCursor++
		}
	case "enter":
		a.runCmdOption(a.cmdCursor)
	}
	return nil
}

func (a *App) runCmdOption(idx int) {
	if idx < 0 || idx >= len(cmdOptions) {
		return
	}
	switch cmdOptions[idx] {
	case "Rename title":
		a.renameActive = true
		iw := 60
		if a.width > 0 && a.width-12 < iw {
			iw = a.width - 12
		}
		if iw < 20 {
			iw = 20
		}
		a.renameInput.Width = iw
		a.renameInput.Focus()
		if target, ok := a.renameTarget(); ok {
			a.renameInput.SetValue(target.Name)
		} else {
			a.renameInput.SetValue("")
		}
		a.renameInput.CursorEnd()
	}
}

func (a *App) applyRenameTitle(newTitle string) {
	if newTitle == "" {
		a.setStatus(StatusErrStyle.Render("X Title cannot be empty"))
		return
	}
	target, ok := a.renameTarget()
	if !ok {
		a.setStatus(StatusErrStyle.Render("X No local track selected"))
		return
	}
	oldPath := target.Path

	newBase := sanitizeLocalName(newTitle)
	if newBase == "" {
		newBase = target.VideoID
	}
	newPath := filepath.Join(filepath.Dir(oldPath), newBase+".opus")
	if newPath != oldPath {
		if err := os.Rename(oldPath, newPath); err != nil {
			a.setStatus(StatusErrStyle.Render("X Rename file failed: " + err.Error()))
			return
		}
	}

	oldJson := strings.TrimSuffix(oldPath, ".opus") + ".json"
	newJson := strings.TrimSuffix(newPath, ".opus") + ".json"
	if data, err := os.ReadFile(oldJson); err == nil {
		var meta map[string]any
		if json.Unmarshal(data, &meta) == nil {
			meta["title"] = newTitle
			if out, err := json.MarshalIndent(meta, "", "  "); err == nil {
				if newJson != oldJson {
					_ = os.Rename(oldJson, newJson)
				}
				_ = os.WriteFile(newJson, out, 0o644)
			}
		}
	}

	target.Name = newTitle
	target.Path = newPath

	for i := range a.playlist {
		if a.playlist[i].ID == "local:"+oldPath {
			a.playlist[i].ID = "local:" + newPath
			a.playlist[i].Title = newTitle
		}
	}
	if a.nowPlay != nil && strings.HasPrefix(a.nowPlay.ID, "local:") && strings.TrimPrefix(a.nowPlay.ID, "local:") == oldPath {
		a.nowPlay.ID = "local:" + newPath
		a.nowPlay.Title = newTitle
	}
	a.setStatus(StatusOKStyle.Render("> Renamed to " + newTitle))
}

func sanitizeLocalName(s string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", `"`, "-", "<", "-", ">", "-", "|", "-")
	return strings.TrimSpace(r.Replace(s))
}

func (a *App) renameTarget() (*LocalFile, bool) {
	if a.sidebarActive == SideDownloads {
		q := strings.ToLower(strings.TrimSpace(a.left.input.Value()))
		tokens := strings.Fields(q)
		idx := 0
		for i := range a.left.locals {
			f := a.left.locals[i]
			if len(tokens) > 0 && !localMatches(f, tokens) {
				continue
			}
			if idx == a.left.dlCursor {
				return &a.left.locals[i], true
			}
			idx++
		}
	}
	if a.nowPlay != nil && strings.HasPrefix(a.nowPlay.ID, "local:") {
		path := strings.TrimPrefix(a.nowPlay.ID, "local:")
		for i := range a.left.locals {
			if a.left.locals[i].Path == path {
				return &a.left.locals[i], true
			}
		}
	}
	return nil, false
}

func (a *App) renderCmdPopup() string {
	var b strings.Builder
	if a.renameActive {
		b.WriteString("\n  ")
		b.WriteString(a.renameInput.View())
		b.WriteString("\n\n")
		b.WriteString(DimItemStyle.Render(" (enter: rename  | esc: back)"))
		w := a.renameInput.Width + 8
		if w < 48 {
			w = 48
		}
		if a.width > 0 && w > a.width-2 {
			w = a.width - 2
		}
		return renderBox(w, "Rename Title", b.String(), lipgloss.Color("#e8593c"))
	}

	b.WriteString("\n\n")
	for i, opt := range cmdOptions {
		marker := "  "
		if i == a.cmdCursor {
			marker = "▸ "
		}
		line := marker + opt
		if i == a.cmdCursor {
			b.WriteString(SelectedItemStyle.Render(line))
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(DimItemStyle.Render(" (enter: select  | esc: close)"))
	return renderBox(40, "Commands", b.String(), lipgloss.Color("#e8593c"))
}
