package backend

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIKeyMiddleware_MissingKey(t *testing.T) {
	mw := APIKeyMiddleware("secret123")
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true })

	r := httptest.NewRequest("GET", "/api/v1/search", nil)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("thieu header phai bi 401, got %d", w.Code)
	}
	if called {
		t.Fatal("handler ben trong khong duoc goi khi thieu key")
	}
}

func TestAPIKeyMiddleware_WrongKey(t *testing.T) {
	mw := APIKeyMiddleware("secret123")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	r := httptest.NewRequest("GET", "/api/v1/search", nil)
	r.Header.Set("X-API-Key", "sai-key")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("sai key phai bi 401, got %d", w.Code)
	}
}

func TestAPIKeyMiddleware_CorrectKey(t *testing.T) {
	mw := APIKeyMiddleware("secret123")
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest("GET", "/api/v1/search", nil)
	r.Header.Set("X-API-Key", "secret123")
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("dung key phai duoc 200, got %d", w.Code)
	}
	if !called {
		t.Fatal("handler ben trong phai duoc goi khi key dung")
	}
}

func TestAPIKeyMiddleware_EmptyExpectedKeyStillRejectsMissingHeader(t *testing.T) {

	mw := APIKeyMiddleware("")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	r := httptest.NewRequest("GET", "/api/v1/search", nil)
	w := httptest.NewRecorder()
	mw(next).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expectedKey rong van phai chan request khong co header, got %d", w.Code)
	}
}
