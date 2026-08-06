package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"runtime"
	"time"

	"github.com/minio/selfupdate"
)

func runSelfUpdate(current string) error {
	if current == "dev" {
		return fmt.Errorf("binary này được build không kèm version (thiếu -ldflags), " +
			"không thể so sánh với bản mới nhất trên GitHub")
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}

	tag, err := latestReleaseTag(httpClient)
	if err != nil {
		return err
	}

	if tag == current {
		fmt.Println("Bạn đang dùng bản mới nhất rồi:", current)
		return nil
	}

	assetName := fmt.Sprintf("som-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		assetName += ".exe"
	}

	downloadURL := fmt.Sprintf(
		"https://github.com/GianT404/SOM/releases/download/%s/%s", tag, assetName,
	)

	fmt.Printf("Đang cập nhật %s → %s...\n", current, tag)

	downloadClient := &http.Client{Timeout: 5 * time.Minute}

	dlResp, err := downloadClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("tải bản mới thất bại: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return fmt.Errorf("tải bản mới thất bại: GitHub trả về %d cho %s "+
			"(kiểm tra lại workflow release có build đúng platform %s/%s chưa)",
			dlResp.StatusCode, downloadURL, runtime.GOOS, runtime.GOARCH)
	}

	if err := selfupdate.Apply(dlResp.Body, selfupdate.Options{}); err != nil {
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			return fmt.Errorf("cập nhật lỗi VÀ rollback cũng lỗi (cài lại thủ công): %w", rerr)
		}
		return fmt.Errorf("cập nhật thất bại, đã tự khôi phục bản cũ: %w", err)
	}

	fmt.Println("Cập nhật thành công lên", tag, "— chạy lại `som` để dùng bản mới.")
	return nil
}

func latestReleaseTag(httpClient *http.Client) (string, error) {
	noRedirectClient := &http.Client{
		Timeout: httpClient.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := noRedirectClient.Get("https://github.com/GianT404/SOM/releases/latest")
	if err != nil {
		return "", fmt.Errorf("kiểm tra release mới thất bại: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusMovedPermanently {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("không đọc được release mới nhất, GitHub trả về %d: %s",
			resp.StatusCode, describeGitHubError(body))
	}

	loc := resp.Header.Get("Location")
	tag := path.Base(loc)
	if tag == "" || tag == "." || tag == "releases" {
		return "", fmt.Errorf("không đọc được tag từ Location header: %q "+
			"(có thể repo chưa có release nào)", loc)
	}
	return tag, nil
}

func describeGitHubError(body []byte) string {
	var e struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &e) == nil && e.Message != "" {
		return e.Message
	}
	return string(body)
}
