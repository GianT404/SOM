package cleaner

import (
	"testing"
)

func TestCleanYouTubeTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Xử lý Official MV",
			input:    "Tatarka - KAWAII (sped up)",
			expected: " KAWAII",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := CleanYouTubeTitle(tt.input)
			if actual != tt.expected {
				t.Errorf("\nĐầu vào: %s\nKỳ vọng: %s\nThực tế: %s", tt.input, tt.expected, actual)
			}
		})
	}
}
