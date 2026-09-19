package transfer

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"som/internal/storage"
)

const (
	DefaultSessionTTL = 10 * time.Minute
	MaxFileSize       = int64(2) << 30 // 2 GiB
	tokenBytes        = 32
)

var (
	ErrSessionExpired = errors.New("transfer session expired")
	ErrUnauthorized   = errors.New("transfer unauthorized")
	ErrConflict       = errors.New("track already exists on target")
)

type ManifestEntry struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Artist    string `json:"artist"`
	Duration  int    `json:"duration"`
	Thumbnail string `json:"thumbnail,omitempty"`
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
	MD5       string `json:"md5"`
}

type Manifest struct {
	Version int             `json:"version"`
	Tracks  []ManifestEntry `json:"tracks"`
}

type PairResponse struct {
	Version int    `json:"version"`
	Type    string `json:"type"`
	Token   string `json:"token"`
}

type SessionInfo struct {
	URLs      []string
	ExpiresAt time.Time
	Paired    bool
}

type Session struct {
	db           *storage.DB
	rootDir      string
	server       *http.Server
	listener     net.Listener
	pairToken    string
	sessionToken string

	mu        sync.RWMutex
	pairedIP  string
	closed    bool
	expiresAt time.Time
	timer     *time.Timer
	closeOnce sync.Once
}

type fileEntry struct {
	ManifestEntry
	filenamePath string
}

func Start(db *storage.DB, rootDir string, ttl time.Duration) (*Session, error) {
	if db == nil {
		return nil, errors.New("transfer: nil storage db")
	}
	if rootDir == "" {
		return nil, errors.New("transfer: empty root directory")
	}
	if ttl <= 0 {
		ttl = DefaultSessionTTL
	}
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return nil, fmt.Errorf("transfer: create root: %w", err)
	}

	pairToken, err := randomToken()
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return nil, fmt.Errorf("transfer: listen: %w", err)
	}

	s := &Session{
		db:        db,
		rootDir:   filepath.Clean(rootDir),
		listener:  listener,
		pairToken: pairToken,
		expiresAt: time.Now().Add(ttl),
	}
	s.server = &http.Server{
		Handler:           s.handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      10 * time.Minute,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}
	s.timer = time.AfterFunc(ttl, s.Close)

	go func() {
		_ = s.server.Serve(listener)
	}()

	return s, nil
}

func (s *Session) Info() SessionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, port, _ := net.SplitHostPort(s.listener.Addr().String())
	urls := make([]string, 0, 4)
	if s.pairToken != "" {
		for _, ip := range privateIPv4s() {
			urls = append(urls, fmt.Sprintf("http://%s:%s/pair?token=%s", ip, port, s.pairToken))
		}
	}

	return SessionInfo{URLs: urls, ExpiresAt: s.expiresAt, Paired: s.sessionToken != ""}
}

func (s *Session) Close() {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()

		if s.timer != nil {
			s.timer.Stop()
		}
		if s.server != nil {
			_ = s.server.Close()
		}
	})
}

func (s *Session) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/pair", s.handlePair)
	mux.HandleFunc("/manifest", s.requireSession(s.handleManifest))
	mux.HandleFunc("/file/", s.requireSession(s.handleFile))
	mux.HandleFunc("/close", s.requireSession(s.handleClose))
	return noStore(mux)
}

func (s *Session) handlePair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || time.Now().After(s.expiresAt) {
		writeJSONError(w, http.StatusGone, ErrSessionExpired.Error())
		return
	}
	if s.pairToken == "" || s.pairToken != r.URL.Query().Get("token") {
		writeJSONError(w, http.StatusUnauthorized, "invalid pairing token")
		return
	}

	remoteIP := remoteIPOf(r)
	if s.pairedIP != "" && s.pairedIP != remoteIP {
		writeJSONError(w, http.StatusForbidden, "session already paired to another device")
		return
	}
	s.pairedIP = remoteIP

	if s.sessionToken == "" {
		token, err := randomToken()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to create session token")
			return
		}
		s.sessionToken = token
		s.pairToken = ""
	}

	s.expiresAt = time.Now().Add(DefaultSessionTTL)
	if s.timer != nil {
		s.timer.Reset(DefaultSessionTTL)
	}

	writeJSON(w, http.StatusOK, PairResponse{Version: 1, Type: "som-transfer", Token: s.sessionToken})
}

