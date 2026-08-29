package storage

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
	dir  string
}

// DefaultDir returns the default download directory (~/.local/share/som).
func DefaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", "som")
	}
	return filepath.Join(home, ".local", "share", "som")
}

// legacyDirs returns previous default directories in reverse chronological order.
func legacyDirs(home string) []string {
	return []string{
		filepath.Join(home, "Music", "SOM_Downloads"),
	}
}

func Open(dir string) (*DB, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dir, "som.db")
	conn, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	conn.SetMaxOpenConns(1)

	db := &DB{conn: conn, dir: dir}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func (db *DB) Dir() string { return db.dir }

func MigrateFromLegacy(dir string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	for _, legacy := range legacyDirs(home) {
		if legacy == dir {
			continue
		}
		if _, err := os.Stat(legacy); os.IsNotExist(err) {
			continue
		}

		log.Printf("[storage] migrating from legacy dir %s -> %s", legacy, dir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create target dir: %w", err)
		}

		// Move files (including som.db) from legacy to target.
		entries, err := os.ReadDir(legacy)
		if err != nil {
			return err
		}

		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			src := filepath.Join(legacy, e.Name())
			dst := filepath.Join(dir, e.Name())
			if _, err := os.Stat(dst); err == nil {
				continue
			}
			if err := os.Rename(src, dst); err != nil {
				log.Printf("[storage] migrate: rename %s -> %s failed: %v", src, dst, err)
				if err := copyFile(src, dst); err != nil {
					log.Printf("[storage] migrate: copy %s -> %s failed: %v", src, dst, err)
					continue
				}
				os.Remove(src)
			}
		}

		// Update paths in the moved DB.
		dbPath := filepath.Join(dir, "som.db")
		conn, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL")
		if err == nil {
			legacyPrefix := legacy + "/"
			newPrefix := dir + "/"
			_, _ = conn.Exec("UPDATE local_files SET path = REPLACE(path, ?, ?) WHERE path LIKE ?",
				legacyPrefix, newPrefix, legacyPrefix+"%")
			_, _ = conn.Exec("UPDATE local_files SET thumbnail = REPLACE(thumbnail, ?, ?) WHERE thumbnail LIKE ?",
				"file://"+legacyPrefix, "file://"+newPrefix, "file://"+legacyPrefix+"%")
			_, _ = conn.Exec("UPDATE playlist_tracks SET track_id = REPLACE(track_id, ?, ?) WHERE track_id LIKE ?",
				"local:"+legacyPrefix, "local:"+newPrefix, "local:"+legacyPrefix+"%")
			conn.Close()
		}

		os.Remove(filepath.Join(legacy, ".query_cache"))
		os.Remove(legacy)
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
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

	CREATE INDEX IF NOT EXISTS idx_playlist_tracks_playlist ON playlist_tracks(playlist_id, position);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
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

func (db *DB) GetSetting(key string) string {
	var v string
	err := db.conn.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&v)
	if err != nil {
		return ""
	}
	return v
}

func (db *DB) SetSetting(key, value string) {
	_, _ = db.conn.Exec("INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", key, value)
}
