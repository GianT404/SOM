package scraper

import "testing"

func TestYtRetryable(t *testing.T) {
	// Các lỗi do player client bị chặn → nên thử client kế tiếp.
	retryable := []string{
		"ERROR: [youtube] abcd1234: HTTP Error 403: Forbidden",
		"ERROR: [youtube] abcd1234: Requested format is not available",
		"ERROR: [youtube] abcd1234: Sign in to confirm you're not a bot. This helps protect our community.",
		"ERROR: [youtube] abcd1234: This video isn't available on this app",
		"ERROR: [youtube] abcd1234: This video is only available to Music Premium members",
		"ERROR: [youtube] abcd1234: This video is not available in your country",
		"ERROR: [youtube] abcd1234: Sign in to view this video",
	}
	for _, msg := range retryable {
		if !ytRetryable(msg) {
			t.Errorf("ytRetryable should be true for: %s", msg)
		}
	}

	// Lỗi vĩnh viễn / network → không retry client.
	notRetryable := []string{
		"ERROR: [youtube] abcd1234: Video unavailable",
		"ERROR: [youtube] abcd1234: Private video",
		"ERROR: [youtube] abcd1234: This video has been removed",
		"ERROR: unable to download video data: context deadline exceeded",
		"ERROR: Could not run yt-dlp: exec: \"yt-dlp\": executable file not found in $PATH",
	}
	for _, msg := range notRetryable {
		if ytRetryable(msg) {
			t.Errorf("ytRetryable should be false for: %s", msg)
		}
	}
}
