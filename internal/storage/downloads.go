package storage

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type LocalFile struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Artist    string `json:"artist"`
	Duration  int    `json:"duration"`
	VideoID   string `json:"video_id"`
	Thumbnail string `json:"thumbnail"`
	FileSize  int64  `json:"file_size"`
	FileMTime string `json:"file_mtime"`
	CreatedAt string `json:"created_at"`
}

type LocalFileMeta struct {
	Artist     string
	Title      string
	VideoID    string
	Thumbnail  string
	LyricsJSON string
}

// UpsertLocalFile inserts or updates a local file record.
func (db *DB) UpsertLocalFile(f LocalFile) error {
	_, err := db.conn.Exec(`
		INSERT INTO local_files (path, name, artist, duration, video_id, thumbnail, file_size, file_mtime)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			name=excluded.name, artist=excluded.artist, duration=excluded.duration,
			video_id=excluded.video_id, thumbnail=excluded.thumbnail,
			file_size=excluded.file_size, file_mtime=excluded.file_mtime
	`, f.Path, f.Name, f.Artist, f.Duration, f.VideoID, f.Thumbnail, f.FileSize, f.FileMTime)
	return err
}

// UpsertLocalFileWithMeta also stores the sidecar JSON (lyrics + metadata).
func (db *DB) UpsertLocalFileWithMeta(f LocalFile, meta *LocalFileMeta) error {
	lyricsJSON := ""
	if meta != nil {
		lyricsJSON = meta.LyricsJSON
		artist := meta.Artist
		if artist != "" {
			f.Artist = artist
		}
		if meta.Title != "" {
			f.Name = meta.Title
		}
		if meta.VideoID != "" {
			f.VideoID = meta.VideoID
		}
		if meta.Thumbnail != "" {
			f.Thumbnail = meta.Thumbnail
		}
	}
	_, err := db.conn.Exec(`
		INSERT INTO local_files (path, name, artist, duration, video_id, thumbnail, file_size, file_mtime, lyrics_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			name=excluded.name, artist=excluded.artist, duration=excluded.duration,
			video_id=excluded.video_id, thumbnail=excluded.thumbnail,
			file_size=excluded.file_size, file_mtime=excluded.file_mtime,
			lyrics_json=excluded.lyrics_json
	`, f.Path, f.Name, f.Artist, f.Duration, f.VideoID, f.Thumbnail, f.FileSize, f.FileMTime, lyricsJSON)
	return err
}

func (db *DB) DeleteLocalFile(path string) error {
	_, err := db.conn.Exec("DELETE FROM local_files WHERE path = ?", path)
	return err
}