func (s *Session) requireSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		closed := s.closed
		expired := time.Now().After(s.expiresAt)
		token := s.sessionToken
		pairedIP := s.pairedIP
		s.mu.RUnlock()

		if closed || expired {
			writeJSONError(w, http.StatusGone, ErrSessionExpired.Error())
			return
		}
		if token == "" || r.Header.Get("X-SOM-Transfer-Token") != token {
			writeJSONError(w, http.StatusUnauthorized, ErrUnauthorized.Error())
			return
		}
		if pairedIP != "" && remoteIPOf(r) != pairedIP {
			writeJSONError(w, http.StatusForbidden, "device is not paired for this session")
			return
		}
		next(w, r)
	}
}

func (s *Session) handleManifest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	manifest, err := s.buildManifest()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, manifest)
}

func (s *Session) handleFile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.serveFile(w, r)
	case http.MethodPut:
		s.receiveFile(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Session) serveFile(w http.ResponseWriter, r *http.Request) {
	id, ok := trackIDFromPath(r.URL.Path)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid track id")
		return
	}

	entry, err := s.findEntry(id)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSONError(w, http.StatusNotFound, "track not found")
		} else {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	f, err := os.Open(entry.filenamePath)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "file not found")
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil || info.IsDir() {
		writeJSONError(w, http.StatusNotFound, "file not found")
		return
	}

	w.Header().Set("Content-Type", contentTypeFor(entry.Filename))
	w.Header().Set("X-SOM-Track-ID", entry.ID)
	w.Header().Set("X-SOM-MD5", entry.MD5)
	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(entry.Filename))
	http.ServeContent(w, r, entry.Filename, info.ModTime(), f)
}

func (s *Session) receiveFile(w http.ResponseWriter, r *http.Request) {
	id, ok := trackIDFromPath(r.URL.Path)
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid track id")
		return
	}
	if r.ContentLength < 0 {
		writeJSONError(w, http.StatusLengthRequired, "content length required")
		return
	}
	if r.ContentLength > MaxFileSize {
		writeJSONError(w, http.StatusRequestEntityTooLarge, "file too large")
		return
	}
	if r.Header.Get("Content-Range") != "" {
		writeJSONError(w, http.StatusNotImplemented, "resumable upload is not supported yet")
		return
	}

	if existing, _ := s.findEntry(id); existing != nil {
		writeJSONError(w, http.StatusConflict, ErrConflict.Error())
		return
	}

	filename := sanitizeFilename(r.Header.Get("X-SOM-Filename"), id)
	if !storage.IsSupportedAudio(filename) {
		writeJSONError(w, http.StatusUnsupportedMediaType, "unsupported audio format")
		return
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" || strings.ContainsAny(ext, "/\\") {
		writeJSONError(w, http.StatusBadRequest, "invalid file extension")
		return
	}

	finalPath := filepath.Join(s.rootDir, id+ext)
	if !pathInside(s.rootDir, finalPath) {
		writeJSONError(w, http.StatusBadRequest, "invalid destination")
		return
	}

	tmp, err := os.CreateTemp(s.rootDir, ".som-sync-*")
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "cannot create temporary file")
		return
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}
	defer cleanup()

	hash := md5.New()
	r.Body = http.MaxBytesReader(w, r.Body, MaxFileSize)
	written, err := io.Copy(io.MultiWriter(tmp, hash), r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "upload failed")
		return
	}
	if written != r.ContentLength {
		writeJSONError(w, http.StatusBadRequest, "incomplete upload")
		return
	}
	if err := tmp.Close(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to close temporary file")
		return
	}

	gotMD5 := hex.EncodeToString(hash.Sum(nil))
	expectedMD5 := strings.ToLower(strings.TrimSpace(r.Header.Get("X-SOM-MD5")))
	if expectedMD5 != "" && expectedMD5 != gotMD5 {
		writeJSONError(w, http.StatusBadRequest, "checksum mismatch")
		return
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to finalize file")
		return
	}

	metadata := map[string]any{
		"title":     strings.TrimSpace(r.Header.Get("X-SOM-Title")),
		"artist":    strings.TrimSpace(r.Header.Get("X-SOM-Artist")),
		"video_id":  id,
		"thumbnail": strings.TrimSpace(r.Header.Get("X-SOM-Thumbnail")),
	}
	data, _ := json.Marshal(metadata)
	sidecar := strings.TrimSuffix(finalPath, ext) + ".json"
	if err := os.WriteFile(sidecar, data, 0o600); err != nil {
		_ = os.Remove(finalPath)
		writeJSONError(w, http.StatusInternalServerError, "failed to write metadata")
		return
	}

	if _, err := s.db.ImportFromFilesystem(s.rootDir); err != nil {
		_ = os.Remove(finalPath)
		_ = os.Remove(sidecar)
		writeJSONError(w, http.StatusInternalServerError, "failed to import uploaded track")
		return
	}

	w.Header().Set("X-SOM-MD5", gotMD5)
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":       id,
		"filename": filepath.Base(finalPath),
		"size":     written,
		"md5":      gotMD5,
	})
}

