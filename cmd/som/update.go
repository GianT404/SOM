package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/minio/selfupdate"
)

const releasesListAPI = "https://api.github.com/repos/GianT404/SOM/releases?per_page=30"

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type ghRelease struct {
	TagName    string    `json:"tag_name"`
	Draft      bool      `json:"draft"`
	Prerelease bool      `json:"prerelease"`
	Assets     []ghAsset `json:"assets"`
}

func runSelfUpdate(current string) error {
	if current == "dev" {
		return fmt.Errorf("binary này được build không kèm version (thiếu -ldflags), " +
			"không thể so sánh với bản mới nhất trên GitHub")
	}

	releases, err := fetchReleases()
	if err != nil {
		return err
	}

	assetName := fmt.Sprintf("som-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		assetName += ".exe"
	}

	// releases/latest trả về release mới nhất theo thời gian (có thể là bản
	// mobile APK không chứa binary TUI), nên phải duyệt danh sách để tìm bản
	// mới nhất thực sự có binary của platform hiện tại.
	target := latestReleaseWithAsset(releases, assetName)
	if target == nil {
		return fmt.Errorf("không tìm thấy bản build %q trong các release — "+
			"kiểm tra lại workflow release TUI có build platform này chưa", assetName)
	}

	if target.TagName == current {
		fmt.Println("Bạn đang dùng bản mới nhất rồi:", current)
		return nil
	}

	var downloadURL string
	for _, a := range target.Assets {
		if a.Name == assetName {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}

	fmt.Printf("Đang cập nhật %s → %s...\n", current, target.TagName)

	downloadClient := &http.Client{Timeout: 5 * time.Minute}

	dlResp, err := downloadClient.Get(downloadURL)
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

	fmt.Println("Cập nhật thành công lên", target.TagName, "— chạy lại `som` để dùng bản mới.")
	return nil
}

func fetchReleases() ([]ghRelease, error) {
	httpClient := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequest("GET", releasesListAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "som-tui")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kiểm tra bản mới thất bại: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub trả về lỗi %d khi kiểm tra release "+
			"(bị giới hạn tần suất? đặt biến GITHUB_TOKEN để tăng giới hạn)", resp.StatusCode)
	}

	var rels []ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rels); err != nil {
		return nil, fmt.Errorf("không đọc được thông tin release: %w", err)
	}
	return rels, nil
}

func latestReleaseWithAsset(rels []ghRelease, assetName string) *ghRelease {
	for i := range rels {
		r := &rels[i]
		if r.Draft || r.Prerelease {
			continue
		}
		for _, a := range r.Assets {
			if a.Name == assetName {
				return r
			}
		}
	}
	return nil
}
