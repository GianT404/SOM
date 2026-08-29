package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"som/internal/domain"
	"som/internal/storage"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
)

var cmdOptions = []string{
	"Add to queue",
	"Audio settings",
	"Playback speed",
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

func (a *App) cmdOptionList() []string {
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
			if t.ID == track.ID {
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
			}
		case "down", "j":
			if a.cmdCursor < len(idxs)-1 {
				a.cmdCursor++
			}
		case "enter":
			if a.cmdCursor < len(idxs) && a.left.plStore != nil {
				pl := a.left.playlists[idxs[a.cmdCursor]]
				track, _ := a.selectedTrackForPlaylist()
				if err := a.left.plStore.RemoveTrackFromPlaylist(pl.ID, track.ID); err == nil {
					for j := range pl.Tracks {
						if pl.Tracks[j].ID == track.ID {
							a.left.playlists[idxs[a.cmdCursor]].Tracks = append(pl.Tracks[:j], pl.Tracks[j+1:]...)
							break
						}
					}
					a.setStatus(StatusOKStyle.Render("> Removed from \"" + pl.Name + "\""))
				}
			}
			a.plRmActive = false
			return nil
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
			}
		case "down", "j":
			if a.cmdCursor < len(audioPresets)-1 {
				a.cmdCursor++
			}
		case "enter":
			p := audioPresets[a.cmdCursor]
			a.player.SetAudioFilter(p.Filter)
			a.activePreset = a.cmdCursor
			a.presetActive = false
			a.showCmdPopup = false
			a.setStatus(StatusOKStyle.Render("Applied: " + p.Name))

			// Khởi động lại ffmpeg để ép ăn filter ngay lập tức
			if a.nowPlay != nil {
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
	if a.plMoveActive {
		switch k.String() {
		case "up", "k":
			if a.cmdCursor > 0 {
				a.cmdCursor--
			}
		case "down", "j":
			if a.cmdCursor < len(a.left.playlists)-1 {
				a.cmdCursor++
			}
		case "enter":
			track, ok := a.selectedTrackForPlaylist()
			if ok && a.left.plStore != nil && a.cmdCursor < len(a.left.playlists) {
				pl := a.left.playlists[a.cmdCursor]
				if err := a.left.plStore.AddTrackToPlaylist(pl.ID, storage.PlaylistTrack{
					ID:       track.ID,
					Title:    track.Title,
					Artist:   track.Artist,
					Duration: track.Duration,
					IsLocal:  track.IsLocal,
				}); err == nil {
					a.left.playlists[a.cmdCursor].Tracks = append(a.left.playlists[a.cmdCursor].Tracks, storage.PlaylistTrack{
						ID:       track.ID,
						Title:    track.Title,
						Artist:   track.Artist,
						Duration: track.Duration,
						IsLocal:  track.IsLocal,
					})
					a.setStatus(StatusOKStyle.Render("> Added to \"" + pl.Name + "\""))
				}
			}
			a.plMoveActive = false
			return nil
		case "esc", ":":
			a.plMoveActive = false
			return nil
		}
		return nil
	}

	if a.infoActive {
		switch k.String() {
		case "esc", "enter", ":", "q":
			a.infoActive = false
			return nil
		}
		return nil
	}

	if a.delActive {
		switch k.String() {
		case "up", "k", "down", "j", "left", "h", "right", "l":
			a.cmdCursor = 1 - a.cmdCursor
		case "enter":
			if a.cmdCursor == 1 {
				a.applyDeleteTrack()
				a.delActive = false
				a.showCmdPopup = false
			} else {
				a.delActive = false
			}
			return nil
		case "esc", ":":
			a.delActive = false
			return nil
		}
		return nil
	}

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

	if a.speedActive {
		switch k.String() {
		case "up", "k":
			if a.cmdCursor > 0 {
				a.cmdCursor--
			}
		case "down", "j":
			if a.cmdCursor < len(playbackSpeeds)-1 {
				a.cmdCursor++
			}
		case "enter":
			pos := 0.0
			if a.nowPlay != nil {
				pos = a.player.Position().Seconds()
			}

			s := playbackSpeeds[a.cmdCursor]
			a.player.SetSpeed(s.Value)
			a.activeSpeed = a.cmdCursor
			a.speedActive = false
			a.showCmdPopup = false
			a.setStatus(StatusOKStyle.Render("Speed set to: " + s.Label))

			if a.nowPlay != nil {
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
		}
	case "down", "j":
		if a.cmdMenuCursor < len(a.cmdOptionList())-1 {
			a.cmdMenuCursor++
		}
	case "enter":
		a.runCmdOption(a.cmdMenuCursor)
	}
	return nil
}

func (a *App) runCmdOption(idx int) {
	opts := a.cmdOptionList()
	if idx < 0 || idx >= len(opts) {
		return
	}
	switch opts[idx] {
	case "Audio settings":
		a.presetActive = true
		a.cmdCursor = a.activePreset
	case "Add to queue":
		track, ok := a.selectedTrackForPlaylist()
		if ok {
			a.trackQueue = append(a.trackQueue, domain.Track{
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
		a.renameActive = true
		iw := 60
		if a.width > 0 && a.width-12 < iw {
			iw = a.width - 12
		}
		if iw < 20 {
			iw = 20
		}
		a.renameInput.SetWidth(iw)
		a.renameInput.Focus()
		if target, ok := a.renameTarget(); ok {
			a.renameInput.SetValue(target.Name)
		} else {
			a.renameInput.SetValue("")
		}
		a.renameInput.CursorEnd()

	case "Playback speed":
		a.speedActive = true
		a.cmdCursor = a.activeSpeed
	case "Delete track":
		if _, ok := a.renameTarget(); ok {
			a.delActive = true
			a.cmdCursor = 0
		} else {
			a.setStatus(StatusErrStyle.Render("X No local track selected"))
		}
	case "Show file info":
		if _, ok := a.renameTarget(); ok {
			a.infoActive = true
		} else {
			a.setStatus(StatusErrStyle.Render("X No local track selected"))
		}
	case "Move to playlist":
		if len(a.left.playlists) == 0 {
			a.setStatus(StatusErrStyle.Render("X No playlists available. Press '/' to create one."))
		} else {
			a.plMoveActive = true
			a.cmdCursor = 0
		}
	case "Remove from playlist":
		a.plRmActive = true
		a.cmdCursor = 0
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
	newPath := filepath.Join(filepath.Dir(oldPath), newBase+filepath.Ext(oldPath))
	if newPath != oldPath {
		if err := os.Rename(oldPath, newPath); err != nil {
			a.setStatus(StatusErrStyle.Render("X Rename file failed: " + err.Error()))
			return
		}
	}

	oldJson := localFileSidecar(oldPath)
	newJson := localFileSidecar(newPath)
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

	// Update SQLite records.
	if a.left.plStore != nil {
		_ = a.left.plStore.RenameLocalFile(oldPath, newPath, newTitle)
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

func (a *App) applyDeleteTrack() {
	target, ok := a.renameTarget()
	if !ok {
		a.setStatus(StatusErrStyle.Render("X No local track selected"))
		return
	}
	path := target.Path

	if a.nowPlay != nil && strings.HasPrefix(a.nowPlay.ID, "local:") && strings.TrimPrefix(a.nowPlay.ID, "local:") == path {
		a.player.Stop()
		a.nowPlay = nil
		a.songStarted = false
	}

	os.Remove(path)
	os.Remove(localFileSidecar(path))

	// Remove from SQLite.
	if a.left.plStore != nil {
		_ = a.left.plStore.DeleteLocalFile(path)
	}

	for i := range a.left.locals {
		if a.left.locals[i].Path == path {
			a.left.locals = append(a.left.locals[:i], a.left.locals[i+1:]...)
			break
		}
	}
	if a.left.dlCursor >= len(a.left.locals) {
		a.left.dlCursor = len(a.left.locals) - 1
	}

	newPlaylist := a.playlist[:0]
	for _, t := range a.playlist {
		if t.ID != "local:"+path {
			newPlaylist = append(newPlaylist, t)
		}
	}
	a.playlist = newPlaylist

	a.setStatus(StatusOKStyle.Render("> Deleted " + target.Name))
}

func (a *App) selectedTrackForPlaylist() (storage.PlaylistTrack, bool) {
	if a.sidebarActive == SideSearch && a.left.searchCursor < len(a.left.tracks) {
		t := a.left.tracks[a.left.searchCursor]
		return storage.PlaylistTrack{ID: t.ID, Title: t.Title, Artist: t.Artist, Duration: t.Duration, IsLocal: a.left.isDownloaded(t)}, true
	}
	if lf, ok := a.renameTarget(); ok {
		return storage.PlaylistTrack{ID: "local:" + lf.Path, Title: lf.Name, Artist: lf.Artist, Duration: lf.Duration, IsLocal: true}, true
	}
	return storage.PlaylistTrack{}, false
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
		return renderBox(56, "Remove from Playlist", b.String(), lipgloss.Color("#e8593c"))
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
		return renderBox(boxW, "Playback speed", b.String(), lipgloss.Color("#e8593c"))
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

		return renderBox(boxW, "Audio settings", b.String(), lipgloss.Color("#e8593c"))
	}

	if a.plMoveActive {
		track, _ := a.selectedTrackForPlaylist()
		b.WriteString("\n ")
		b.WriteString(NormalItemStyle.Render("Add \"" + track.Title + "\" to:"))
		b.WriteString("\n\n ")
		for i, pl := range a.left.playlists {
			line := "  " + pl.Name
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
		b.WriteString(DimItemStyle.Render(" (enter: select  | esc: back)"))
		return renderBox(56, "Move to Playlist", b.String(), lipgloss.Color("#e8593c"))
	}

	if a.infoActive {
		target, ok := a.renameTarget()
		if ok {
			name := target.Name
			artist := target.Artist
			if artist == "" {
				artist = "-"
			}
			durStr := FormatDuration(target.Duration)
			var sizeStr, bitrateStr string
			var pathStr string
			if fi, err := os.Stat(target.Path); err == nil {
				sizeStr = formatBytes(fi.Size())
				if target.Duration > 0 {
					kbps := (fi.Size() * 8) / (1000 * int64(target.Duration))
					bitrateStr = fmt.Sprintf("~%d kbps", kbps)
				} else {
					bitrateStr = "-"
				}
				pathStr = target.Path
			} else {
				sizeStr = "-"
				bitrateStr = "-"
				pathStr = target.Path
			}
			b.WriteString("\n " + DimItemStyle.Render(" Title:") + LocalFileStyle.Render(" "+name))
			b.WriteString("\n " + DimItemStyle.Render(" Artist:") + LocalFileStyle.Render(" "+artist))
			b.WriteString("\n " + DimItemStyle.Render(" Duration: ") + LocalFileStyle.Render(durStr))
			b.WriteString("\n " + DimItemStyle.Render(" Size: ") + LocalFileStyle.Render(sizeStr))
			b.WriteString("\n " + DimItemStyle.Render(" Bitrate: ") + LocalFileStyle.Render(bitrateStr))
			b.WriteString("\n " + DimItemStyle.Render(" Path: ") + LocalFileStyle.Render(pathStr))
			b.WriteString("\n\n")
			b.WriteString(DimItemStyle.Render(" (esc: close)"))
			return renderBox(64, "File Info", b.String(), lipgloss.Color("#E8593C"))
		}
	}

	if a.delActive {
		target, _ := a.renameTarget()
		name := "(No local track)"
		if target != nil {
			name = target.Name
		}
		b.WriteString("\n ")
		b.WriteString(DimItemStyle.Render("Delete \"" + name + "\" permanently?"))
		b.WriteString("\n\n ")

		cancelStyle := NormalItemStyle
		confirmStyle := NormalItemStyle
		if a.cmdCursor == 0 {
			cancelStyle = SelectedItemStyle
		} else {
			confirmStyle = SelectedItemStyle.Foreground(deleteColor)
		}
		b.WriteString(fmt.Sprintf("%s     %s", cancelStyle.Render("[ Cancel ]"), confirmStyle.Render("[ Delete ]")))
		b.WriteString("\n\n")
		b.WriteString(DimItemStyle.Render(" (enter: confirm  | esc: back)"))
		return renderBox(60, "Delete Track", b.String(), lipgloss.Color("#E24B4A"))
	}

	if a.renameActive {
		b.WriteString("\n  ")
		b.WriteString(a.renameInput.View())
		b.WriteString("\n\n")
		b.WriteString(DimItemStyle.Render(" (enter: rename  | esc: back)"))
		w := a.renameInput.Width() + 8
		if w < 48 {
			w = 48
		}
		if a.width > 0 && w > a.width-2 {
			w = a.width - 2
		}
		return renderBox(w, "Rename Title", b.String(), lipgloss.Color("#e8593c"))
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
	return renderBox(40, "Commands", b.String(), lipgloss.Color("#e8593c"))
}
