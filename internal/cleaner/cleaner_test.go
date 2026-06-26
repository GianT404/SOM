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
			input:    "Hà Anh Tuấn - Tháng Tư Là Lời Nói Dối Của Em (Official MV)",
			expected: "Tháng Tư Là Lời Nói Dối Của Em",
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
