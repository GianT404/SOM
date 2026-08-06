package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/minio/selfupdate"
)

const releasesListAPI = "https://api.github.com/repos/GianT404/SOM/releases?per_page=30"

const releasesAtomURL = "https://github.com/GianT404/SOM/releases.atom"

const repoDownloadBase = "https://github.com/GianT404/SOM/releases/download"

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

	assetName := fmt.Sprintf("som-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		assetName += ".exe"
	}

	target, downloadURL, err := findUpdate(assetName)
	if err != nil {
		return err
	}

	if target.TagName == current {
		fmt.Println("Bạn đang dùng bản mới nhất rồi:", current)
		return nil
	}
	if versionCompare(target.TagName, current) <= 0 {
		fmt.Println("Bạn đang dùng bản mới nhất rồi:", current)
		return nil
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

// findUpdate tìm release mới nhất thực sự chứa binary của platform hiện tại.
// Ưu tiên gọi API (cho biết chính xác draft/prerelease/assets); nếu API lỗi
// (thường là rate-limit 403 khi không có GITHUB_TOKEN) thì fallback sang atom
// feed + thử URL tải trực tiếp — không bị giới hạn, không cần token.
func findUpdate(assetName string) (*ghRelease, string, error) {
	if rels, err := fetchReleases(); err == nil {
		if t := latestReleaseWithAsset(rels, assetName); t != nil {
			return t, assetURLOf(t, assetName), nil
		}
		return nil, "", fmt.Errorf("không tìm thấy bản build %q trong các release — "+
			"kiểm tra lại workflow release TUI có build platform này chưa", assetName)
	}

	tags, ferr := fetchRecentTagsFromAtom()
	if ferr != nil {
		return nil, "", fmt.Errorf("kiểm tra bản mới thất bại (GitHub API lỗi và feed cũng lỗi)")
	}
	for _, tag := range tags {
		u := releaseDownloadURL(tag, assetName)
		if assetExists(u) {
			return &ghRelease{TagName: tag}, u, nil
		}
	}
	return nil, "", fmt.Errorf("không tìm thấy bản build %q trong các release gần đây", assetName)
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
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API trả về %d", resp.StatusCode)
	}

	var rels []ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rels); err != nil {
		return nil, err
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

func assetURLOf(r *ghRelease, assetName string) string {
	for _, a := range r.Assets {
		if a.Name == assetName {
			return a.BrowserDownloadURL
		}
	}
	return releaseDownloadURL(r.TagName, assetName)
}

type atomFeed struct {
	Entries []struct {
		Links []struct {
			Href string `xml:"href,attr"`
		} `xml:"link"`
	} `xml:"entry"`
}

func fetchRecentTagsFromAtom() ([]string, error) {
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Get(releasesAtomURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feed release trả về %d", resp.StatusCode)
	}

	var feed atomFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, err
	}

	var tags []string
	for _, e := range feed.Entries {
		for _, l := range e.Links {
			idx := strings.Index(l.Href, "/releases/tag/")
			if idx >= 0 {
				tags = append(tags, l.Href[idx+len("/releases/tag/"):])
				break
			}
		}
	}
	return tags, nil
}

func releaseDownloadURL(tag, assetName string) string {
	return fmt.Sprintf("%s/%s/%s", repoDownloadBase, tag, assetName)
}

func assetExists(u string) bool {
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Head(u)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

var versionPattern = regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)

func versionNums(tag string) [3]int {
	m := versionPattern.FindStringSubmatch(tag)
	if len(m) != 4 {
		return [3]int{}
	}
	var v [3]int
	for i := 0; i < 3; i++ {
		v[i], _ = strconv.Atoi(m[i+1])
	}
	return v
}

// versionCompare trả về -1/0/+1 khi a nhỏ hơn/bằng/lớn hơn b.
func versionCompare(a, b string) int {
	ma, mb := versionNums(a), versionNums(b)
	for i := 0; i < 3; i++ {
		if ma[i] < mb[i] {
			return -1
		}
		if ma[i] > mb[i] {
			return 1
		}
	}
	return 0
}
