package storage

type HistoryEntry struct {
	ID        int64  `json:"id"`
	TrackID   string `json:"track_id"`
	TrackType string `json:"track_type"` // "remote" or "local"
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	PlayedAt  string `json:"played_at"`
}

func (db *DB) RecordPlay(trackID, trackType, title, artist string) error {
	_, err := db.conn.Exec(
		"INSERT INTO history (track_id, track_type, title, artist) VALUES (?, ?, ?, ?)",
		trackID, trackType, title, artist,
	)
	return err
}

func (db *DB) GetRecentHistory(limit int) ([]HistoryEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.conn.Query(
		"SELECT id, track_id, track_type, title, artist, played_at FROM history ORDER BY id DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		if err := rows.Scan(&e.ID, &e.TrackID, &e.TrackType, &e.Title, &e.Artist, &e.PlayedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (db *DB) ClearHistory() error {
	_, err := db.conn.Exec("DELETE FROM history")
	return err
}
