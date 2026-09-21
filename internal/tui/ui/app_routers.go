package ui

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"som/internal/domain"
	"som/internal/storage"
	"som/internal/tui/avrcp"
	"som/internal/tui/player"

	tea "charm.land/bubbletea/v2"
)

func (a *App) handleTick() tea.Cmd {
	a.left.animTick++
	a.right.TickAt()
	var cmds []tea.Cmd

	if !a.left.loadingStream && a.playback.SongStarted && a.player.State() == player.Stopped && a.playback.NowPlay != nil {
		playErr := a.player.PlaybackError()
		a.playback.NowPlay = nil
		if a.avrcp != nil {
			a.avrcp.UpdatePlaybackStatus("Stopped")
		}
		if playErr != nil {
			a.setStatus(StatusErrStyle.Render("X playback failed: " + playErr.Error()))
		} else if a.sidebarActive != SideImport {
			if a.player.PlayFromBuffer() {
				if a.playback.NextPlay != nil {
					a.playback.NowPlay = a.playback.NextPlay
					if len(a.playback.Queue) > 0 && a.playback.Queue[0].ID == a.playback.NowPlay.ID {
						a.playback.Queue = a.playback.Queue[1:]
						if a.left.qCursor >= len(a.playback.Queue) {
							a.left.qCursor = maxInt(len(a.playback.Queue)-1, 0)
						}
					} else {
						for i, tr := range a.playback.Playlist {
							if tr.ID == a.playback.NowPlay.ID {
								a.playback.CurrentIdx = i
								break
							}
						}
					}
					a.playback.NextPlay = nil
					a.playback.SongStarted = true
					a.playback.PlayerGen = a.player.Generation()
					a.updateCursorForTrack(*a.playback.NowPlay)
					a.right.SetTrack(a.playback.NowPlay)
					a.setStatus(StatusOKStyle.Render(">  " + a.playback.NowPlay.Title))
					if a.avrcp != nil {
						a.avrcp.UpdateMetadata(a.playback.NowPlay.ID, a.playback.NowPlay.Title, a.playback.NowPlay.Artist, "", a.playback.NowPlay.Thumbnail, int64(a.playback.NowPlay.Duration)*1_000_000)
						a.avrcp.UpdatePlaybackStatus("Playing")
					}
					a.loadLyricsForTrack(*a.playback.NowPlay)
				}
			} else if len(a.playback.Queue) > 0 {
				t := a.playback.Queue[0]
				a.playback.Queue = a.playback.Queue[1:]
				if a.left.qCursor >= len(a.playback.Queue) {
					a.left.qCursor = maxInt(len(a.playback.Queue)-1, 0)
				}
				a.highlightTrackInSidebar(t)
				cmds = append(cmds, a.playTrackAt(-1, t))
				a.setStatus(StatusOKStyle.Render(fmt.Sprintf("> Playing from queue: %s", t.Title)))
			} else {
				cmds = append(cmds, a.playNext())
			}
		}
	}

	if a.playback.SongStarted && a.player.State() == player.Playing && a.playback.NowPlay != nil && a.sidebarActive != SideImport {
		pos := a.player.Position()
		dur := time.Duration(a.playback.NowPlay.Duration) * time.Second
		if remaining := dur - pos; remaining > 0 && remaining <= 3*time.Second {
			a.triggerPreDecodeNext()
		}
	}

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
		a.playback.Playlist = a.left.tracks
		a.playback.ShuffleHist = nil
		a.playback.History = nil
		a.activeContext = SideSearch
		idx := -1
		for i, tr := range a.playback.Playlist {
			if tr.ID == t.ID {
				idx = i
				break
			}
		}
		a.left.loadingStream = true
		cmds = append(cmds, a.left.spinner.Tick, a.playTrackAt(idx, t))

	case PlayLocalMsg:
		locals := a.left.locals
		if len(locals) == 0 {
			a.setStatus(StatusErrStyle.Render("X No local files found"))
			break
		}
		a.playback.Playlist = make([]domain.Track, len(locals))
		a.playback.ShuffleHist = nil
		a.playback.History = nil
		idx := -1
		for i, lf := range locals {
			a.playback.Playlist[i] = domain.Track{ID: "local:" + lf.Path, Title: lf.Name, Artist: lf.Artist, Duration: lf.Duration, Thumbnail: lf.Thumbnail}
			if lf.Path == msg.Path || lf.Name == msg.Title {
				idx = i
			}
		}
		if idx < 0 {
			idx = 0
		}
		a.activeContext = SideDownloads
		cmds = append(cmds, a.playTrackAt(idx, a.playback.Playlist[idx]))
		for fi, lf := range a.left.getFilteredLocals() {
			if lf.Path == msg.Path || lf.Name == msg.Title {
				a.left.dlCursor = fi
				a.left.dlOffset = 0
				if fi >= a.left.visibleRows() {
					a.left.dlOffset = fi - a.left.visibleRows() + 1
				}
				break
			}
		}

	case StreamStartedMsg:
		if msg.Err != nil {
			a.left.loadingStream = false
			a.setStatus(StatusErrStyle.Render("X " + msg.Err.Error()))
			cmds = append(cmds, a.playNext())
			break
		}
		if msg.Gen != a.playback.PlayerGen {
			a.left.loadingStream = false
			break
		}
		a.left.loadingStream = false
		a.playback.NowPlay = &msg.Track
		a.playback.SongStarted = true
		a.right.SetTrack(&msg.Track)
		a.setStatus(StatusOKStyle.Render(">  " + msg.Track.Title))
		if msg.LyricsErr != nil {
			a.right.SetLyrics(domain.LyricsResp{Plain: "(no lyrics available)"})
		} else {
			a.right.SetLyrics(msg.Lyrics)
		}
		cmds = append(cmds, a.right.spinner.Tick)
		if a.avrcp != nil {
			a.avrcp.UpdateMetadata(msg.Track.ID, msg.Track.Title, msg.Track.Artist, "", msg.Track.Thumbnail, int64(msg.Track.Duration)*1_000_000)
			a.avrcp.UpdatePlaybackStatus("Playing")
		}

	case PlayPlaylistMsg:
		a.playback.Playlist = msg.Tracks
		a.playback.ShuffleHist = nil
		a.playback.History = nil
		a.activeContext = SidePlaylists
		a.left.loadingStream = true
		cmds = append(cmds, a.left.spinner.Tick, a.playTrackAt(msg.Index, msg.Tracks[msg.Index]))

	case PlayQueueMsg:
		if msg.Index >= 0 && msg.Index < len(a.playback.Queue) {
			t := a.playback.Queue[msg.Index]
			a.playback.Queue = append(a.playback.Queue[:msg.Index], a.playback.Queue[msg.Index+1:]...)
			if a.left.qCursor >= len(a.playback.Queue) {
				a.left.qCursor = maxInt(len(a.playback.Queue)-1, 0)
			}
			a.highlightTrackInSidebar(t)
			cmds = append(cmds, a.playTrackAt(-1, t))
			a.setStatus(StatusOKStyle.Render(fmt.Sprintf("> Playing from queue: %s", t.Title)))
		}

	case RemoveFromQueueMsg:
		if msg.Index >= 0 && msg.Index < len(a.playback.Queue) {
			removed := a.playback.Queue[msg.Index]
			a.playback.Queue = append(a.playback.Queue[:msg.Index], a.playback.Queue[msg.Index+1:]...)
			if a.left.qCursor >= len(a.playback.Queue) {
				a.left.qCursor = maxInt(len(a.playback.Queue)-1, 0)
			}
			a.setStatus(StatusOKStyle.Render(fmt.Sprintf("> Removed from queue: %s", removed.Title)))
		}

	case avrcp.AVRCPCmdMsg:
		switch msg.Cmd {
		case "next":
			a.player.Stop()
			cmds = append(cmds, a.playNext())
		case "previous":
			cmds = append(cmds, a.playPrev())
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
					cmds = append(cmds, a.playTrackAt(a.playback.CurrentIdx, *a.playback.NowPlay))
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
	case DownloadDoneMsg:
		if msg.Err != nil {
			a.setStatus(StatusErrStyle.Render(msg.Err.Error()))
		} else {
			if a.left.plStore != nil && msg.Path != "" {
				info, _ := os.Stat(msg.Path)
				fileSize, fileMTime := int64(0), ""
				if info != nil {
					fileSize = info.Size()
					fileMTime = info.ModTime().Format("2006-01-02 15:04:05")
				}
				lr, _ := getCachedLyrics(a.provider, msg.Track.ID, msg.Track.Title, msg.Track.Artist, msg.Track.Duration)
				lrJSON, _ := json.Marshal(lr)
				_ = a.left.plStore.UpsertLocalFileWithMeta(storage.LocalFile{Path: msg.Path, Name: msg.Track.Title, Artist: msg.Track.Artist, Duration: msg.Track.Duration, VideoID: msg.Track.ID, Thumbnail: msg.Track.Thumbnail, FileSize: fileSize, FileMTime: fileMTime}, &storage.LocalFileMeta{Artist: msg.Track.Artist, Title: msg.Track.Title, VideoID: msg.Track.ID, Thumbnail: msg.Track.Thumbnail, LyricsJSON: string(lrJSON)})
			}
			a.setStatus(StatusOKStyle.Render("Saved " + msg.Path))
		}
	case ImportDoneMsg:
		a.handleImportDone(msg)
	case RenameDoneMsg:
		if msg.Err != nil {
			a.setStatus(StatusErrStyle.Render("X " + msg.Err.Error()))
			break
		}
		for i := range a.playback.Playlist {
			if a.playback.Playlist[i].ID == "local:"+msg.OldPath {
				a.playback.Playlist[i].ID = "local:" + msg.NewPath
				a.playback.Playlist[i].Title = msg.NewTitle
			}
		}
		if a.playback.NowPlay != nil && strings.HasPrefix(a.playback.NowPlay.ID, "local:") && strings.TrimPrefix(a.playback.NowPlay.ID, "local:") == msg.OldPath {
			a.playback.NowPlay.ID = "local:" + msg.NewPath
			a.playback.NowPlay.Title = msg.NewTitle
		}
		a.left.scanLocalFiles()
		if a.left.plStore != nil {
			if pls, err := a.left.plStore.LoadAllPlaylists(); err == nil {
				a.left.playlists = pls
				if a.left.activePlaylist != nil {
					for i := range a.left.playlists {
						if a.left.playlists[i].ID == a.left.activePlaylist.ID {
							a.left.activePlaylist = &a.left.playlists[i]
							break
						}
					}
				}
			}
		}
		a.setStatus(StatusOKStyle.Render("> Renamed to " + msg.NewTitle))
	case DeleteDoneMsg:
		if msg.Err != nil {
			a.setStatus(StatusErrStyle.Render("X " + msg.Err.Error()))
			break
		}
		if a.playback.NowPlay != nil && a.playback.NowPlay.ID == "local:"+msg.Path {
			a.player.Stop()
			a.playback.NowPlay = nil
			a.playback.SongStarted = false
			a.playback.NextPlay = nil
			a.right.SetTrack(nil)
		}
		a.left.scanLocalFiles()
		if a.left.dlCursor >= len(a.left.locals) {
			a.left.dlCursor = maxInt(len(a.left.locals)-1, 0)
		}
		if a.left.plStore != nil {
			if pls, err := a.left.plStore.LoadAllPlaylists(); err == nil {
				a.left.playlists = pls
				if a.left.activePlaylist != nil {
					for i := range a.left.playlists {
						if a.left.playlists[i].ID == a.left.activePlaylist.ID {
							a.left.activePlaylist = &a.left.playlists[i]
							break
						}
					}
				}
			}
		}
		var newPl []domain.Track
		for _, t := range a.playback.Playlist {
			if t.ID != "local:"+msg.Path {
				newPl = append(newPl, t)
			}
		}
		a.playback.Playlist = newPl

		var newQueue []domain.Track
		for _, t := range a.playback.Queue {
			if t.ID != "local:"+msg.Path {
				newQueue = append(newQueue, t)
			}
		}
		a.playback.Queue = newQueue
		a.left.queue = a.playback.Queue
		a.setStatus(StatusOKStyle.Render("> Deleted " + msg.Name))
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

	if a.showHelpPopup {
		switch msg.String() {
		case "?", "esc", "q":
			a.showHelpPopup = false
		}
		return nil
	}
	if a.showEscMenu {
		switch msg.String() {
		case "esc", "q":
			a.showEscMenu = false
		case "up":
			if a.escMenuCursor > 0 {
				a.escMenuCursor--
			}
		case "down":
			if a.escMenuCursor < len(escMenuItems)-1 {
				a.escMenuCursor++
			}
		case "enter":
			switch a.escMenuCursor {
			case 0:
				a.showEscMenu = false
				a.showSettings = true
				a.settingsCursor = 0
			case 1:
				a.showEscMenu = false
				a.showHelpPopup = true
			case 2:
				a.showEscMenu = false
				return tea.Quit
			}
		}
		return nil
	}
	if a.showSettings {
		items := a.settingSwitches()
		switch msg.String() {
		case "esc", "q":
			a.showSettings = false
		case "up":
			if a.settingsCursor > 0 {
				a.settingsCursor--
			}
		case "down":
			if a.settingsCursor < len(items)-1 {
				a.settingsCursor++
			}
		case "left":
			if a.settingsCursor >= 0 && a.settingsCursor < len(items) {
				items[a.settingsCursor].ToggleLeft()
				a.applySetting(a.settingsCursor, items[a.settingsCursor].Value())
			}
		case "right":
			if a.settingsCursor >= 0 && a.settingsCursor < len(items) {
				items[a.settingsCursor].ToggleRight()
				a.applySetting(a.settingsCursor, items[a.settingsCursor].Value())
			}
		}
		return nil
	}
	if a.showCmdPopup {
		if c := a.updateCmdPopup(msg); c != nil {
			cmds = append(cmds, c)
		}
		return tea.Batch(cmds...)
	}

	switch msg.String() {
	case "esc":
		if a.moveSession != nil {
			a.moveSession = nil
			a.setStatus(StatusMsgStyle.Render("> No changes to playlist"))
		} else if a.palette.Visible() {
			a.palette.Close()
		} else if !a.left.input.Focused() && !a.left.plInput.Focused() && !a.left.showDeletePopup && !a.left.showPlInput {
			if a.sidebarActive == SidePlaylists && a.left.activePlaylist != nil {
			} else {
				a.showEscMenu = true
				a.escMenuCursor = 0
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
				a.activeModal = modal
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
			a.palette.Close()
		} else {
			cmds = append(cmds, a.palette.Open())
		}
	case ":":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		if a.showCmdPopup {
			a.showCmdPopup = false
		} else {
			a.showCmdPopup = true
			a.cmdCursor = 0
			a.cmdMenuCursor = 0
		}
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
		a.right.TickAt()
	case "left":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		a.player.SeekBy(-5)
		a.right.TickAt()
	case "]", "}":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		cmds = append(cmds, a.playNext())
	case "[", "{":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		cmds = append(cmds, a.playPrev())
	case "r", "R":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		a.playback.Random = !a.playback.Random
		a.playback.ShuffleHist = nil
		a.playback.History = nil
		a.syncPlaylistState()
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
		a.setStatus(StatusMsgStyle.Render(fmt.Sprintf("Volume: %d%%", int(math.Round(v*100)))))
	case "-", "_":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		v := a.player.Volume() - 0.05
		a.player.SetVolume(v)
		if v <= 0.01 {
			a.setStatus(StatusMsgStyle.Render("Volume: MUTE"))
		} else {
			a.setStatus(StatusMsgStyle.Render(fmt.Sprintf("Volume: %d%%", int(math.Round(v*100)))))
		}
	case "?":
		if a.left.input.Focused() || a.left.plInput.Focused() {
			break
		}
		a.showHelpPopup = true
	}
	return tea.Batch(cmds...)
}
