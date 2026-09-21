package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"som/internal/domain"
	"som/internal/storage"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-runewidth"
)

var cmdOptions = []string{
	"Add to queue",
	"Audio settings",
	"Playback speed",
	"Sort",
	"Rename title",
	"Delete track",
	"Move to playlist",
	"Show file info",
}

var audioPresets = []struct{ Name, Filter, Desc string }{
	{"Normal", "dynaudnorm=f=250:g=11:p=0.9:m=10", "Normalize volume, preserve original quality"},
	{"Bass Boost", "dynaudnorm=f=250:g=11:p=0.9:m=10,bass=g=8:f=100:w=0.5", "Significantly boost the sub-bass range."},
	{"Nightcore", "dynaudnorm=f=250:g=11:p=0.9:m=10,asetrate=48000*1.2,aresample=44100", "Fast (1.25x), high-pitched voice (pitch up)"},
	{"Daycore", "dynaudnorm=f=250:g=11:p=0.9:m=10,asetrate=48000*0.85,aresample=48000,aecho=0.8:0.88:60:0.4", "Slow (0.85x), deep and muffled + Reverb"},
	{"Lo-Fi", "dynaudnorm=f=250:g=11:p=0.9:m=10,lowpass=f=800,volume=1.2", "High-cut filter, old radio/muffled effect"},
}

var playbackSpeeds = []struct {
	Label string
	Value float64
}{
	{"0.25x", 0.25}, {"0.5x", 0.5}, {"0.75x", 0.75},
	{"1.0x", 1.0}, {"1.25x", 1.25}, {"1.5x", 1.5},
	{"1.75x", 1.75}, {"2.0x", 2.0},
}

var sortOptions = []struct {
	Key  string
	Name string
}{
	{"name", "Name"},
	{"date", "Date downloaded"},
	{"duration", "Duration"},
}

func (a *App) cmdOptionList() []string {
	if a.sidebarActive == SidePlaylists {
		return []string{
			"Audio settings",
			"Playback speed",
			"Show file info",
			"Remove from playlist",
		}
	}

	opts := cmdOptions
	if _, ok := a.selectedTrackForPlaylist(); ok && len(a.playlistsContainingSelected()) > 0 {
		opts = append(append([]string{}, cmdOptions...), "Remove from playlist")
	}
	return opts
}

