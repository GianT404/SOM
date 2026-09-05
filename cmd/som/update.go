package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func runCheckUpdate(current string) error {
	if current == "dev" {
		return fmt.Errorf("this binary was built without an embedded version (missing -ldflags), " +
			"cannot compare with the latest GitHub release")
	}

	assetName := fmt.Sprintf("som-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		assetName += ".exe"
	}

	target, _, err := findUpdate(assetName)
	if err != nil {
		return err
	}

	if versionCompare(target.TagName, current) <= 0 {
		fmt.Println("You are on the latest version:", current)
		return nil
	}

	fmt.Printf("A new version %s is available — you are on %s.\n", target.TagName, current)
	printChangelog(current, target.TagName)
	fmt.Println("Run `som --upgrade` to install it.")
	return nil
}

func runSelfUpdate(current string) error {
	if current == "dev" {
		return fmt.Errorf("this binary was built without an embedded version (missing -ldflags), " +
			"cannot compare with the latest GitHub release")
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
		fmt.Println("You are already on the latest version:", current)
		return nil
	}
	if versionCompare(target.TagName, current) <= 0 {
		fmt.Println("You are already on the latest version:", current)
		return nil
	}

	fmt.Printf("A new version %s is available — you are on %s.\n\n", target.TagName, current)
	printChangelog(current, target.TagName)

	fmt.Printf("Updating %s → %s...\n", current, target.TagName)

	// Xác thực chữ ký minisign của bản phát hành trước khi tải/ghi đè.
	// File ký được publish kèm asset dạng <asset>.minisig.
	verifier, verr := newUpgradeVerifier(downloadURL + ".minisig")
	if verr != nil {
		if !allowUnverified() {
			return fmt.Errorf("refusing to update: %v\nset SOM_ALLOW_UNVERIFIED=1 to skip verification (not recommended)", verr)
		}
		fmt.Println("warning: SOM_ALLOW_UNVERIFIED=1 — skipping signature verification")
	}

	downloadClient := &http.Client{Timeout: 5 * time.Minute}

	dlResp, err := downloadClient.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download new version: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download new version: GitHub returned %d", dlResp.StatusCode)
	}

	var reader io.Reader = dlResp.Body
	total := dlResp.ContentLength
	if total > 0 {
		pr := &progressReader{total: total}
		reader = pr.Wrap(dlResp.Body)
	}

	opts := selfupdate.Options{}
	if verifier != nil {
		opts.Verifier = verifier
	}
	if err := selfupdate.Apply(reader, opts); err != nil {
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			return fmt.Errorf("update failed AND rollback also failed (reinstall manually): %w", rerr)
		}
		return fmt.Errorf("update failed, rolled back to the previous version: %w", err)
	}
	if total > 0 {
		fmt.Printf("\r[%-40s] 100%%  \n", strings.Repeat("#", 40))
	}

	fmt.Println("Successfully updated to", target.TagName, "— run `som` again to use the new version.")
	return nil
}

// newUpgradeVerifier tải chữ ký minisign của bản phát hành và dựng bộ xác thực bằng public key nhúng trong binary.
func newUpgradeVerifier(sigURL string) (*selfupdate.Verifier, error) {
	if somMinisignPublicKey == "" {
		return nil, fmt.Errorf("this binary was built without an embedded signing key")
	}
	v := selfupdate.NewVerifier()
	if err := v.LoadFromURL(sigURL, somMinisignPublicKey, nil); err != nil {
		return nil, fmt.Errorf("failed to fetch or parse signature %s: %w", sigURL, err)
	}
	return v, nil
}

func allowUnverified() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SOM_ALLOW_UNVERIFIED"))) {
	case "1", "true", "yes", "y":
		return true
	}
	return false
}

