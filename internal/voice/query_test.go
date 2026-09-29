package voice

import "testing"

func TestExtractSearchQuery(t *testing.T) {
	tests := []struct {
		name       string
		transcript string
		want       string
	}{
		{name: "vietnamese", transcript: "tìm bài Numb", want: "Numb"},
		{name: "vietnamese song", transcript: "Tìm kiếm bài hát Numb", want: "Numb"},
		{name: "english", transcript: "search for numb", want: "numb"},
		{name: "stt variant", transcript: "Sợp bài Chán Gái 505", want: "Chán Gái 505"},
		{name: "passthrough", transcript: "Numb", want: "Numb"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ExtractSearchQuery(test.transcript)
			if got != test.want {
				t.Fatalf("query = %q, want %q", got, test.want)
			}
		})
	}
}
