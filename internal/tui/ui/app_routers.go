package ui

import (
	"encoding/json"
	"fmt"
	"os"

	"som/internal/domain"
	"som/internal/storage"
	"som/internal/tui/avrcp"
	"som/internal/tui/player"

	tea "charm.land/bubbletea/v2"
)

func (a *App) handleTick() tea.Cmd {
	a.left.animTick++
	var cmds []tea.Cmd

	cmds = append(cmds, func() tea.Msg { return PlaybackTickMsg{} })

	if a.avrcp != nil && a.playback.NowPlay != nil {
		a.avrcp.UpdatePosition(a.player.Position().Microseconds())
	}

	cmds = append(cmds, tick())
	return tea.Batch(cmds...)
}

// XỬ LÝ ÂM THANH & TRẠNG THÁI PLAYER
func (a *App) handleAudioEvents(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case PlayStartedMsg:
		t := msg.Track
		a.activeContext = SideSearch
		idx := -1
		for i, tr := range a.left.tracks {
			if tr.ID == t.ID {
				idx = i
				break
			}
		}
		a.left.loadingStream = true
		cmds = append(cmds,
			func() tea.Msg { return SetPlaylistMsg{Tracks: a.left.tracks} },
			a.left.spinner.Tick,
			func() tea.Msg { return PlayTrackAtMsg{Index: idx, Track: t} },
		)

	case PlayLocalMsg:
		locals := a.left.locals
		if len(locals) == 0 {
			a.setStatus(StatusErrStyle.Render("X No local files found"))
			break
		}

		pl := make([]domain.Track, len(locals))
		idx := -1
		for i, lf := range locals {
			pl[i] = domain.Track{
				ID:        "local:" + lf.Path,
				Title:     lf.Name,
				Artist:    lf.Artist,
				Duration:  lf.Duration,
				Thumbnail: lf.Thumbnail,
			}
			if lf.Path == msg.Path || lf.Name == msg.Title {
				idx = i
			}
		}
		if idx < 0 {
			idx = 0
		}

		a.activeContext = SideDownloads

		cmds = append(cmds,
			func() tea.Msg { return SetPlaylistMsg{Tracks: pl} },
			func() tea.Msg { return PlayTrackAtMsg{Index: idx, Track: pl[idx]} },
		)

	case StreamResolvedMsg:
		if msg.Err != nil {
			a.setStatus(StatusErrStyle.Render("X Error stream: " + msg.Err.Error()))
			cmds = append(cmds, func() tea.Msg { return PlayNextMsg{} })
		}

	case PlayPlaylistMsg:
		a.activeContext = SidePlaylists
		a.left.loadingStream = true
		cmds = append(cmds,
			func() tea.Msg { return SetPlaylistMsg{Tracks: msg.Tracks} },
			a.left.spinner.Tick,
			func() tea.Msg { return PlayTrackAtMsg{Index: msg.Index, Track: msg.Tracks[msg.Index]} },
		)

	case PlaybackErrorMsg:
		a.setStatus(StatusErrStyle.Render("X Playback error: " + msg.Err.Error()))
		a.left.loadingStream = false
		// Chuyển bài tự động nếu lỗi
		cmds = append(cmds, func() tea.Msg { return PlayNextMsg{} })

	case TrackChangedMsg:
		t := msg.Track
		a.setStatus(StatusOKStyle.Render(">  " + t.Title))

		if a.avrcp != nil {
			a.avrcp.UpdateMetadata(t.ID, t.Title, t.Artist, "", t.Thumbnail, int64(t.Duration)*1_000_000)
			a.avrcp.UpdatePlaybackStatus("Playing")
		}

	case PlaybackStateChangedMsg:
		if a.avrcp != nil {
			if msg.State == int(player.Playing) {
				a.avrcp.UpdatePlaybackStatus("Playing")
			} else if msg.State == int(player.Paused) {
				a.avrcp.UpdatePlaybackStatus("Paused")
			} else if msg.State == int(player.Stopped) {
				a.avrcp.UpdatePlaybackStatus("Stopped")
			}
		}

	case avrcp.AVRCPCmdMsg:
		switch msg.Cmd {
		case "next":
			a.player.Stop()
			cmds = append(cmds, func() tea.Msg { return PlayNextMsg{} })
		case "previous":
			cmds = append(cmds, func() tea.Msg { return PlayPrevMsg{} })
		case "play", "playpause":
			switch a.player.State() {
			case player.Paused:
				a.player.TogglePause()
				if a.avrcp != nil {
					a.avrcp.UpdatePlaybackStatus("Playing")
				}
			case player.Playing:
				a.player.TogglePause()
				if a.avrcp != nil {
					a.avrcp.UpdatePlaybackStatus("Paused")
				}
			case player.Stopped:
				if a.playback.NowPlay != nil {
					cmds = append(cmds, func() tea.Msg { return PlayTrackAtMsg{Index: a.playback.CurrentIdx, Track: *a.playback.NowPlay} })
				}
			}
		case "pause":
			if a.player.State() == player.Playing {
				a.player.TogglePause()
				if a.avrcp != nil {
					a.avrcp.UpdatePlaybackStatus("Paused")
				}
			} else if a.player.State() == player.Paused {
				a.player.TogglePause()
				if a.avrcp != nil {
					a.avrcp.UpdatePlaybackStatus("Playing")
				}
			}
		case "stop":
			a.player.Stop()
			if a.avrcp != nil {
				a.avrcp.UpdatePlaybackStatus("Stopped")
			}
		default:
			if len(msg.Cmd) > 5 && msg.Cmd[:5] == "seek:" {
				var offsetUs int64
				fmt.Sscanf(msg.Cmd[5:], "%d", &offsetUs)
				a.player.SeekBy(float64(offsetUs) / 1000000.0)
			}
		}
		if a.avrcp != nil {
			cmds = append(cmds, a.avrcp.WatchCommands())
		}

	case ApplySpeedMsg:
		pos := 0.0
		if a.playback.NowPlay != nil {
			pos = a.player.Position().Seconds()
		}
		a.player.SetSpeed(msg.Value)
		a.activeSpeed = msg.Index
		a.setStatus(StatusOKStyle.Render("Speed set to: " + msg.Label))
		if a.playback.NowPlay != nil {
			a.player.SeekTo(pos)
		}

	case ApplyPresetMsg:
		a.player.SetAudioFilter(msg.Filter)
		a.activePreset = msg.Index
		a.setStatus(StatusOKStyle.Render("Applied: " + msg.Name))
		if a.playback.NowPlay != nil {
			pos := int(a.player.Position().Seconds())
			a.player.SeekTo(float64(pos))
		}
	}
	return tea.Batch(cmds...)
}