// findUpdate tìm release mới nhất thực sự chứa binary của platform hiện tại.
func findUpdate(assetName string) (*ghRelease, string, error) {
	if rels, err := fetchReleases(); err == nil {
		if t := latestReleaseWithAsset(rels, assetName); t != nil {
			return t, assetURLOf(t, assetName), nil
		}
		return nil, "", fmt.Errorf("no build %q found in the releases — "+
			"check whether the release workflow builds this platform", assetName)
	}

	tags, ferr := fetchRecentTagsFromAtom()
	if ferr != nil {
		return nil, "", fmt.Errorf("failed to check for updates (GitHub API and release feed both failed)")
	}
	bestTag := ""
	for _, tag := range tags {
		if assetExists(releaseDownloadURL(tag, assetName)) {
			if bestTag == "" || versionCompare(tag, bestTag) > 0 {
				bestTag = tag
			}
		}
	}
	if bestTag != "" {
		return &ghRelease{TagName: bestTag}, releaseDownloadURL(bestTag, assetName), nil
	}
	return nil, "", fmt.Errorf("no build %q found in recent releases", assetName)
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
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var rels []ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rels); err != nil {
		return nil, err
	}
	return rels, nil
}

func latestReleaseWithAsset(rels []ghRelease, assetName string) *ghRelease {
	var best *ghRelease
	for i := range rels {
		r := &rels[i]
		if r.Draft || r.Prerelease {
			continue
		}
		hasAsset := false
		for _, a := range r.Assets {
			if a.Name == assetName {
				hasAsset = true
				break
			}
		}
		if !hasAsset {
			continue
		}
		// Chọn bản có version cao nhất
		if best == nil || versionCompare(r.TagName, best.TagName) > 0 {
			best = r
		}
	}
	return best
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
		return nil, fmt.Errorf("release feed returned %d", resp.StatusCode)
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

const compareAPI = "https://api.github.com/repos/GianT404/SOM/compare/%s...%s"

type ghCompareCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
	} `json:"commit"`
}

// printChangelog liệt kê các commit giữa 2 tag bằng GitHub compare API.
// Nếu API lỗi (rate-limit…) thì im lặng bỏ qua, không chặn việc upgrade.
func printChangelog(base, head string) {
	commits, err := fetchCompareCommits(base, head)
	if err != nil || len(commits) == 0 {
		return
	}

	fmt.Printf("Changes from %s → %s (%d commits):\n", base, head, len(commits))

	const maxShow = 30
	start := len(commits) - maxShow
	if start < 0 {
		start = 0
	}
	for i := len(commits) - 1; i >= start; i-- {
		msg := strings.TrimSpace(strings.SplitN(commits[i].Commit.Message, "\n", 2)[0])
		if msg == "" {
			continue
		}
		sha := commits[i].SHA
		if len(sha) > 7 {
			sha = sha[:7]
		}
		fmt.Printf("  \u2022 %s  %s\n", sha, msg)
	}
	if len(commits) > maxShow {
		fmt.Printf("  \u2026 and %d more commits\n", len(commits)-maxShow)
	}
	fmt.Println()
}

func fetchCompareCommits(base, head string) ([]ghCompareCommit, error) {
	reqURL := fmt.Sprintf(compareAPI, url.PathEscape(base), url.PathEscape(head))

	httpClient := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", reqURL, nil)
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
		return nil, fmt.Errorf("GitHub compare returned %d", resp.StatusCode)
	}

	var cmp struct {
		TotalCommits int               `json:"total_commits"`
		Commits      []ghCompareCommit `json:"commits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&cmp); err != nil {
		return nil, err
	}
	return cmp.Commits, nil
}

type progressReader struct {
	total   int64
	current int64
}

func (pr *progressReader) Wrap(r io.Reader) io.Reader {
	return &progressReading{pr: pr, r: r}
}

type progressReading struct {
	pr *progressReader
	r  io.Reader
}

func (p *progressReading) Read(buf []byte) (int, error) {
	n, err := p.r.Read(buf)
	p.pr.current += int64(n)
	p.pr.print()
	return n, err
}

func (pr *progressReader) print() {
	if pr.total <= 0 {
		return
	}
	pct := int(float64(pr.current) / float64(pr.total) * 100)
	if pct > 100 {
		pct = 100
	}
	filled := pct * 40 / 100
	bar := strings.Repeat("#", filled) + strings.Repeat(" ", 40-filled)
	fmt.Printf("\r[%s] %3d%%", bar, pct)
}
