package backend

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// trustedProxies là danh sách IP/CIDR được phép cung cấp X-Forwarded-For,
// cấu hình qua biến môi trường TRUSTED_PROXIES (phân cách bằng dấu phẩy).
// Rỗng = KHÔNG tin XFF, chỉ dùng IP kết nối trực tiếp (an toàn mặc định).
var trustedProxies = parseTrustedProxies(os.Getenv("TRUSTED_PROXIES"))

func parseTrustedProxies(s string) []*net.IPNet {
	var out []*net.IPNet
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.Contains(p, "/") {
			if _, ipnet, err := net.ParseCIDR(p); err == nil {
				out = append(out, ipnet)
			}
			continue
		}
		ip := net.ParseIP(p)
		if ip == nil {
			continue
		}
		bits := 32
		if ip.To4() == nil {
			bits = 128
		}
		out = append(out, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
	}
	return out
}

func isTrustedProxy(ip net.IP) bool {
	for _, n := range trustedProxies {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

type visitor struct {
	tokens   float64
	lastSeen time.Time
}

type ipRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     float64
	burst    float64
}

func NewIPRateLimiter(rate, burst float64) *ipRateLimiter {
	l := &ipRateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		burst:    burst,
	}
	go l.cleanupLoop()
	return l
}

func (l *ipRateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	v, ok := l.visitors[key]
	if !ok {
		l.visitors[key] = &visitor{tokens: l.burst - 1, lastSeen: now}
		return true
	}

	elapsed := now.Sub(v.lastSeen).Seconds()
	v.tokens += elapsed * l.rate
	if v.tokens > l.burst {
		v.tokens = l.burst
	}
	v.lastSeen = now

	if v.tokens < 1 {
		return false
	}
	v.tokens--
	return true
}

func (l *ipRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		for key, v := range l.visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(l.visitors, key)
			}
		}
		l.mu.Unlock()
	}
}

// rateLimitKey ưu tiên device_id đã xác thực (trong context từ AuthMiddleware),
// fallback về IP. Tránh việc nhiều thiết bị sau cùng NAT bị chặn lẫn nhau.
func rateLimitKey(r *http.Request) string {
	if id := DeviceIDFromContext(r.Context()); id != "" {
		return "dev:" + id
	}
	return "ip:" + clientIP(r)
}

func (l *ipRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(rateLimitKey(r)) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "rate limit exceeded, please slow down",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	peer := net.ParseIP(host)
	// Chỉ tin XFF khi peer trực tiếp là proxy đã cấu hình (vd Caddy trong
	// docker network). Nếu backend bị lộ ra ngoài, attacker không thể tự đặt
	// XFF để giả IP nữa vì peer của nó không nằm trong trustedProxies.
	if peer != nil && isTrustedProxy(peer) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			last := strings.TrimSpace(parts[len(parts)-1])
			if last != "" {
				return last
			}
		}
	}

	return host
}
