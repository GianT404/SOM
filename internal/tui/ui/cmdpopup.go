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

type CmdMenuModal struct {
	options []string
	cursor  int
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

func NewCmdMenuModal(options []string) *CmdMenuModal {
	return &CmdMenuModal{options: options, cursor: 0}
}
func (m *CmdMenuModal) Init() tea.Cmd { return nil }
func (m *CmdMenuModal) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc", ":", "q":
			// Đóng menu này (lùi về 1 nấc, hoặc tắt luôn nếu là nấc cuối)
			return m, func() tea.Msg { return CloseModalMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.options) - 1
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
		case "enter":
			if len(m.options) > 0 {
				opt := m.options[m.cursor]
				return m, func() tea.Msg { return ExecuteCmdOptionMsg{Option: opt} }
			}
		}
	}
	return m, nil
}
func (m *CmdMenuModal) View() string {
	var b strings.Builder
	b.WriteString("\n")
	for i, opt := range m.options {
		line := "  " + opt
		if i == m.cursor {
			pad := 36 - runewidth.StringWidth(line)
			if pad < 0 {
				pad = 0
			}
			b.WriteString(SelectedItemStyle.Render(line+strings.Repeat(" ", pad)) + "\n")
		} else {
			b.WriteString(NormalItemStyle.Render(line) + "\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(DimItemStyle.Render(" (enter: select  | esc: close)"))
	return renderBox(40, "Commands", b.String(), themeCol("#e8593c"))
}

// ==========================================
// 2. REMOVE FROM PLAYLIST MODAL
// ==========================================
type RemoveTrackModal struct {
	idxs      []int
	playlists []storage.Playlist
	track     storage.PlaylistTrack
	cursor    int
}

func NewRemoveTrackModal(idxs []int, playlists []storage.Playlist, track storage.PlaylistTrack) *RemoveTrackModal {
	return &RemoveTrackModal{idxs: idxs, playlists: playlists, track: track, cursor: 0}
}
func (m *RemoveTrackModal) Init() tea.Cmd { return nil }
func (m *RemoveTrackModal) Update(msg tea.Msg) (Overlay, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc", ":", "q":
			return m, func() tea.Msg { return CloseModalMsg{} }
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.idxs) - 1
			}
		case "down", "j":
			if m.cursor < len(m.idxs)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
		case "enter":
			if len(m.idxs) > 0 {
				idx := m.idxs[m.cursor]
				return m, tea.Batch(
					func() tea.Msg { return CloseModalMsg{} },
					func() tea.Msg { return ExecuteRemovePlMsg{PlIdx: idx, Track: m.track} },
				)
			}
		}
	}
	return m, nil
}
func (m *RemoveTrackModal) View() string {
	var b strings.Builder
	b.WriteString("\n ")
	b.WriteString(NormalItemStyle.Render("Remove \"" + m.track.Title + "\" from:"))
	b.WriteString("\n\n ")
	for i, plIdx := range m.idxs {
		marker := "  "
		if i == m.cursor {
			marker = "▸ "
		}
		line := marker + m.playlists[plIdx].Name
		if i == m.cursor {
			pad := 51 - runewidth.StringWidth(line)
			if pad < 0 {
				pad = 0
			}
			b.WriteString(SelectedItemStyle.Render(line+strings.Repeat(" ", pad)) + "\n ")
		} else {
			b.WriteString(NormalItemStyle.Render(line) + "\n ")
		}
	}
	b.WriteString("\n ")
	b.WriteString(DimItemStyle.Render(" (enter: remove  | esc: back)"))
	return renderBox(56, "Remove from Playlist", b.String(), themeCol("#e8593c"))
}

// --- APP ROUTING CỦA LỆNH ---
func (a *App) runCmdOption(opt string) tea.Cmd {
	switch opt {
	case "Audio settings":
		modal := NewPresetModal(a.activePreset)
		a.modals = append(a.modals, modal)
		return modal.Init()
	case "Sort":
		modal := NewSortModal(a.left.sortPref)
		a.modals = append(a.modals, modal)
		return modal.Init()
	case "Add to queue":
		track, ok := a.selectedTrackForPlaylist()
		if ok {
			a.playback.Queue = append(a.playback.Queue, domain.Track{
				ID: track.ID, Title: track.Title, Artist: track.Artist, Duration: track.Duration,
			})
			a.setStatus(StatusOKStyle.Render(fmt.Sprintf("> Queued: %s", track.Title)))
		} else {
			a.setStatus(StatusErrStyle.Render("X No track selected"))
		}
		a.modals = nil
		return nil
	case "Rename title":
		if target, ok := a.renameTarget(); ok {
			modal := NewRenameModal(target, a.left.plStore, a.width)
			a.modals = append(a.modals, modal)
			return modal.Init()
		}
		a.setStatus(StatusErrStyle.Render("X No local track selected"))
		// Lưu ý: Không đóng menu nếu lỗi, để user chọn lại
		return nil
	case "Playback speed":
		modal := NewSpeedModal(a.activeSpeed)
		a.modals = append(a.modals, modal)
		return modal.Init()
	case "Delete track":
		if target, ok := a.renameTarget(); ok {
			modal := NewDeleteModal(target, a.left.plStore)
			a.modals = append(a.modals, modal)
			return modal.Init()
		}
		a.setStatus(StatusErrStyle.Render("X No local track selected"))
		return nil
	case "Show file info":
		if target, ok := a.renameTarget(); ok {
			modal := NewInfoModal(target)
			a.modals = append(a.modals, modal)
			return modal.Init()
		}
		a.setStatus(StatusErrStyle.Render("X No local track selected"))
		return nil
	case "Move to playlist":
		if len(a.left.playlists) == 0 {
			modal := NewMoveCreateModal(a.width)
			a.modals = append(a.modals, modal)
			return modal.Init()
		}
		modal := NewMovePickModal(a.left.playlists)
		a.modals = append(a.modals, modal)
		return modal.Init()
	case "Remove from playlist":
		track, ok := a.selectedTrackForPlaylist()
		if ok {
			if idxs := a.playlistsContainingSelected(); len(idxs) > 0 {
				modal := NewRemoveTrackModal(idxs, a.left.playlists, track)
				a.modals = append(a.modals, modal)
				return modal.Init()
			}
		}
		a.setStatus(StatusErrStyle.Render("X Track not in any playlist"))
		return nil
	}
	return nil
}

// --- HELPER FUNC ---
func (a *App) cmdOptionList() []string {
	if a.sidebarActive == SidePlaylists {
		return []string{"Audio settings", "Playback speed", "Show file info", "Remove from playlist"}
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
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05Z"} {
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
