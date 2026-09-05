package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"som/internal/domain"
	"som/internal/storage"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ImportFile struct {
	Path     string
	Name     string
	Duration int
	Selected bool
}

type ImportPanel struct {
	files       []ImportFile
	cursor      int
	offset      int
	previewPath string
	importing   bool
	importDone  int
	importTotal int
	spinner     spinner.Model
	width       int
	height      int
}

func NewImportPanel() ImportPanel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = StatusOKStyle
	return ImportPanel{spinner: sp}
}

func (ip *ImportPanel) ScanImportDirs(downloadDir string, store *storage.DB) {
	ip.files = nil

	dirs := []string{
		downloadDir,
		filepath.Join(os.Getenv("HOME"), "Music"),
		filepath.Join(os.Getenv("HOME"), "Downloads"),
	}

	imported := make(map[string]bool)
	if store != nil {
		if rows, err := store.ListAllLocalFilesSorted("name"); err == nil {
			for _, f := range rows {
				imported[f.Path] = true
			}
		}
	}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !storage.IsSupportedAudio(e.Name()) {
				continue
			}
			absPath := filepath.Join(dir, e.Name())
			if imported[absPath] {
				continue
			}
			ext := filepath.Ext(e.Name())
			name := strings.TrimSuffix(e.Name(), ext)
			ip.files = append(ip.files, ImportFile{
				Path: absPath,
				Name: name,
			})
		}
	}
}

func (ip *ImportPanel) SetSize(w, h int) {
	ip.width = w
	ip.height = h
}

func (ip ImportPanel) ViewImportContent(w, h int) string {
	innerW := w - 4

	count := len(ip.files)

	if ip.importing {
		progress := fmt.Sprintf("Imported %d / %d files...", ip.importDone, ip.importTotal)
		content := ip.spinner.View() + " " + progress
		return renderBox(w, "Importing", content, lipgloss.Color("#e8593c"))
	}

	if count == 0 {
		return renderBox(w, fmt.Sprintf("Import (%d)", count), DimItemStyle.Render(" No new audio files found."), lipgloss.Color("#7c7986"))
	}

	vis := ip.visibleRows()
	end := ip.offset + vis
	if end > count {
		end = count
	}

	var b strings.Builder
	for i := ip.offset; i < end; i++ {
		f := ip.files[i]
		tick := " "
		if f.Selected {
			tick = "+"
		}
		if i == ip.cursor {
			line := fmt.Sprintf("   [%s] %s", tick, f.Name)
			b.WriteString("\n")
			b.WriteString(LocalFileSelectedStyle.Width(innerW).Render(line))
		} else {
			line := fmt.Sprintf("   [%s] %s", tick, f.Name)
			b.WriteString("\n")
			b.WriteString(LocalFileStyle.Width(innerW).Render(line))
		}
	}

	selCount := ip.selectedCount()
	listBox := renderBox(w, fmt.Sprintf("Import (%d/%d)", selCount, count), b.String(), lipgloss.Color("#7c7986"))

	return listBox
}

func (ip ImportPanel) visibleRows() int {
	h := ip.height - 6
	if h < 3 {
		h = 3
	}
	return h
}

func (ip *ImportPanel) selectedCount() int {
	n := 0
	for _, f := range ip.files {
		if f.Selected {
			n++
		}
	}
	return n
}

type ImportDoneMsg struct {
	Imported int
}

type ImportProgressMsg struct {
	Done  int
	Total int
}

func importCmd(store *storage.DB, files []ImportFile) tea.Cmd {
	return func() tea.Msg {
		total := 0
		for _, f := range files {
			if f.Selected {
				total++
			}
		}
		imported := 0
		for _, f := range files {
			if !f.Selected {
				continue
			}
			dur := storage.GetFileDuration(f.Path)
			info, err := os.Stat(f.Path)
			if err != nil {
				continue
			}
			_ = store.UpsertLocalFile(storage.LocalFile{
				Path:      f.Path,
				Name:      f.Name,
				Duration:  dur,
				FileSize:  info.Size(),
				FileMTime: info.ModTime().Format("2006-01-02T15:04:05Z07:00"),
			})
			imported++
		}
		return ImportDoneMsg{Imported: imported}
	}
}

func (a *App) handleImportKeys(msg tea.KeyMsg) []tea.Cmd {
	ip := &a.importPanel

	if ip.importing {
		return nil
	}

	switch msg.String() {
	case "up":
		if ip.cursor > 0 {
			ip.cursor--
			if ip.cursor < ip.offset {
				ip.offset = ip.cursor
			}
		} else if len(ip.files) > 0 {
			ip.cursor = len(ip.files) - 1
			if ip.cursor >= ip.offset+ip.visibleRows() {
				ip.offset = ip.cursor - ip.visibleRows() + 1
			}
		}
	case "down":
		if ip.cursor < len(ip.files)-1 {
			ip.cursor++
			if ip.cursor >= ip.offset+ip.visibleRows() {
				ip.offset++
			}
		} else if len(ip.files) > 0 {
			ip.cursor = 0
			ip.offset = 0
		}
	case ".", " ":
		if len(ip.files) > 0 && ip.cursor < len(ip.files) {
			ip.files[ip.cursor].Selected = !ip.files[ip.cursor].Selected
		}
	case "enter":
		if len(ip.files) > 0 && ip.cursor < len(ip.files) {
			f := ip.files[ip.cursor]
			if ip.previewPath == f.Path {
				a.player.Stop()
				ip.previewPath = ""
				a.nowPlay = nil
				a.songStarted = false
				a.right.SetTrack(nil)
			} else {
				a.cancelResolve()
				a.left.loadingStream = false
				if err := a.player.Play(f.Path); err != nil {
					a.setStatus(StatusErrStyle.Render("X " + err.Error()))
				} else {
					ip.previewPath = f.Path
					if f.Duration == 0 {
						f.Duration = storage.GetFileDuration(f.Path)
						ip.files[ip.cursor] = f
					}
					track := &domain.Track{
						ID:       "local:" + f.Path,
						Title:    f.Name,
						Duration: f.Duration,
					}
					a.nowPlay = track
					a.songStarted = true
					a.playerGen = a.player.Generation()
					a.right.SetTrack(track)
				}
			}
		}
	case "i":
		if ip.selectedCount() > 0 {
			return a.startImport()
		}
	case "r":
		ip.ScanImportDirs(a.downloadDir, a.left.plStore)
		ip.cursor = 0
		ip.offset = 0
		a.setStatus(StatusOKStyle.Render(fmt.Sprintf("> Rescanned: %d new files", len(ip.files))))
	}
	return nil
}

func (a *App) startImport() []tea.Cmd {
	ip := &a.importPanel
	store := a.left.plStore
	if store == nil {
		a.setStatus(StatusErrStyle.Render("X Database not initialized"))
		return nil
	}

	ip.importing = true
	ip.importDone = 0
	ip.importTotal = ip.selectedCount()

	return []tea.Cmd{
		ip.spinner.Tick,
		importCmd(store, ip.files),
	}
}

// handleImportDone xử lý khi import hoàn tất.
func (a *App) handleImportDone(msg ImportDoneMsg) {
	ip := &a.importPanel
	ip.importing = false

	a.setStatus(StatusOKStyle.Render(fmt.Sprintf("> Imported %d file(s)", msg.Imported)))
	a.left.scanLocalFiles()

	var remaining []ImportFile
	for _, f := range ip.files {
		if !f.Selected {
			remaining = append(remaining, f)
		}
	}
	ip.files = remaining
	ip.cursor = 0
	ip.offset = 0
}
