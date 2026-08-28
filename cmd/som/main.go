package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"

	"som/internal/domain"
	"som/internal/local"
	"som/internal/scraper"
	"som/internal/tui/api"
	"som/internal/tui/ui"

	tea "charm.land/bubbletea/v2"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/spf13/cobra"
)

var Version = "dev"

func main() {
	var serverURL string
	var upgradeFlag, installFlag, versionFlag, checkUpdateFlag, uninstallFlag, updateYtdlpFlag, changelogFlag bool

	var rootCmd = &cobra.Command{
		Use:   "som",
		Short: "SOM - Terminal Music Player",
		Run: func(cmd *cobra.Command, args []string) {
			if updateYtdlpFlag {
				ytdlpPath := os.Getenv("YTDLP_PATH")
				if ytdlpPath == "" {
					ytdlpPath = "yt-dlp"
				}
				if err := scraper.UpdateYtdlp(ytdlpPath); err != nil {
					fmt.Fprintln(os.Stderr, "Update yt-dlp failed:", err)
					os.Exit(1)
				}
				return
			}
			if versionFlag {
				fmt.Println("som", Version)
				return
			}
			if changelogFlag {
				runChangelog(Version)
				return
			}
			if checkUpdateFlag {
				if err := runCheckUpdate(Version); err != nil {
					fmt.Fprintln(os.Stderr, "Check update failed:", err)
					os.Exit(1)
				}
				return
			}
			if uninstallFlag {
				if err := runUninstall(); err != nil {
					fmt.Fprintln(os.Stderr, "Uninstall failed:", err)
					os.Exit(1)
				}
				return
			}
			if installFlag {
				if err := runInstall(); err != nil {
					fmt.Fprintln(os.Stderr, "Install failed:", err)
					os.Exit(1)
				}
				return
			}
			if upgradeFlag {
				if err := runSelfUpdate(Version); err != nil {
					fmt.Fprintln(os.Stderr, "Upgrade failed:", err)
					os.Exit(1)
				}
				return
			}

			// Chạy TUI chính
			middleware.DefaultLogger = middleware.RequestLogger(
				&middleware.DefaultLogFormatter{
					Logger:  log.New(io.Discard, "", 0),
					NoColor: true,
				},
			)

			var provider domain.MusicProvider
			if serverURL == "" {
				ytdlpPath := os.Getenv("YTDLP_PATH")
				if ytdlpPath == "" {
					ytdlpPath = "yt-dlp"
				}
				sc := scraper.NewYtdlpScraper(ytdlpPath)
				provider = &local.DirectProvider{Scraper: sc}
			} else {
				provider = api.NewHTTPProvider(serverURL)
			}

			app := ui.NewApp(provider)

			defer func() {
				if r := recover(); r != nil {
					path := ui.LogBuf.DumpCrash(fmt.Sprintf("panic: %v", r))
					fmt.Fprintf(os.Stderr, "\nsom: crash detected; log dumped to %s\n", path)
					fmt.Fprintf(os.Stderr, "panic: %v\n%s", r, debug.Stack())
					os.Exit(2)
				}
			}()

			p := tea.NewProgram(app)

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
			go func() {
				for s := range sigCh {
					switch s {
					case syscall.SIGQUIT:
						path := ui.LogBuf.DumpCrash("SIGQUIT received")
						fmt.Fprintf(os.Stderr, "\nsom: SIGQUIT; log dumped to %s\n", path)
						os.Exit(130)
					default:
						p.Quit()
					}
				}
			}()
			defer signal.Stop(sigCh)

			if _, err := p.Run(); err != nil {
				if errors.Is(err, tea.ErrProgramPanic) {
					path := ui.LogBuf.DumpCrash("program panic (stack printed above)")
					fmt.Fprintf(os.Stderr, "\nsom: crash detected; log dumped to %s\n", path)
					os.Exit(2)
				}
				fmt.Fprintf(os.Stderr, "Error TUI: %v\n", err)
				os.Exit(1)
			}
		},
	}

	// Đăng ký toàn bộ cờ vào Cobra (tự động ăn khớp completion)
	rootCmd.Flags().StringVar(&serverURL, "server", "", "URL of Google Cloud backend (leave empty to run locally)")
	rootCmd.Flags().BoolVar(&upgradeFlag, "upgrade", false, "download and install the latest SOM release from GitHub")
	rootCmd.Flags().BoolVar(&installFlag, "install", false, "copy this binary to /usr/local/bin (or platform equivalent)")
	rootCmd.Flags().BoolVar(&versionFlag, "version", false, "print the current version and exit")
	rootCmd.Flags().BoolVar(&checkUpdateFlag, "check-update", false, "check whether a newer SOM release exists without installing")
	rootCmd.Flags().BoolVar(&uninstallFlag, "uninstall", false, "remove the installed som binary from your machine")
	rootCmd.Flags().BoolVar(&updateYtdlpFlag, "update-ytdlp", false, "update the yt-dlp binary to the latest version")
	rootCmd.Flags().BoolVar(&changelogFlag, "changelog", false, "print the commits of the current version")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// runChangelog in các commit từ tag trước đến tag hiện tại (version hiện tại).
func runChangelog(version string) {
	tag := strings.TrimSpace(version)
	if tag == "" || tag == "dev" {
		fmt.Fprintln(os.Stderr, "changelog: no version tag available (dev build)")
		return
	}

	prevTag := gitTagPrev(tag)

	var rangeSpec string
	if prevTag != "" {
		rangeSpec = prevTag + ".." + tag
	} else {
		rangeSpec = tag
	}

	out, err := exec.Command("git", "log", rangeSpec, "--oneline", "--no-decorate").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "changelog: git log failed: %v\n", err)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Printf("changelog: %s — no commits found\n", tag)
		return
	}

	fmt.Printf("changelog: %s (%d commits)\n\n", tag, len(lines))
	for _, line := range lines {
		fmt.Println("  " + line)
	}
}

// gitTagPrev tìm tag đứng trước currentTag theo thứ tự tạo (creatordate).
func gitTagPrev(currentTag string) string {
	out, err := exec.Command("git", "tag", "--sort=-creatordate").CombinedOutput()
	if err != nil {
		return ""
	}
	tags := strings.Split(strings.TrimSpace(string(out)), "\n")
	found := false
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == currentTag {
			found = true
			continue
		}
		if found && t != "" {
			return t
		}
	}
	return ""
}
