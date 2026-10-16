package ui

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"som/internal/domain"
	"som/internal/storage"
	"som/internal/tui/avrcp"
	"som/internal/tui/player"

	tea "charm.land/bubbletea/v2"
)

func (a *App) handleTick() tea.Cmd {
	if a.player == nil ||
		a.playback == nil ||
		!a.playback.SongStarted {
		a.playbackTickActive = false
		a.listenTickAt = time.Time{}
		return nil
	}

	var cmds []tea.Cmd

	cmds = append(cmds, func() tea.Msg {
		return PlaybackTickMsg{}
	})

	if a.player.State() == player.Stopped {
		a.playbackTickActive = false
		a.listenTickAt = time.Time{}
		return tea.Batch(cmds...)
	}

	if a.player.State() == player.Playing {
		if a.left.plStore != nil {
			now := time.Now()

			if !a.listenTickAt.IsZero() {
				a.listenRemainder += now.Sub(a.listenTickAt)
			}

			a.listenTickAt = now

			seconds := int64(a.listenRemainder / time.Second)
			if seconds > 0 {
				a.listenRemainder -= time.Duration(seconds) * time.Second

				totalSeconds, _ := strconv.ParseInt(
					a.left.plStore.GetSetting(storage.TotalListeningTimeSettingKey),
					10,
					64,
				)

				a.left.plStore.SetSetting(
					storage.TotalListeningTimeSettingKey,
					strconv.FormatInt(totalSeconds+seconds, 10),
				)
			}
		} else {
			a.listenTickAt = time.Time{}
		}

		// Giữ nguyên logic cũ
		if a.avrcp != nil && a.playback.NowPlay != nil {
			a.avrcp.UpdatePosition(a.player.Position().Microseconds())
		}

		cmds = append(cmds, tick())
	}

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
		if a.playback != nil && msg.Gen != a.playback.PlayerGen {
			break
		}
		if msg.Err != nil {
			a.setStatus(StatusErrStyle.Render("X Error stream: " + msg.Err.Error()))
			cmds = append(cmds, func() tea.Msg { return PlayNextMsg{} })
		}

	case PlayPlaylistMsg:
		if msg.Index < 0 || msg.Index >= len(msg.Tracks) {
			return nil
		}
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
		if a.playback != nil && msg.Gen < a.playback.PlayerGen {
			break
		}
		t := msg.Track
		if msg.IsLocal {
			a.setStatus(StatusOKStyle.Render(">  " + t.Title))
		}

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
			if a.playback != nil {
				if c := a.playback.Stop(); c != nil {
					cmds = append(cmds, c)
				}
			} else {
				a.player.Stop()
			}
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
	case LyricsLanguageSelectedMsg:
		a.right.SetPreferredLanguage(msg.Language)
		if a.left.plStore != nil && msg.Language != "" {
			a.left.plStore.SetSetting(lyricsLanguageSettingKey, msg.Language)
		}
	case NobodyHereMsg:
		modal := NewVersionModal(a.version, a.width, a.height)
		a.modals = append(a.modals, modal)
		cmds = append(cmds, modal.Init())
	case PinStatusMsg:
		if msg.Success {
			a.setStatus(StatusOKStyle.Render(">  " + msg.Message))
		} else {
			a.setStatus(StatusErrStyle.Render("X  " + msg.Message))
		}
	case RenameDoneMsg:
		if msg.Err != nil {
			a.setStatus(StatusErrStyle.Render(
				"X  Rename failed: " + msg.Err.Error(),
			))
			break
		}

		a.setStatus(StatusOKStyle.Render(
			`>  Renamed title: "` + msg.NewTitle + `"`,
		))

	case DeleteDoneMsg:
		if msg.Err != nil {
			a.setStatus(StatusErrStyle.Render(
				"X  Delete failed: " + msg.Err.Error(),
			))
			break
		}

		a.setStatus(StatusOKStyle.Render(
			`>  Deleted: "` + msg.Name + `"`,
		))
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
	case RenamePlaylistDoneMsg:
		if msg.Err != nil {
			a.setStatus(
				StatusErrStyle.Render("X " + msg.Err.Error()),
			)
			break
		}

		updated := false

		for i := range a.left.playlists {
			if a.left.playlists[i].ID == msg.ID {
				a.left.playlists[i].Name = msg.NewName
				updated = true
				break
			}
		}

		if !updated {
			a.setStatus(
				StatusErrStyle.Render("X Playlist no longer exists"),
			)
			break
		}

		// Nếu activePlaylist đang trỏ vào slice cũ thì bind lại.
		a.left.rebindActivePlaylist()

		// Rename có thể làm playlist biến mất khỏi search hiện tại.
		filtered := a.left.getFilteredPlaylists()

		if len(filtered) == 0 {
			a.left.plCursor = 0
			a.left.plOffset = 0
		} else {
			if a.left.plCursor >= len(filtered) {
				a.left.plCursor = len(filtered) - 1
			}
			if a.left.plCursor < 0 {
				a.left.plCursor = 0
			}
			a.left.scrollPlIntoView()
		}

		a.setStatus(
			StatusOKStyle.Render(
				`> Renamed playlist to "` + msg.NewName + `"`,
			),
		)
	case ExecuteRemovePlMsg:
		if a.left.plStore == nil ||
			msg.PlIdx < 0 ||
			msg.PlIdx >= len(a.left.playlists) {
			break
		}

		pl := a.left.playlists[msg.PlIdx]

		if err := a.left.plStore.RemoveTrackFromPlaylist(pl.ID, msg.Track.Path); err != nil {
			a.setStatus(StatusErrStyle.Render("X Failed: " + err.Error()))
			break
		}

		// Xóa khỏi state hiện tại.
		for j := range pl.Tracks {
			if pl.Tracks[j].Path == msg.Track.Path ||
				pl.Tracks[j].ID == msg.Track.ID {
				a.left.playlists[msg.PlIdx].Tracks =
					append(pl.Tracks[:j], pl.Tracks[j+1:]...)
				break
			}
		}

		if a.left.activePlaylist != nil &&
			a.left.activePlaylist.ID == pl.ID {

			tracks := a.left.getFilteredPlaylistTracks()

			if len(tracks) == 0 {
				a.left.plCursor = 0
				a.left.plOffset = 0
			} else {
				if a.left.plCursor >= len(tracks) {
					a.left.plCursor = len(tracks) - 1
				}
				if a.left.plCursor < 0 {
					a.left.plCursor = 0
				}

				a.left.scrollPlIntoView()
			}
		}

		a.setStatus(
			StatusOKStyle.Render("> Removed from \"" + pl.Name + "\""),
		)

	case StreamStartedMsg:
		if a.playback != nil && msg.Gen != a.playback.PlayerGen {
			break
		}

		if a.playback != nil && a.playback.NowPlay != nil {
			a.setStatus(
				StatusOKStyle.Render(">  " + a.playback.NowPlay.Title),
			)
		}

	case DownloadDoneMsg:
		if msg.Err != nil {
			log.Printf("[download] failed title=%q error=%v", msg.Track.Title, msg.Err)
			a.setStatus(StatusErrStyle.Render(msg.Err.Error()))
		} else {
			log.Printf("[download] saving metadata title=%q path=%q", msg.Track.Title, msg.Path)
			a.setStatus(StatusMsgStyle.Render("> Saving metadata for " + msg.Path + "..."))
			if a.left.plStore != nil && msg.Path != "" {
				// Dispatch lệnh chạy ngầm
				cmds = append(cmds, saveLocalMetaCmd(a.provider, a.left.plStore, msg.Path, msg.Track))
			}
		}
	case MetaSavedMsg:
		if msg.Err != nil {
			log.Printf("[download] metadata save failed path=%q error=%v", msg.Path, msg.Err)
			a.setStatus(StatusErrStyle.Render("X Metadata error: " + msg.Err.Error()))
		} else {
			log.Printf("[download] metadata saved path=%q", msg.Path)
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
				a.left.rebindActivePlaylist()
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
