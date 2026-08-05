package backend

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientIP_XFF_TakesLastHop(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 10.0.0.1")
	r.RemoteAddr = "10.0.0.1:12345"

	got := clientIP(r)
	want := "10.0.0.1"
	if got != want {
		t.Fatalf("clientIP = %q, muon %q (phai lay hop cuoi, khong lay gia tri client tu khai)", got, want)
	}
}

func TestClientIP_NoXFF_FallsBackToRemoteAddr(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "203.0.113.9:54321"

	got := clientIP(r)
	if got != "203.0.113.9" {
		t.Fatalf("clientIP = %q, muon 203.0.113.9", got)
	}
}

func TestClientIP_MalformedRemoteAddr(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "khong-phai-host-port"

	got := clientIP(r)
	if got != "khong-phai-host-port" {
		t.Fatalf("clientIP phai fallback nguyen RemoteAddr khi khong parse duoc, got %q", got)
	}
}

func TestAllow_BurstThenBlock(t *testing.T) {
	l := &ipRateLimiter{visitors: make(map[string]*visitor), rate: 1, burst: 2}

	if !l.allow("1.2.3.4") {
		t.Fatal("request 1 trong burst phai duoc allow")
	}
	if !l.allow("1.2.3.4") {
		t.Fatal("request 2 trong burst phai duoc allow")
	}
	if l.allow("1.2.3.4") {
		t.Fatal("request 3 vuot burst phai bi chan")
	}
}

func TestAllow_RefillOverTime(t *testing.T) {
	l := &ipRateLimiter{visitors: make(map[string]*visitor), rate: 10, burst: 1}

	if !l.allow("5.5.5.5") {
		t.Fatal("request dau tien phai duoc allow")
	}
	if l.allow("5.5.5.5") {
		t.Fatal("request thu 2 ngay lap tuc phai bi chan")
	}

	time.Sleep(150 * time.Millisecond)
	if !l.allow("5.5.5.5") {
		t.Fatal("sau khi doi du lau, token phai duoc refill")
	}
}

func TestAllow_DifferentIPsIndependent(t *testing.T) {
	l := &ipRateLimiter{visitors: make(map[string]*visitor), rate: 1, burst: 1}

	if !l.allow("1.1.1.1") {
		t.Fatal("ip 1 phai duoc allow")
	}
	if !l.allow("2.2.2.2") {
		t.Fatal("ip 2 doc lap voi ip 1, phai duoc allow rieng")
	}
}

func TestMiddleware_BlocksAfterBurst(t *testing.T) {
	l := &ipRateLimiter{visitors: make(map[string]*visitor), rate: 0.001, burst: 1}
	called := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusOK)
	})
	mw := l.Middleware(next)

	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "9.9.9.9:1"

	w1 := httptest.NewRecorder()
	mw.ServeHTTP(w1, r)
	if w1.Code != http.StatusOK {
		t.Fatalf("request dau tien phai qua duoc, code=%d", w1.Code)
	}

	w2 := httptest.NewRecorder()
	mw.ServeHTTP(w2, r)
	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("request thu 2 vuot burst phai bi 429, code=%d", w2.Code)
	}
	if w2.Header().Get("Retry-After") == "" {
		t.Fatal("response 429 phai co header Retry-After")
	}
	if called != 1 {
		t.Fatalf("handler chi duoc goi 1 lan, thuc te %d lan", called)
	}
}