func (a *App) playlistsContainingSelected() []int {
	track, ok := a.selectedTrackForPlaylist()
	if !ok {
		return nil
	}
	var idxs []int
	for i, pl := range a.left.playlists {
		for _, t := range pl.Tracks {
			if t.Path == track.Path || "local:"+t.Path == track.ID || t.ID == track.ID {
				idxs = append(idxs, i)
				break
			}
		}
	}
	return idxs
}
func (a *App) updateCmdPopup(k tea.KeyMsg) tea.Cmd {
	if a.plRmActive {
		idxs := a.playlistsContainingSelected()
		switch k.String() {
		case "up", "k":
			if a.cmdCursor > 0 {
				a.cmdCursor--
			} else {
				a.cmdCursor = len(idxs) - 1
			}
		case "down", "j":
			if a.cmdCursor < len(idxs)-1 {
				a.cmdCursor++
			} else {
				a.cmdCursor = 0
			}
		case "enter":
			if a.cmdCursor < len(idxs) && a.left.plStore != nil {
				pl := a.left.playlists[idxs[a.cmdCursor]]
				track, _ := a.selectedTrackForPlaylist()
				if err := a.left.plStore.RemoveTrackFromPlaylist(pl.ID, track.Path); err == nil {
					for j := range pl.Tracks {
						if pl.Tracks[j].Path == track.Path || pl.Tracks[j].ID == track.ID {
							a.left.playlists[idxs[a.cmdCursor]].Tracks = append(pl.Tracks[:j], pl.Tracks[j+1:]...)
							break
						}
					}
					a.setStatus(StatusOKStyle.Render("> Removed from \"" + pl.Name + "\""))
				} else {
					a.setStatus(StatusErrStyle.Render("X Failed: " + err.Error()))
				}
				a.plRmActive = false
				return nil
			}
		case "esc", ":":
			a.plRmActive = false
			return nil
		}
		return nil
	}
	if a.presetActive {
		switch k.String() {
		case "up", "k":
			if a.cmdCursor > 0 {
				a.cmdCursor--
			} else {
				a.cmdCursor = len(audioPresets) - 1
			}
		case "down", "j":
			if a.cmdCursor < len(audioPresets)-1 {
				a.cmdCursor++
			} else {
				a.cmdCursor = 0
			}
		case "enter":
			p := audioPresets[a.cmdCursor]
			a.player.SetAudioFilter(p.Filter)
			a.activePreset = a.cmdCursor
			a.presetActive = false
			a.showCmdPopup = false
			a.setStatus(StatusOKStyle.Render("Applied: " + p.Name))

			// Khởi động lại ffmpeg để ép ăn filter ngay lập tức
			if a.playback.NowPlay != nil {
				pos := int(a.player.Position().Seconds())
				a.player.SeekTo(float64(pos))
			}
			return nil
		case "esc", ":":
			a.presetActive = false
			return nil
		}
		return nil
	}

	if a.sortActive {
		switch k.String() {
		case "up", "k":
			if a.cmdCursor > 0 {
				a.cmdCursor--
			} else {
				a.cmdCursor = len(sortOptions) - 1
			}
		case "down", "j":
			if a.cmdCursor < len(sortOptions)-1 {
				a.cmdCursor++
			} else {
				a.cmdCursor = 0
			}
		case "enter":
			chosen := sortOptions[a.cmdCursor]
			a.left.sortPref = chosen.Key
			if a.left.plStore != nil {
				a.left.plStore.SetSetting("sort_downloads", chosen.Key)
			}
			a.left.scanLocalFiles()
			a.sortActive = false
			a.showCmdPopup = false
			a.setStatus(StatusOKStyle.Render("> Sorted by " + chosen.Name))
			return nil
		case "esc", ":":
			a.sortActive = false
			return nil
		}
		return nil
	}
	if a.speedActive {
		switch k.String() {
		case "up", "k":
			if a.cmdCursor > 0 {
				a.cmdCursor--
			} else {
				a.cmdCursor = len(playbackSpeeds) - 1
			}
		case "down", "j":
			if a.cmdCursor < len(playbackSpeeds)-1 {
				a.cmdCursor++
			} else {
				a.cmdCursor = 0
			}
		case "enter":
			pos := 0.0
			if a.playback.NowPlay != nil {
				pos = a.player.Position().Seconds()
			}

			s := playbackSpeeds[a.cmdCursor]
			a.player.SetSpeed(s.Value)
			a.activeSpeed = a.cmdCursor
			a.speedActive = false
			a.showCmdPopup = false
			a.setStatus(StatusOKStyle.Render("Speed set to: " + s.Label))

			if a.playback.NowPlay != nil {
				a.player.SeekTo(pos)
			}
			return nil
		case "esc", ":", "q":
			a.speedActive = false
			return nil
		}
		return nil
	}

	switch k.String() {
	case "esc", ":":
		a.showCmdPopup = false
	case "up", "k":
		if a.cmdMenuCursor > 0 {
			a.cmdMenuCursor--
		} else {
			a.cmdMenuCursor = len(a.cmdOptionList()) - 1
		}
	case "down", "j":
		if a.cmdMenuCursor < len(a.cmdOptionList())-1 {
			a.cmdMenuCursor++
		} else {
			a.cmdMenuCursor = 0
		}
	case "enter":
		return a.runCmdOption(a.cmdMenuCursor)
	}
	return nil
}

func (a *App) runCmdOption(idx int) tea.Cmd {
	opts := a.cmdOptionList()
	if idx < 0 || idx >= len(opts) {
		return nil
	}
	switch opts[idx] {
	case "Audio settings":
		a.presetActive = true
		a.cmdCursor = a.activePreset
	case "Sort":
		a.sortActive = true
		// Find current sort index
		for i, s := range sortOptions {
			if s.Key == a.left.sortPref {
				a.cmdCursor = i
				break
			}
		}
	case "Add to queue":
		track, ok := a.selectedTrackForPlaylist()
		if ok {
			a.playback.Queue = append(a.playback.Queue, domain.Track{
				ID:       track.ID,
				Title:    track.Title,
				Artist:   track.Artist,
				Duration: track.Duration,
			})
			a.setStatus(StatusOKStyle.Render(fmt.Sprintf("> Queued: %s", track.Title)))
		} else {
			a.setStatus(StatusErrStyle.Render("X No track selected"))
		}
		a.showCmdPopup = false
	case "Rename title":
		target, ok := a.renameTarget()
		if !ok {
			a.setStatus(StatusErrStyle.Render("X No local track selected"))
			return nil
		}

		// Tắt menu commands
		a.showCmdPopup = false

		// Khởi tạo Modal mới và gán vào activeModal
		modal := NewRenameModal(target, a.left.plStore, a.width)
		a.activeModal = modal
		return modal.Init()

	case "Playback speed":
		a.speedActive = true
		a.cmdCursor = a.activeSpeed
	case "Delete track":
		if target, ok := a.renameTarget(); ok {
			a.showCmdPopup = false
			modal := NewDeleteModal(target, a.left.plStore)
			a.activeModal = modal
			return modal.Init()
		} else {
			a.setStatus(StatusErrStyle.Render("X No local track selected"))
		}
	case "Show file info":
		if target, ok := a.renameTarget(); ok {
			a.showCmdPopup = false
			modal := NewInfoModal(target)
			a.activeModal = modal
			return modal.Init()
		} else {
			a.setStatus(StatusErrStyle.Render("X No local track selected"))
		}
	case "Move to playlist":
		a.showCmdPopup = false
		return a.startMoveToPlaylist()
	case "Remove from playlist":
		a.plRmActive = true
		a.cmdCursor = 0
	}
	return nil
}

