package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var suggestHTTPClient = &http.Client{Timeout: 3 * time.Second}

// Suggest gọi YouTube autocomplete
func Suggest(ctx context.Context, keyword string) ([]string, error) {
	params := url.Values{
		"client": {"firefox"},
		"ds":     {"yt"},
		"q":      {keyword},
		"gl":     {"US"},
		"hl":     {"en"},
	}
	reqURL := "https://suggestqueries.google.com/complete/search?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := suggestHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("suggest: HTTP %d", resp.StatusCode)
	}

	var raw []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("suggest: decode: %w", err)
	}
	if len(raw) < 2 {
		return nil, fmt.Errorf("suggest: unexpected payload")
	}

	var suggestions []string
	if err := json.Unmarshal(raw[1], &suggestions); err != nil {
		return nil, fmt.Errorf("suggest: decode list: %w", err)
	}

	out := make([]string, 0, len(suggestions))
	for _, s := range suggestions {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out, nil
}
