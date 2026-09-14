package storage

import (
	"fmt"
	"strconv"
	"time"
)

type Playlist struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Tracks []PlaylistTrack `json:"tracks"`
}

type PlaylistTrack struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Duration  int    `json:"duration"`
	Thumbnail string `json:"thumbnail"`
	Path      string `json:"path"`
}

// ── Playlist CRUD ────────────────────────────────────────────────

func (db *DB) CreatePlaylist(name string) (Playlist, error) {
	id := "pl_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	_, err := db.conn.Exec("INSERT INTO playlists (id, name) VALUES (?, ?)", id, name)
	if err != nil {
		return Playlist{}, err
	}
	return Playlist{ID: id, Name: name}, nil
}

func (db *DB) DeletePlaylist(id string) error {
	_, err := db.conn.Exec("DELETE FROM playlists WHERE id = ?", id)
	return err
}

func (db *DB) RenamePlaylist(id, name string) error {
	_, err := db.conn.Exec("UPDATE playlists SET name = ? WHERE id = ?", name, id)
	return err
}

func (db *DB) ListPlaylists() ([]Playlist, error) {
	rows, err := db.conn.Query("SELECT id, name FROM playlists ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var playlists []Playlist
	for rows.Next() {
		var p Playlist
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, err
		}
		playlists = append(playlists, p)
	}
	return playlists, rows.Err()
}

// ── Playlist Tracks ──────────────────────────────────────────────

func (db *DB) GetPlaylistTracks(playlistID string) ([]PlaylistTrack, error) {
	query := `
		SELECT 
			pt.track_id, 
			lf.name, 
			lf.artist, 
			lf.duration, 
			lf.thumbnail, 
			lf.path 
		FROM playlist_tracks pt
		JOIN local_files lf ON pt.track_id = lf.path
		WHERE pt.playlist_id = ? 
		ORDER BY pt.position
	`
	rows, err := db.conn.Query(query, playlistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []PlaylistTrack
	for rows.Next() {
		var t PlaylistTrack
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Duration, &t.Thumbnail, &t.Path); err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func (db *DB) AddTrackToPlaylist(playlistID string, trackID string) error {
	res, err := db.conn.Exec(`
		INSERT INTO playlist_tracks (playlist_id, track_id, position)
		VALUES (?, ?, COALESCE((SELECT MAX(position) + 1 FROM playlist_tracks WHERE playlist_id = ?), 0))
		ON CONFLICT(playlist_id, track_id) DO NOTHING`,
		playlistID, trackID, playlistID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("track already in playlist")
	}
	return nil
}
func (db *DB) RemoveTrackFromPlaylist(playlistID, trackID string) error {
	_, err := db.conn.Exec(
		"DELETE FROM playlist_tracks WHERE playlist_id = ? AND track_id = ?",
		playlistID, trackID,
	)
	return err
}

// ── All playlists with tracks (bulk load) ────────────────────────

func (db *DB) LoadAllPlaylists() ([]Playlist, error) {
	playlists, err := db.ListPlaylists()
	if err != nil {
		return nil, err
	}
	for i := range playlists {
		tracks, err := db.GetPlaylistTracks(playlists[i].ID)
		if err != nil {
			return nil, err
		}
		playlists[i].Tracks = tracks
	}
	return playlists, nil
}