// XỬ LÝ DỮ LIỆU, SQLITE & THAY ĐỔI TRẠNG THÁI PANEL
func (a *App) handleDataEvents(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case SearchResultMsg:
		if msg.Err != nil {
			a.setStatus(StatusErrStyle.Render("X " + msg.Err.Error()))
		}
	case OpenSettingsMsg:

		modal := NewSettingsModal(a.settingSwitches(), a.width)

		a.modals = append(a.modals, modal)

		cmds = append(cmds, modal.Init())

	case OpenHelpMsg:

		modal := NewHelpModal(a.width, a.height)

		a.modals = append(a.modals, modal)

		cmds = append(cmds, modal.Init())

	case ApplySettingMsg:

		a.applySetting(msg.Index, msg.Value)
	case ExecuteCmdOptionMsg:
		if c := a.runCmdOption(msg.Option); c != nil {
			cmds = append(cmds, c)
		}
	case ExecuteRemovePlMsg:
		if a.left.plStore != nil {
			pl := a.left.playlists[msg.PlIdx]
			if err := a.left.plStore.RemoveTrackFromPlaylist(pl.ID, msg.Track.Path); err == nil {
				for j := range pl.Tracks {
					if pl.Tracks[j].Path == msg.Track.Path || pl.Tracks[j].ID == msg.Track.ID {
						a.left.playlists[msg.PlIdx].Tracks = append(pl.Tracks[:j], pl.Tracks[j+1:]...)
						break
					}
				}
				a.setStatus(StatusOKStyle.Render("> Removed from \"" + pl.Name + "\""))
			} else {
				a.setStatus(StatusErrStyle.Render("X Failed: " + err.Error()))
			}
		}
	case DownloadDoneMsg:
		if msg.Err != nil {
			a.setStatus(StatusErrStyle.Render(msg.Err.Error()))
		} else {
			// Cập nhật trạng thái tải xong, đang xử lý meta
			a.setStatus(StatusMsgStyle.Render("> Saving metadata for " + msg.Path + "..."))
			if a.left.plStore != nil && msg.Path != "" {
				// Dispatch lệnh chạy ngầm
				cmds = append(cmds, saveLocalMetaCmd(a.provider, a.left.plStore, msg.Path, msg.Track))
			}
		}
	case MetaSavedMsg:
		if msg.Err != nil {
			a.setStatus(StatusErrStyle.Render("X Metadata error: " + msg.Err.Error()))
		} else {
			a.setStatus(StatusOKStyle.Render("Saved " + msg.Path))
		}
	case ImportDoneMsg:
		a.handleImportDone(msg)

	case ApplySortMsg:
		a.left.sortPref = msg.Key
		if a.left.plStore != nil {
			a.left.plStore.SetSetting("sort_downloads", msg.Key)
		}
		a.left.scanLocalFiles()
		a.setStatus(StatusOKStyle.Render("> Sorted by " + msg.Name))
	case InitMoveSessionMsg:
		idx := msg.TargetPlIdx
		if idx == -1 && msg.NewPlName != "" && a.left.plStore != nil {
			if pl, err := a.left.plStore.CreatePlaylist(msg.NewPlName); err == nil {
				a.left.playlists = append(a.left.playlists, pl)
				idx = len(a.left.playlists) - 1
			} else {
				a.setStatus(StatusErrStyle.Render("X " + err.Error()))
				break
			}
		}
		if idx >= 0 {
			a.moveSession = &MoveSession{TargetPlIdx: idx, Selected: make(map[string]bool)}
			cmds = append(cmds, a.switchSidebar(SideDownloads))
			a.left.input.Blur()
		}
	case ExecuteMoveMsg:
		a.applyMoveToPlaylist()
		a.moveSession = nil
	}
	return tea.Batch(cmds...)
}

