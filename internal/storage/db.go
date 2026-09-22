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

func (db *DB) RenameLocalFile(oldPath, newPath, newName string) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE local_files SET path = ?, name = ? WHERE path = ?", newPath, newName, oldPath)
	if err != nil {
		return err
	}

	// Chỉ cập nhật khóa liên kết, mọi metadata tự động được JOIN lên UI
	_, err = tx.Exec("UPDATE playlist_tracks SET track_id = ? WHERE track_id = ?", newPath, oldPath)
	if err != nil {
		return err
	}

	return tx.Commit()
}

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
	if err := runMigrations(db.conn); err != nil {
		return err
	}
	_fts := `CREATE VIRTUAL TABLE IF NOT EXISTS local_files_fts USING fts5(
		name, artist, video_id, content='local_files', content_rowid='rowid'
	);`
	if _, err := db.conn.Exec(_fts); err != nil {
		return fmt.Errorf("create FTS table: %w", err)
	}

	triggers := `
	CREATE TRIGGER IF NOT EXISTS local_files_ai AFTER INSERT ON local_files BEGIN
	  INSERT INTO local_files_fts(rowid, name, artist, video_id) VALUES (new.rowid, new.name, new.artist, new.video_id);
	END;
	CREATE TRIGGER IF NOT EXISTS local_files_ad AFTER DELETE ON local_files BEGIN
	  INSERT INTO local_files_fts(local_files_fts, rowid, name, artist, video_id) VALUES ('delete', old.rowid, old.name, old.artist, old.video_id);
	END;
	CREATE TRIGGER IF NOT EXISTS local_files_au AFTER UPDATE ON local_files BEGIN
	  INSERT INTO local_files_fts(local_files_fts, rowid, name, artist, video_id) VALUES ('delete', old.rowid, old.name, old.artist, old.video_id);
	  INSERT INTO local_files_fts(rowid, name, artist, video_id) VALUES (new.rowid, new.name, new.artist, new.video_id);
	END;`
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

const createLyricsCacheTable = `
CREATE TABLE IF NOT EXISTS lyrics_cache (
    cache_key TEXT PRIMARY KEY,
    lyrics_json TEXT NOT NULL,
    expires_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_lyrics_cache_expires ON lyrics_cache(expires_at);
`

// Cleanup job: Chạy cái này mỗi khi khởi động app để dọn rác
func (db *DB) CleanupExpiredLyrics() error {
	_, err := db.conn.Exec(`DELETE FROM lyrics_cache WHERE expires_at < CURRENT_TIMESTAMP`)
	return err
}

func (db *DB) GetLyricsCache(key string) (string, error) {
	var jsonStr string
	// Chỉ lấy nếu chưa hết hạn
	err := db.conn.QueryRow(`
        SELECT lyrics_json FROM lyrics_cache 
        WHERE cache_key = ? AND expires_at > CURRENT_TIMESTAMP
    `, key).Scan(&jsonStr)
	if err != nil {
		return "", err
	}
	return jsonStr, nil
}

func (db *DB) PutLyricsCache(key string, lyricsJSON string, isEmpty bool) error {
	// Nếu có lời: cache 7 ngày. Nếu rỗng/lỗi: cache 1 giờ.
	hours := 24 * 7
	if isEmpty {
		hours = 1
	}
	expiresAt := time.Now().Add(time.Duration(hours) * time.Hour).UTC().Format(time.RFC3339)

	_, err := db.conn.Exec(`
        INSERT INTO lyrics_cache (cache_key, lyrics_json, expires_at)
        VALUES (?, ?, ?)
        ON CONFLICT(cache_key) DO UPDATE SET 
            lyrics_json = excluded.lyrics_json,
            expires_at = excluded.expires_at
    `, key, lyricsJSON, expiresAt)
	return err
}
