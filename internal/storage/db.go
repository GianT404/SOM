package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

func Open() (*DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, "Music", "SOM_Downloads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dir, "som.db")
	conn, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	conn.SetMaxOpenConns(1)

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS playlists (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		created_at TEXT DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS playlist_tracks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		playlist_id TEXT NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
		track_id TEXT NOT NULL,
		title TEXT NOT NULL,
		artist TEXT DEFAULT '',
		duration INTEGER DEFAULT 0,
		is_local INTEGER DEFAULT 0,
		position INTEGER NOT NULL,
		UNIQUE(playlist_id, track_id)
	);

	CREATE TABLE IF NOT EXISTS local_files (
		path TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		artist TEXT DEFAULT '',
		duration INTEGER DEFAULT 0,
		video_id TEXT DEFAULT '',
		thumbnail TEXT DEFAULT '',
		file_size INTEGER DEFAULT 0,
		file_mtime TEXT DEFAULT '',
		lyrics_json TEXT DEFAULT '',
		created_at TEXT DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		track_id TEXT NOT NULL,
		track_type TEXT NOT NULL DEFAULT 'remote',
		title TEXT NOT NULL,
		artist TEXT DEFAULT '',
		played_at TEXT DEFAULT (datetime('now'))
	);

	CREATE INDEX IF NOT EXISTS idx_history_played_at ON history(played_at DESC);
	CREATE INDEX IF NOT EXISTS idx_history_track_id ON history(track_id);
	CREATE INDEX IF NOT EXISTS idx_playlist_tracks_playlist ON playlist_tracks(playlist_id, position);
	`
	if _, err := db.conn.Exec(schema); err != nil {
		return err
	}

	// FTS5 virtual table (separate statement — CREATE VIRTUAL TABLE can't be inside transactions on some builds).
_fts := `CREATE VIRTUAL TABLE IF NOT EXISTS local_files_fts USING fts5(
		name, artist, video_id,
		content='local_files',
		content_rowid='rowid'
	)`
	if _, err := db.conn.Exec(_fts); err != nil {
		return fmt.Errorf("create FTS table: %w", err)
	}

	// Triggers to keep FTS in sync.
	triggers := `
	CREATE TRIGGER IF NOT EXISTS local_files_ai AFTER INSERT ON local_files BEGIN
		INSERT INTO local_files_fts(rowid, name, artist, video_id)
		VALUES (new.rowid, new.name, new.artist, new.video_id);
	END;

	CREATE TRIGGER IF NOT EXISTS local_files_ad AFTER DELETE ON local_files BEGIN
		INSERT INTO local_files_fts(local_files_fts, rowid, name, artist, video_id)
		VALUES ('delete', old.rowid, old.name, old.artist, old.video_id);
	END;

	CREATE TRIGGER IF NOT EXISTS local_files_au AFTER UPDATE ON local_files BEGIN
		INSERT INTO local_files_fts(local_files_fts, rowid, name, artist, video_id)
		VALUES ('delete', old.rowid, old.name, old.artist, old.video_id);
		INSERT INTO local_files_fts(rowid, name, artist, video_id)
		VALUES (new.rowid, new.name, new.artist, new.video_id);
	END;
	`
	if _, err := db.conn.Exec(triggers); err != nil {
		return fmt.Errorf("create FTS triggers: %w", err)
	}

	return nil
}

func now() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05")
}
