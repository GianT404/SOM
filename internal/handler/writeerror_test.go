package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestWriteError_ProducesValidJSON(t *testing.T) {
	cases := []string{
		`error co dau "quote" ben trong`,
		"error co backslash \\ va newline\ntrong stderr yt-dlp",
		"error binh thuong khong co gi dac biet",
		`{"looks":"like json but is not"}`,
	}

	for _, msg := range cases {
		w := httptest.NewRecorder()
		writeError(w, 500, msg)

		var decoded map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("response khong phai JSON hop le voi input %q: %v\nbody: %s", msg, err, w.Body.String())
		}
		if decoded["error"] != msg {
			t.Fatalf("noi dung error bi sai lech, muon %q, got %q", msg, decoded["error"])
		}
	}
}