// XỬ LÝ TOÀN BỘ PHÍM BẤM
func (a *App) handleKeys(msg tea.KeyPressMsg) tea.Cmd {
	var cmds []tea.Cmd

	if msg.String() == "ctrl+c" || msg.String() == "alt+q" {
		a.player.Stop()
		if a.avrcp != nil {
			a.avrcp.Close()
		}
		return tea.Quit
	}

	switch msg.String() {
	case "esc":
		if a.moveSession != nil {
			a.moveSession = nil
			a.setStatus(StatusMsgStyle.Render("> No changes to playlist"))
		} else if a.palette.Visible() {
			a.palette = a.palette.Close(a.sidebarActive)
		} else if !a.left.input.Focused() && !a.left.plInput.Focused() && !a.left.showDeletePopup && !a.left.showPlInput {
			if a.sidebarActive == SidePlaylists && a.left.activePlaylist != nil {
			} else {
				modal := NewEscMenuModal()
				a.modals = append(a.modals, modal)
				cmds = append(cmds, modal.Init())
			}
		}
	case ".":
		if a.moveSession != nil && a.sidebarActive == SideDownloads && !a.left.input.Focused() {
			a.toggleMoveSelection()
		}
	case "i":
		if a.moveSession != nil && a.sidebarActive == SideDownloads && !a.left.input.Focused() {
			if a.selectedMoveCount() == 0 {
				a.setStatus(StatusErrStyle.Render("X No tracks selected to move"))
			} else {
				plName := a.left.playlists[a.moveSession.TargetPlIdx].Name
				modal := NewMoveConfirmModal(plName, a.selectedMoveCount())
				a.modals = []Overlay{modal}
				cmds = append(cmds, modal.Init())
			}
		}
	case "1", "2", "3", "4", "5", "6", "7":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		targetTab := SidebarItem(msg.String()[0] - '1')
		cmds = append(cmds, a.switchSidebar(targetTab))
	case "\\":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		if a.palette.Visible() {
			a.palette = a.palette.Close(a.sidebarActive)
		} else {
			var cmd tea.Cmd
			a.palette, cmd = a.palette.Open()
			cmds = append(cmds, cmd)
		}
	case ":":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		modal := NewCmdMenuModal(a.cmdOptionList())
		a.modals = []Overlay{modal}
		cmds = append(cmds, modal.Init())
		cmds = append(cmds, modal.Init())
	case "tab":
		if a.left.input.Focused() {
			a.left.input.Blur()
		} else if a.left.plInput.Focused() {
			a.left.plInput.Blur()
			a.left.showPlInput = false
		} else {
			next := (a.sidebarActive + 1) % sideCount
			cmds = append(cmds, a.switchSidebar(next))
		}
	case "space":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		a.player.TogglePause()
		if a.avrcp != nil {
			if a.player.State() == player.Playing {
				a.avrcp.UpdatePlaybackStatus("Playing")
			} else if a.player.State() == player.Paused {
				a.avrcp.UpdatePlaybackStatus("Paused")
			}
		}
	case "right":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		a.player.SeekBy(5)
		cmds = append(cmds, func() tea.Msg { return PlaybackTickMsg{} })
	case "left":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		a.player.SeekBy(-5)
		cmds = append(cmds, func() tea.Msg { return PlaybackTickMsg{} })
	case "]", "}":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		cmds = append(cmds, func() tea.Msg { return PlayNextMsg{} })
	case "[", "{":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		cmds = append(cmds, func() tea.Msg { return PlayPrevMsg{} })
	case "r", "R":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		if a.sidebarActive == SideImport {
			break
		}
		cmds = append(cmds, func() tea.Msg { return ToggleRandomMsg{} })
	case "up":
		if a.sidebarActive == SideLogs {
			if a.logOffset < LogBuf.Len()-1 {
				a.logOffset++
			}
		}
	case "down":
		if a.sidebarActive == SideLogs {
			if a.logOffset > 0 {
				a.logOffset--
			}
		}
	case "+", "=":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		v := a.player.Volume() + 0.05
		a.player.SetVolume(v)
	case "-", "_":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		v := a.player.Volume() - 0.05
		a.player.SetVolume(v)

	case "?":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		modal := NewHelpModal(a.width, a.height)
		a.modals = append(a.modals, modal)
		cmds = append(cmds, modal.Init())
	}
	return tea.Batch(cmds...)
}
func saveLocalMetaCmd(p domain.MusicProvider, store *storage.DB, path string, t domain.Track) tea.Cmd {
	return func() tea.Msg {
		info, _ := os.Stat(path)
		fileSize, fileMTime := int64(0), ""
		if info != nil {
			fileSize = info.Size()
			fileMTime = info.ModTime().Format("2006-01-02 15:04:05")
		}

		// Tác vụ mạng chạy ngầm, không block UI
		lr, _ := getCachedLyrics(p, store, t.ID, t.Title, t.Artist, t.Duration)
		lrJSON, _ := json.Marshal(lr)

		err := store.UpsertLocalFileWithMeta(storage.LocalFile{
			Path:      path,
			Name:      t.Title,
			Artist:    t.Artist,
			Duration:  t.Duration,
			VideoID:   t.ID,
			Thumbnail: t.Thumbnail,
			FileSize:  fileSize,
			FileMTime: fileMTime,
		}, &storage.LocalFileMeta{
			Artist:     t.Artist,
			Title:      t.Title,
			VideoID:    t.ID,
			Thumbnail:  t.Thumbnail,
			LyricsJSON: string(lrJSON),
		})

		return MetaSavedMsg{Path: path, Track: t, Err: err}
	}
}
