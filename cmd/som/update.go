package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/minio/selfupdate"
)

const releasesAPI = "https://api.github.com/repos/GianT404/SOM/releases/latest"

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

func runSelfUpdate(current string) error {
	if current == "dev" {
		return fmt.Errorf("binary này được build không kèm version (thiếu -ldflags), " +
			"không thể so sánh với bản mới nhất trên GitHub")
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequest("GET", releasesAPI, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("kiểm tra bản mới thất bại: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub trả về lỗi %d khi kiểm tra release", resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return fmt.Errorf("không đọc được thông tin release: %w", err)
	}

	if rel.TagName == current {
		fmt.Println("Bạn đang dùng bản mới nhất rồi:", current)
		return nil
	}

	assetName := fmt.Sprintf("som-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		assetName += ".exe"
	}

	var downloadURL string
	for _, a := range rel.Assets {
		if a.Name == assetName {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("không tìm thấy bản build %q trong release %s "+
			"(kiểm tra lại workflow release có build đúng platform này chưa)",
			assetName, rel.TagName)
	}

	fmt.Printf("Đang cập nhật %s → %s...\n", current, rel.TagName)

	dlResp, err := httpClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("tải bản mới thất bại: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return fmt.Errorf("tải bản mới thất bại: GitHub trả về %d", dlResp.StatusCode)
	}

	if err := selfupdate.Apply(dlResp.Body, selfupdate.Options{}); err != nil {
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			return fmt.Errorf("cập nhật lỗi VÀ rollback cũng lỗi (cài lại thủ công): %w", rerr)
		}
		return fmt.Errorf("cập nhật thất bại, đã tự khôi phục bản cũ: %w", err)
	}

	fmt.Println("Cập nhật thành công lên", rel.TagName, "— chạy lại `som` để dùng bản mới.")
	return nil
}