func (s *Session) handleClose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "closing"})
	go s.Close()
}

func (s *Session) buildManifest() (Manifest, error) {
	files, err := s.db.ListAllLocalFilesSorted("name")
	if err != nil {
		return Manifest{}, err
	}

	tracks := make([]ManifestEntry, 0, len(files))
	for _, f := range files {
		if f.VideoID == "" || !pathInside(s.rootDir, f.Path) {
			continue
		}
		info, err := os.Stat(f.Path)
		if err != nil || info.IsDir() {
			continue
		}
		md5sum, err := fileMD5(f.Path)
		if err != nil {
			continue
		}
		tracks = append(tracks, ManifestEntry{
			ID:        f.VideoID,
			Title:     f.Name,
			Artist:    f.Artist,
			Duration:  f.Duration,
			Thumbnail: f.Thumbnail,
			Filename:  filepath.Base(f.Path),
			Size:      info.Size(),
			MD5:       md5sum,
		})
	}

	return Manifest{Version: 1, Tracks: tracks}, nil
}

func (s *Session) findEntry(id string) (*fileEntry, error) {
	f, err := s.db.GetLocalFileByVideoID(id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, os.ErrNotExist
	}
	if !pathInside(s.rootDir, f.Path) {
		return nil, errors.New("stored file path escapes transfer root")
	}

	info, err := os.Stat(f.Path)
	if err != nil || info.IsDir() {
		return nil, os.ErrNotExist
	}
	md5sum, err := fileMD5(f.Path)
	if err != nil {
		return nil, err
	}

	return &fileEntry{
		ManifestEntry: ManifestEntry{
			ID:        f.VideoID,
			Title:     f.Name,
			Artist:    f.Artist,
			Duration:  f.Duration,
			Thumbnail: f.Thumbnail,
			Filename:  filepath.Base(f.Path),
			Size:      info.Size(),
			MD5:       md5sum,
		},
		filenamePath: f.Path,
	}, nil
}

func trackIDFromPath(path string) (string, bool) {
	id := strings.TrimPrefix(path, "/file/")
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") || strings.Contains(id, "\\") || strings.Contains(id, "..") || len(id) > 128 {
		return "", false
	}
	return id, true
}

func sanitizeFilename(name, fallbackID string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "." || name == string(filepath.Separator) || name == "" {
		return fallbackID + ".m4a"
	}
	name = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', 0:
			return '-'
		default:
			return r
		}
	}, name)
	return name
}

func pathInside(root, path string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}

func fileMD5(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func contentTypeFor(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".m4a", ".mp4", ".alac":
		return "audio/mp4"
	case ".opus":
		return "audio/opus"
	case ".ogg":
		return "audio/ogg"
	case ".mp3":
		return "audio/mpeg"
	case ".flac":
		return "audio/flac"
	case ".wav":
		return "audio/wav"
	case ".aac":
		return "audio/aac"
	case ".webm":
		return "audio/webm"
	default:
		return "application/octet-stream"
	}
}

func remoteIPOf(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func privateIPv4s() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	seen := make(map[string]struct{})
	var out []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil || !ip.IsPrivate() {
				continue
			}
			key := ip.String()
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, key)
		}
	}
	return out
}

func randomToken() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("transfer: random token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