func (a *App) selectedTrackForPlaylist() (storage.PlaylistTrack, bool) {
	if a.sidebarActive == SideSearch && a.left.searchCursor < len(a.left.tracks) {
		t := a.left.tracks[a.left.searchCursor]
		return storage.PlaylistTrack{ID: t.ID, Title: t.Title, Artist: t.Artist, Duration: t.Duration}, true
	}
	if lf, ok := a.renameTarget(); ok {
		return storage.PlaylistTrack{ID: "local:" + lf.Path, Title: lf.Name, Artist: lf.Artist, Duration: lf.Duration}, true
	}
	return storage.PlaylistTrack{}, false
}

func (a *App) localPathTaken(path string) bool {
	if a.left.plStore != nil {
		if lf, err := a.left.plStore.GetLocalFile(path); err == nil && lf != nil {
			return true
		}
	}
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func sanitizeLocalName(s string) string {
	r := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", `"`, "-", "<", "-", ">", "-", "|", "-")
	return strings.TrimSpace(r.Replace(s))
}

func formatBytes(n int64) string {
	const unit = 1000
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "kMGTPE"[exp])
}

func formatDBTime(s string) string {
	if s == "" {
		return "-"
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("15:04:05 02-01-2006")
		}
	}
	return s
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
	if a.playback.NowPlay != nil && strings.HasPrefix(a.playback.NowPlay.ID, "local:") {
		path := strings.TrimPrefix(a.playback.NowPlay.ID, "local:")
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
	if a.plRmActive {
		track, _ := a.selectedTrackForPlaylist()
		idxs := a.playlistsContainingSelected()
		b.WriteString("\n ")
		b.WriteString(NormalItemStyle.Render("Remove \"" + track.Title + "\" from:"))
		b.WriteString("\n\n ")
		for i, plIdx := range idxs {
			marker := "  "
			if i == a.cmdCursor {
				marker = "▸ "
			}
			line := marker + a.left.playlists[plIdx].Name
			if i == a.cmdCursor {
				pad := 51 - runewidth.StringWidth(line)
				if pad < 0 {
					pad = 0
				}
				b.WriteString(SelectedItemStyle.Render(line + strings.Repeat(" ", pad)))
			} else {
				b.WriteString(NormalItemStyle.Render(line))
			}
			b.WriteString("\n ")
		}
		b.WriteString("\n ")
		b.WriteString(DimItemStyle.Render(" (enter: remove  | esc: back)"))
		return renderBox(56, "Remove from Playlist", b.String(), themeCol("#e8593c"))
	}
	if a.speedActive {
		const boxW = 35
		const innerW = boxW - 4
		var b strings.Builder
		b.WriteString("\n")
		for i, s := range playbackSpeeds {
			cursor := "   "
			tick := " "
			if i == a.activeSpeed {
				tick = "+"
			}
			namePart := fmt.Sprintf(" %s [%s] %s", cursor, tick, s.Label)
			if i == a.cmdCursor {
				pad := innerW - runewidth.StringWidth(namePart)
				if pad < 0 {
					pad = 0
				}
				b.WriteString(SelectedItemStyle.Render(namePart+strings.Repeat(" ", pad)) + "\n")
			} else {
				b.WriteString(NormalItemStyle.Render(namePart) + "\n")
			}
		}
		b.WriteString("\n")
		b.WriteString(DimItemStyle.Render(" (enter: apply  | esc: back)"))
		return renderBox(boxW, "Playback speed", b.String(), themeCol("#e8593c"))
	}
	if a.sortActive {
		const boxW = 35
		const innerW = boxW - 4
		var b strings.Builder
		b.WriteString("\n")
		for i, s := range sortOptions {
			tick := " "
			if s.Key == a.left.sortPref {
				tick = "+"
			}
			namePart := fmt.Sprintf("   [%s] %s", tick, s.Name)
			if i == a.cmdCursor {
				pad := innerW - runewidth.StringWidth(namePart)
				if pad < 0 {
					pad = 0
				}
				b.WriteString(SelectedItemStyle.Render(namePart+strings.Repeat(" ", pad)) + "\n")
			} else {
				b.WriteString(NormalItemStyle.Render(namePart) + "\n")
			}
		}
		b.WriteString("\n")
		b.WriteString(DimItemStyle.Render(" (enter: apply  | esc: back)"))
		return renderBox(boxW, "Sort by", b.String(), themeCol("#e8593c"))
	}
	if a.presetActive {
		const boxW = 55
		const innerW = boxW - 4
		b.WriteString("\n")
		for i, p := range audioPresets {
			cursor := " "
			tick := " "
			if i == a.activePreset {
				tick = "+"
			}

			namePart := fmt.Sprintf(" %s [%s] %s", cursor, tick, p.Name)

			if i == a.cmdCursor {
				pad := innerW - runewidth.StringWidth(namePart)
				if pad < 0 {
					pad = 0
				}
				b.WriteString(SelectedItemStyle.Render(namePart+strings.Repeat(" ", pad)) + "\n")
			} else {
				b.WriteString(NormalItemStyle.Render(namePart) + "\n")
			}
		}

		b.WriteString("\n")
		activeDesc := audioPresets[a.cmdCursor].Desc
		b.WriteString(DimItemStyle.Render("  "+activeDesc) + "\n")

		b.WriteString("\n")
		b.WriteString(DimItemStyle.Render(" (enter: apply  | esc: back)"))

		return renderBox(boxW, "Audio settings", b.String(), themeCol("#e8593c"))
	}

	b.WriteString("\n")
	for i, opt := range a.cmdOptionList() {
		line := "  " + opt
		if i == a.cmdMenuCursor {
			pad := 36 - runewidth.StringWidth(line)
			if pad < 0 {
				pad = 0
			}
			b.WriteString(SelectedItemStyle.Render(line + strings.Repeat(" ", pad)))
		} else {
			b.WriteString(NormalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(DimItemStyle.Render(" (enter: select  | esc: close)"))
	return renderBox(40, "Commands", b.String(), themeCol("#e8593c"))
}

type RenameDoneMsg struct {
	OldPath  string
	NewPath  string
	NewTitle string
	Err      error
}

type DeleteDoneMsg struct {
	Path string
	Name string
	Err  error
}

func renameCmd(plStore *storage.DB, oldPath, newPath, newTitle string) tea.Cmd {
	return func() tea.Msg {
		if newPath != oldPath {
			if _, err := os.Stat(newPath); err == nil {
				return RenameDoneMsg{Err: fmt.Errorf("file already exists: %s", filepath.Base(newPath))}
			}
			if plStore != nil {
				if lf, err := plStore.GetLocalFile(newPath); err == nil && lf != nil {
					return RenameDoneMsg{Err: fmt.Errorf("path taken in DB: %s", filepath.Base(newPath))}
				}
			}
			if err := os.Rename(oldPath, newPath); err != nil {
				return RenameDoneMsg{Err: fmt.Errorf("rename file: %w", err)}
			}
		}

		oldJson := localFileSidecar(oldPath)
		newJson := localFileSidecar(newPath)
		if data, err := os.ReadFile(oldJson); err == nil {
			var meta map[string]any
			if json.Unmarshal(data, &meta) == nil {
				meta["title"] = newTitle
				if out, err := json.MarshalIndent(meta, "", "  "); err == nil {
					if newPath != oldPath && newJson != oldJson {
						_ = os.Rename(oldJson, newJson)
					}
					_ = os.WriteFile(newJson, out, 0o644)
				}
			}
		}

		if plStore != nil {
			if err := plStore.RenameLocalFile(oldPath, newPath, newTitle); err != nil {
				return RenameDoneMsg{Err: fmt.Errorf("db update: %w", err)}
			}
		}

		return RenameDoneMsg{OldPath: oldPath, NewPath: newPath, NewTitle: newTitle}
	}
}

func deleteCmd(plStore *storage.DB, path, name string) tea.Cmd {
	return func() tea.Msg {
		os.Remove(path)
		os.Remove(localFileSidecar(path))
		if plStore != nil {
			if err := plStore.DeleteLocalFile(path); err != nil {
				return DeleteDoneMsg{Err: fmt.Errorf("db delete: %w", err)}
			}
		}

		return DeleteDoneMsg{Path: path, Name: name}
	}
}
