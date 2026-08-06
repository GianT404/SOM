package backend

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	ErrDeviceBanned    = errors.New("device is banned")
	ErrInvalidDeviceID = errors.New("invalid device id")
)

var deviceIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)

// device lưu trạng thái một device đã đăng ký.
type device struct {
	token     string
	createdAt time.Time
	lastSeen  time.Time
}

// DeviceRegistry quản lý device token trong bộ nhớ. Không bền vững qua restart,
// nhưng app tự đăng ký lại (idempotent theo device_id) khi cần.
type DeviceRegistry struct {
	mu      sync.Mutex
	devices map[string]*device // deviceID -> device
	tokens  map[string]string  // token -> deviceID
	banned  map[string]bool
}

func NewDeviceRegistry() *DeviceRegistry {
	r := &DeviceRegistry{
		devices: make(map[string]*device),
		tokens:  make(map[string]string),
		banned:  make(map[string]bool),
	}
	go r.cleanupLoop()
	return r
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (r *DeviceRegistry) cleanupLoop() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		r.mu.Lock()
		for id, d := range r.devices {
			if time.Since(d.lastSeen) > 90*24*time.Hour {
				delete(r.tokens, d.token)
				delete(r.devices, id)
			}
		}
		r.mu.Unlock()
	}
}

// ValidDeviceID kiểm tra định dạng device_id được chấp nhận.
func (r *DeviceRegistry) ValidDeviceID(id string) bool {
	return deviceIDRe.MatchString(id)
}

// Register trả về token cho deviceID. Idempotent: cùng ID trả lại token cũ
// (trừ khi device đã bị ban hoặc hết hạn 90 ngày không hoạt động).
func (r *DeviceRegistry) Register(deviceID string) (string, error) {
	if !r.ValidDeviceID(deviceID) {
		return "", ErrInvalidDeviceID
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.banned[deviceID] {
		return "", ErrDeviceBanned
	}
	if d, ok := r.devices[deviceID]; ok {
		d.lastSeen = time.Now()
		return d.token, nil
	}

	token, err := randomToken()
	if err != nil {
		return "", err
	}
	r.devices[deviceID] = &device{token: token, createdAt: time.Now(), lastSeen: time.Now()}
	r.tokens[token] = deviceID
	return token, nil
}

// Validate trả về deviceID nếu token hợp lệ và device chưa bị ban.
func (r *DeviceRegistry) Validate(token string) (string, bool) {
	if token == "" {
		return "", false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	id, ok := r.tokens[token]
	if !ok {
		return "", false
	}
	if r.banned[id] {
		return "", false
	}
	if d, ok := r.devices[id]; ok {
		d.lastSeen = time.Now()
	}
	return id, true
}

// Ban vô hiệu hoá deviceID và token tương ứng.
func (r *DeviceRegistry) Ban(deviceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.BanLocked(deviceID)
}

func (r *DeviceRegistry) BanLocked(deviceID string) {
	r.banned[deviceID] = true
	if d, ok := r.devices[deviceID]; ok {
		delete(r.tokens, d.token)
	}
}

// BanFromEnv ban một danh sách device_id phân cách bằng dấu phẩy (ban thủ công
// qua biến môi trường BANNED_DEVICES khi khởi động server).
func (r *DeviceRegistry) BanFromEnv(s string) {
	for _, id := range strings.Split(s, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			r.Ban(id)
		}
	}
}

type ctxKey int

const deviceIDKey ctxKey = iota

// DeviceIDFromContext trả về device_id đã xác thực (nếu request dùng token).
func DeviceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(deviceIDKey).(string); ok {
		return v
	}
	return ""
}

// AuthMiddleware chấp nhận credential theo hai cách:
//   - Header X-API-Key: key tĩnh (chỉ nằm trên server, cho admin/curl).
//   - Header X-Device-Token: token per-device từ /api/v1/auth/register.
//
// Khi xác thực bằng token, device_id được gắn vào context (rate limiter theo
// device thay vì IP).
func AuthMiddleware(expectedKey string, reg *DeviceRegistry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key := r.Header.Get("X-API-Key"); key != "" {
				if len(key) == len(expectedKey) &&
					subtle.ConstantTimeCompare([]byte(key), []byte(expectedKey)) == 1 {
					next.ServeHTTP(w, r)
					return
				}
			}

			if tok := r.Header.Get("X-Device-Token"); tok != "" {
				if id, ok := reg.Validate(tok); ok {
					ctx := context.WithValue(r.Context(), deviceIDKey, id)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "missing or invalid credentials",
			})
		})
	}
}
