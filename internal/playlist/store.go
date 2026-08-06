package playlist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type Track struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Duration int    `json:"duration"`
	IsLocal  bool   `json:"is_local"`
}

type Playlist struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Tracks []Track `json:"tracks"`
}

type Store struct {
	filePath string
	mu       sync.Mutex
}

func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, "Music", "SOM_Downloads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{
		filePath: filepath.Join(dir, "playlists.json"),
	}, nil
}

func (s *Store) Load() ([]Playlist, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func (s *Store) load() ([]Playlist, error) {
	data, err := os.ReadFile(s.filePath)
	if os.IsNotExist(err) {
		return []Playlist{}, nil
	}
	if err != nil {
		return nil, err
	}
	var playlists []Playlist
	if err := json.Unmarshal(data, &playlists); err != nil {
		return nil, err
	}
	return playlists, nil
}

func (s *Store) Save(playlists []Playlist) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save(playlists)
}

// save ghi atomic: ghi ra file tạm cùng thư mục rồi rename đè lên file đích.
// os.Rename là 1 syscall duy nhất trên cùng filesystem, nên nếu process crash
// hoặc mất điện giữa lúc ghi, playlists.json cũ vẫn nguyên vẹn (không bao giờ
// ở trạng thái ghi dở dang) thay vì bị hỏng như os.WriteFile trực tiếp trước đây.
func (s *Store) save(playlists []Playlist) error {
	data, err := json.MarshalIndent(playlists, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.filePath)
	tmp, err := os.CreateTemp(dir, ".playlists-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, s.filePath); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

func (s *Store) CreatePlaylist(name string) (Playlist, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	playlists, err := s.load()
	if err != nil {
		return Playlist{}, err
	}

	pl := Playlist{
		ID:   "pl_" + strconv.FormatInt(time.Now().UnixNano(), 10),
		Name: name,
	}

	playlists = append(playlists, pl)
	if err := s.save(playlists); err != nil {
		return Playlist{}, err
	}
	return pl, nil
}

func (s *Store) DeletePlaylist(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	playlists, err := s.load()
	if err != nil {
		return err
	}

	var filtered []Playlist
	for _, p := range playlists {
		if p.ID != id {
			filtered = append(filtered, p)
		}
	}

	return s.save(filtered)
}

func (s *Store) AddTrack(playlistID string, track Track) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	playlists, err := s.load()
	if err != nil {
		return err
	}

	for i, p := range playlists {
		if p.ID == playlistID {
			for _, t := range p.Tracks {
				if t.ID == track.ID {
					return fmt.Errorf("bài hát đã có trong playlist")
				}
			}
			playlists[i].Tracks = append(playlists[i].Tracks, track)
			return s.save(playlists)
		}
	}
	return fmt.Errorf("không tìm thấy playlist")
}

func (s *Store) RemoveTrack(playlistID string, trackID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	playlists, err := s.load()
	if err != nil {
		return err
	}

	for i, p := range playlists {
		if p.ID == playlistID {
			var filtered []Track
			for _, t := range p.Tracks {
				if t.ID != trackID {
					filtered = append(filtered, t)
				}
			}
			playlists[i].Tracks = filtered
			return s.save(playlists)
		}
	}
	return fmt.Errorf("không tìm thấy playlist")
}
