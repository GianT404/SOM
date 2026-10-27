package storage

import (
	"database/sql"
	"fmt"
	"log"
)

type migration struct {
	version int
	up      func(tx *sql.Tx) error
}

var migrations = []migration{
	{
		version: 1,
		up: func(tx *sql.Tx) error {
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
			);`
			_, err := tx.Exec(schema)
			return err
		},
	},
	{
		// V2: Chuẩn hóa dữ liệu - Tái cấu trúc bảng playlist_tracks thành bảng nối (Junction Table)
		version: 2,
		up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
			CREATE TABLE playlist_tracks_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				playlist_id TEXT NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
				track_id TEXT NOT NULL REFERENCES local_files(path) ON DELETE CASCADE,
				position INTEGER NOT NULL,
				UNIQUE(playlist_id, track_id)
			);`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`
			INSERT INTO playlist_tracks_new (playlist_id, track_id, position)
				SELECT pt.playlist_id, REPLACE(pt.track_id, 'local:', ''), pt.position
				FROM playlist_tracks AS pt
				JOIN playlists AS p ON p.id = pt.playlist_id
				JOIN local_files AS lf ON lf.path = REPLACE(pt.track_id, 'local:', '')
				WHERE pt.track_id LIKE 'local:%';`)
			if err != nil {
				return err
			}

			if _, err := tx.Exec("DROP TABLE playlist_tracks;"); err != nil {
				return err
			}
			if _, err := tx.Exec("ALTER TABLE playlist_tracks_new RENAME TO playlist_tracks;"); err != nil {
				return err
			}

			_, err = tx.Exec("CREATE INDEX IF NOT EXISTS idx_playlist_tracks_playlist ON playlist_tracks(playlist_id, position);")
			return err
		},
	},
	{
		// V3: Enforce foreign keys and cascade local-file path renames.
		// Invalid historical links are omitted while rebuilding the join table.
		version: 3,
		up: func(tx *sql.Tx) error {
			if _, err := tx.Exec(`
				CREATE TABLE playlist_tracks_v3 (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
					playlist_id TEXT NOT NULL REFERENCES playlists(id) ON DELETE CASCADE ON UPDATE CASCADE,
					track_id TEXT NOT NULL REFERENCES local_files(path) ON DELETE CASCADE ON UPDATE CASCADE,
					position INTEGER NOT NULL,
					UNIQUE(playlist_id, track_id)
				);`); err != nil {
				return err
			}

			if _, err := tx.Exec(`
				INSERT INTO playlist_tracks_v3 (id, playlist_id, track_id, position)
				SELECT pt.id, pt.playlist_id, pt.track_id, pt.position
				FROM playlist_tracks AS pt
				JOIN playlists AS p ON p.id = pt.playlist_id
				JOIN local_files AS lf ON lf.path = pt.track_id;`); err != nil {
				return err
			}

			if _, err := tx.Exec("DROP TABLE playlist_tracks;"); err != nil {
				return err
			}
			if _, err := tx.Exec("ALTER TABLE playlist_tracks_v3 RENAME TO playlist_tracks;"); err != nil {
				return err
			}
			_, err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_playlist_tracks_playlist ON playlist_tracks(playlist_id, position);")
			return err
		},
	},
}

func runMigrations(db *sql.DB) error {
	var currentVersion int
	if err := db.QueryRow("PRAGMA user_version").Scan(&currentVersion); err != nil {
		return fmt.Errorf("failed to read user_version: %w", err)
	}

	for _, m := range migrations {
		if m.version > currentVersion {
			log.Printf("[storage] Running migration v%d...", m.version)
			tx, err := db.Begin()
			if err != nil {
				return fmt.Errorf("failed to begin tx for migration v%d: %w", m.version, err)
			}

			if err := m.up(tx); err != nil {
				tx.Rollback()
				return fmt.Errorf("migration v%d failed: %w", m.version, err)
			}

			if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", m.version)); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to bump user_version to %d: %w", m.version, err)
			}

			if err := tx.Commit(); err != nil {
				return fmt.Errorf("failed to commit migration v%d: %w", m.version, err)
			}
			log.Printf("[storage] Migration v%d applied successfully.", m.version)
		}
	}
	return nil
}