func (db *DB) GetLocalFile(path string) (*LocalFile, error) {
	var f LocalFile
	err := db.conn.QueryRow(
		"SELECT path, name, artist, duration, video_id, thumbnail, file_size, file_mtime FROM local_files WHERE path = ?", path,
	).Scan(&f.Path, &f.Name, &f.Artist, &f.Duration, &f.VideoID, &f.Thumbnail, &f.FileSize, &f.FileMTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (db *DB) GetLocalFileMeta(path string) (*LocalFileMeta, error) {
	var m LocalFileMeta
	err := db.conn.QueryRow(
		"SELECT artist, video_id, thumbnail, lyrics_json FROM local_files WHERE path = ?", path,
	).Scan(&m.Artist, &m.VideoID, &m.Thumbnail, &m.LyricsJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (db *DB) GetLocalFileLyrics(path string) (string, error) {
	var s string
	err := db.conn.QueryRow("SELECT lyrics_json FROM local_files WHERE path = ?", path).Scan(&s)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return s, err
}

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

	// Update playlist tracks that reference the old local path.
	_, err = tx.Exec(
		"UPDATE playlist_tracks SET track_id = ?, title = ? WHERE track_id = ?",
		"local:"+newPath, newName, "local:"+oldPath,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) ListAllLocalFiles() ([]LocalFile, error) {
	return db.ListAllLocalFilesSorted("name")
}

func (db *DB) ListAllLocalFilesSorted(sort string) ([]LocalFile, error) {
	var orderBy string
	switch sort {
	case "date":
		orderBy = "created_at DESC"
	case "duration":
		orderBy = "duration DESC"
	default:
		orderBy = "name COLLATE NOCASE"
	}
	rows, err := db.conn.Query(
		"SELECT path, name, artist, duration, video_id, thumbnail, file_size, file_mtime, created_at FROM local_files ORDER BY "+orderBy,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []LocalFile
	for rows.Next() {
		var f LocalFile
		if err := rows.Scan(&f.Path, &f.Name, &f.Artist, &f.Duration, &f.VideoID, &f.Thumbnail, &f.FileSize, &f.FileMTime, &f.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (db *DB) CountLocalFiles() int {
	var n int
	_ = db.conn.QueryRow("SELECT COUNT(*) FROM local_files").Scan(&n)
	return n
}

func (db *DB) IsDownloaded(videoID, title string, duration int) bool {
	if videoID != "" {
		var n int
		_ = db.conn.QueryRow("SELECT COUNT(*) FROM local_files WHERE video_id = ?", videoID).Scan(&n)
		if n > 0 {
			return true
		}
	}
	if title == "" {
		return false
	}
	normalized := normalizeTitle(title)
	var n int
	_ = db.conn.QueryRow(
		"SELECT COUNT(*) FROM local_files WHERE REPLACE(REPLACE(REPLACE(LOWER(name),' ',''),'.',''),'-','') = ? AND ABS(duration - ?) <= 2",
		normalized, duration,
	).Scan(&n)
	return n > 0
}

func normalizeTitle(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, "-", "")
	return s
}

// ── FTS Search ───────────────────────────────────────────────────

func (db *DB) SearchLocalFiles(query string) ([]LocalFile, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return db.ListAllLocalFiles()
	}

	// Build FTS5 query: each token is a separate term.
	tokens := strings.Fields(strings.ToLower(query))
	var conditions []string
	var args []any
	for _, tok := range tokens {
		conditions = append(conditions, "local_files_fts MATCH ?")
		args = append(args, tok+"*") // prefix match
	}

	sqlQuery := fmt.Sprintf(`
		SELECT lf.path, lf.name, lf.artist, lf.duration, lf.video_id, lf.thumbnail, lf.file_size, lf.file_mtime
		FROM local_files lf
		INNER JOIN local_files_fts ON local_files_fts.rowid = lf.rowid
		WHERE %s
		ORDER BY rank
		LIMIT 500
	`, strings.Join(conditions, " AND "))

	rows, err := db.conn.Query(sqlQuery, args...)
	if err != nil {
		// Fallback to LIKE search if FTS fails.
		return db.searchLocalFilesLike(query)
	}
	defer rows.Close()

	var files []LocalFile
	for rows.Next() {
		var f LocalFile
		if err := rows.Scan(&f.Path, &f.Name, &f.Artist, &f.Duration, &f.VideoID, &f.Thumbnail, &f.FileSize, &f.FileMTime); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (db *DB) searchLocalFilesLike(query string) ([]LocalFile, error) {
	tokens := strings.Fields(strings.ToLower(query))
	var conditions []string
	var args []any
	for _, tok := range tokens {
		conditions = append(conditions, "(LOWER(name) LIKE ? OR LOWER(artist) LIKE ?)")
		like := "%" + tok + "%"
		args = append(args, like, like)
	}

	sqlQuery := fmt.Sprintf(`
		SELECT path, name, artist, duration, video_id, thumbnail, file_size, file_mtime
		FROM local_files
		WHERE %s
		ORDER BY name COLLATE NOCASE
		LIMIT 500
	`, strings.Join(conditions, " AND "))

	rows, err := db.conn.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []LocalFile
	for rows.Next() {
		var f LocalFile
		if err := rows.Scan(&f.Path, &f.Name, &f.Artist, &f.Duration, &f.VideoID, &f.Thumbnail, &f.FileSize, &f.FileMTime); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

// ── Filesystem import (one-time migration) ───────────────────────

func isSupportedAudio(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".opus") ||
		strings.HasSuffix(lower, ".mp3") ||
		strings.HasSuffix(lower, ".mp4")
}

func localFileSidecar(path string) string {
	ext := filepath.Ext(path)
	return strings.TrimSuffix(path, ext) + ".json"
}

func getFileDuration(path string) int {
	cmd := exec.Command("ffprobe", "-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0
	}
	sec, err := strconv.ParseFloat(strings.TrimSpace(out.String()), 64)
	if err != nil {
		return 0
	}
	return int(sec)
}

// ImportFromFilesystem scans the download directory and imports any audio files
// that are not yet in the database. Files already in the DB (matched by path)
// are skipped without re-probing duration.
func (db *DB) ImportFromFilesystem(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}

	// Build set of already-imported paths.
	existing := make(map[string]bool)
	rows, err := db.conn.Query("SELECT path FROM local_files")
	if err == nil {
		for rows.Next() {
			var p string
			if rows.Scan(&p) == nil {
				existing[p] = true
			}
		}
		rows.Close()
	}

	imported := 0
	for _, e := range entries {
		if e.IsDir() || !isSupportedAudio(e.Name()) {
			continue
		}
		localPath := filepath.Join(dir, e.Name())
		if existing[localPath] {
			continue
		}
		ext := filepath.Ext(e.Name())
		baseName := strings.TrimSuffix(localPath, ext)
		name := strings.TrimSuffix(e.Name(), ext)
		artist := ""
		videoID := ""
		thumbnail := ""
		var lyricsJSON string

		jsonPath := baseName + ".json"
		if data, err := os.ReadFile(jsonPath); err == nil {
			var lr struct {
				Artist    string `json:"artist"`
				Title     string `json:"title"`
				VideoID   string `json:"video_id"`
				Thumbnail string `json:"thumbnail"`
			}
			if json.Unmarshal(data, &lr) == nil {
				artist = lr.Artist
				videoID = lr.VideoID
				thumbnail = lr.Thumbnail
				if lr.Title != "" {
					name = lr.Title
				}
				lyricsJSON = string(data)
			}
		}
		if thumbnail == "" {
			imgPath := baseName + ".jpg"
			if _, errStat := os.Stat(imgPath); errStat == nil {
				absPath, _ := filepath.Abs(imgPath)
				thumbnail = "file://" + absPath
			}
		}

		info, err := os.Stat(localPath)
		if err != nil {
			continue
		}
		fileSize := info.Size()
		fileMTime := info.ModTime().Format(time.RFC3339)

		duration := getFileDuration(localPath)

		f := LocalFile{
			Path:      localPath,
			Name:      name,
			Artist:    artist,
			Duration:  duration,
			VideoID:   videoID,
			Thumbnail: thumbnail,
			FileSize:  fileSize,
			FileMTime: fileMTime,
		}
		meta := &LocalFileMeta{
			Artist:     artist,
			Title:      name,
			VideoID:    videoID,
			Thumbnail:  thumbnail,
			LyricsJSON: lyricsJSON,
		}
		if err := db.UpsertLocalFileWithMeta(f, meta); err != nil {
			continue
		}
		imported++
	}
	return imported, nil
}
