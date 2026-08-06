package backend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRegister_Idempotent(t *testing.T) {
	reg := NewDeviceRegistry()
	id := "device-1234567"

	tok1, err := reg.Register(id)
	if err != nil {
		t.Fatal(err)
	}
	tok2, err := reg.Register(id)
	if err != nil {
		t.Fatal(err)
	}
	if tok1 != tok2 {
		t.Fatalf("register not idempotent: %s != %s", tok1, tok2)
	}
}

func TestRegister_InvalidID(t *testing.T) {
	reg := NewDeviceRegistry()
	for _, id := range []string{
		"",
		"short",
		strings.Repeat("a", 65),
		"bad id!",
		"http://evil.example",
		"id with space",
	} {
		if _, err := reg.Register(id); err != ErrInvalidDeviceID {
			t.Fatalf("expected ErrInvalidDeviceID for %q, got %v", id, err)
		}
	}
}

func TestRegister_TokenEntropy(t *testing.T) {
	reg := NewDeviceRegistry()
	tok1, _ := reg.Register("device-aaaaaaa")
	tok2, _ := reg.Register("device-bbbbbbb")
	if tok1 == tok2 {
		t.Fatal("tokens should be unique")
	}
	if len(tok1) != 64 {
		t.Fatalf("expected 64 hex chars (32 bytes), got %d", len(tok1))
	}
}

func TestValidate_Token(t *testing.T) {
	reg := NewDeviceRegistry()
	id := "device-1234567"
	tok, _ := reg.Register(id)

	got, ok := reg.Validate(tok)
	if !ok || got != id {
		t.Fatalf("expected device %q, got %q ok=%v", id, got, ok)
	}
	if _, ok := reg.Validate("bogus-token"); ok {
		t.Fatal("bogus token accepted")
	}
	if _, ok := reg.Validate(""); ok {
		t.Fatal("empty token accepted")
	}
}

func TestBan_InvalidatesTokenAndRegister(t *testing.T) {
	reg := NewDeviceRegistry()
	id := "device-1234567"
	tok, _ := reg.Register(id)

	reg.Ban(id)

	if _, ok := reg.Validate(tok); ok {
		t.Fatal("banned token still accepted")
	}
	if _, err := reg.Register(id); err != ErrDeviceBanned {
		t.Fatalf("expected ErrDeviceBanned, got %v", err)
	}
}

func TestBanFromEnv(t *testing.T) {
	reg := NewDeviceRegistry()
	reg.BanFromEnv("dev-aaaaa, dev-bbbbb ,,dev-ccccc")

	for _, id := range []string{"dev-aaaaa", "dev-bbbbb", "dev-ccccc"} {
		if _, err := reg.Register(id); err != ErrDeviceBanned {
			t.Fatalf("expected banned for %q, got %v", id, err)
		}
	}
	if _, err := reg.Register("dev-ddddd"); err != nil {
		t.Fatalf("non-banned device should register, got %v", err)
	}
}

func TestAuthMiddleware_KeyOrToken(t *testing.T) {
	reg := NewDeviceRegistry()
	var gotDevice string
	mw := AuthMiddleware("secret-key-123", reg)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotDevice = DeviceIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// Static key hợp lệ.
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "secret-key-123")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("valid api key rejected: %d", rr.Code)
	}
	if gotDevice != "" {
		t.Fatalf("api key request should have no device id, got %q", gotDevice)
	}

	// Sai key tĩnh.
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "wrong")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("wrong api key accepted: %d", rr.Code)
	}

	// Device token hợp lệ.
	id := "device-1234567"
	tok, _ := reg.Register(id)
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Device-Token", tok)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("valid device token rejected: %d", rr.Code)
	}
	if gotDevice != id {
		t.Fatalf("expected device %q in context, got %q", id, gotDevice)
	}

	// Token bị ban.
	reg.Ban(id)
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Device-Token", tok)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("banned token accepted: %d", rr.Code)
	}

	// Không có credential.
	req = httptest.NewRequest("GET", "/", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("request without credentials accepted: %d", rr.Code)
	}
}
