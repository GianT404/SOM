package storage

import (
	"database/sql"
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
	ID       string `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Duration int    `json:"duration"`
	IsLocal  bool   `json:"is_local"`
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

func (db *DB) GetPlaylist(id string) (*Playlist, error) {
	var p Playlist
	err := db.conn.QueryRow("SELECT id, name FROM playlists WHERE id = ?", id).Scan(&p.ID, &p.Name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	tracks, err := db.GetPlaylistTracks(id)
	if err != nil {
		return nil, err
	}
	p.Tracks = tracks
	return &p, nil
}

// ── Playlist Tracks ──────────────────────────────────────────────

func (db *DB) GetPlaylistTracks(playlistID string) ([]PlaylistTrack, error) {
	rows, err := db.conn.Query(
		"SELECT track_id, title, artist, duration, is_local FROM playlist_tracks WHERE playlist_id = ? ORDER BY position",
		playlistID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []PlaylistTrack
	for rows.Next() {
		var t PlaylistTrack
		var isLocal int
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.Duration, &isLocal); err != nil {
			return nil, err
		}
		t.IsLocal = isLocal == 1
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func (db *DB) AddTrackToPlaylist(playlistID string, track PlaylistTrack) error {
	isLocal := 0
	if track.IsLocal {
		isLocal = 1
	}

	res, err := db.conn.Exec(`
		INSERT INTO playlist_tracks (playlist_id, track_id, title, artist, duration, is_local, position)
		VALUES (?, ?, ?, ?, ?, ?,
			COALESCE((SELECT MAX(position) + 1 FROM playlist_tracks WHERE playlist_id = ?), 0))
		ON CONFLICT(playlist_id, track_id) DO NOTHING`,
		playlistID, track.ID, track.Title, track.Artist, track.Duration, isLocal, playlistID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("bài hát đã có trong playlist")
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
