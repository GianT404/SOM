package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLyricsSearchHandler_ValidatesQueryAndLimit(t *testing.T) {
	tests := []struct {
		name string
		url string
		want int
	}{
		{name: "missing query", url: "/api/v1/lyrics/search", want: http.StatusBadRequest},
		{name: "empty query", url: "/api/v1/lyrics/search?q=%20%20", want: http.StatusBadRequest},
		{name: "query too long", url: "/api/v1/lyrics/search?q=" + strings.Repeat("x", 101), want: http.StatusBadRequest},
		{name: "limit too large", url: "/api/v1/lyrics/search?q=track&limit=8", want: http.StatusBadRequest},
		{name: "invalid limit", url: "/api/v1/lyrics/search?q=track&limit=abc", want: http.StatusBadRequest},
	}
	handler := &LyricsSearchHandler{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tt.url, nil)
			handler.ServeHTTP(recorder, request)
			if recorder.Code != tt.want {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.want)
			}
		})
	}
}
