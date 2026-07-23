package playlist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

// Load đọc tất cả playlist từ file
func (s *Store) Load() ([]Playlist, error) {
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
	data, err := json.MarshalIndent(playlists, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0o644)
}

func (s *Store) CreatePlaylist(name string) (Playlist, error) {
	playlists, err := s.Load()
	if err != nil {
		return Playlist{}, err
	}

	pl := Playlist{
		ID:   "pl_" + strconv.FormatInt(time.Now().UnixNano(), 10),
		Name: name,
	}

	playlists = append(playlists, pl)
	if err := s.Save(playlists); err != nil {
		return Playlist{}, err
	}
	return pl, nil
}

func (s *Store) DeletePlaylist(id string) error {
	playlists, err := s.Load()
	if err != nil {
		return err
	}

	var filtered []Playlist
	for _, p := range playlists {
		if p.ID != id {
			filtered = append(filtered, p)
		}
	}

	return s.Save(filtered)
}

func (s *Store) AddTrack(playlistID string, track Track) error {
	playlists, err := s.Load()
	if err != nil {
		return err
	}

	for i, p := range playlists {
		if p.ID == playlistID {
			// Kiểm tra trùng lặp
			for _, t := range p.Tracks {
				if t.ID == track.ID {
					return fmt.Errorf("bài hát đã có trong playlist")
				}
			}
			playlists[i].Tracks = append(playlists[i].Tracks, track)
			return s.Save(playlists)
		}
	}
	return fmt.Errorf("không tìm thấy playlist")
}

func (s *Store) RemoveTrack(playlistID string, trackID string) error {
	playlists, err := s.Load()
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
			return s.Save(playlists)
		}
	}
	return fmt.Errorf("không tìm thấy playlist")
}
