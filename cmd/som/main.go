package main

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"syscall"

	"som/internal/domain"
	"som/internal/local"
	"som/internal/scraper"
	"som/internal/storage"
	"som/internal/tui/api"
	"som/internal/tui/ui"

	tea "charm.land/bubbletea/v2"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/spf13/cobra"
)

var Version = "dev"

//go:embed changelog.txt
var embeddedChangelog string

func main() {
	var serverURL string
	var apiKey string
	var downloadDir string
	var upgradeFlag, installFlag, versionFlag, checkUpdateFlag, uninstallFlag, updateYtdlpFlag, changelogFlag, syncAssetsFlag, totalTimeFlag, libraryStatsFlag bool

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
			if syncAssetsFlag {
				installedPath, err := os.Executable()
				if err != nil {
					fmt.Fprintln(os.Stderr, "Asset sync failed:", err)
					os.Exit(1)
				}
				if err := installDesktopAssets(installedPath); err != nil {
					fmt.Fprintln(os.Stderr, "Asset sync failed:", err)
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

			if totalTimeFlag {
				totalTimeDir := downloadDir
				if totalTimeDir == "" {
					if env := os.Getenv("SOM_DOWNLOAD_DIR"); env != "" {
						totalTimeDir = env
					} else {
						totalTimeDir = storage.DefaultDir()
					}
				}
				if err := storage.MigrateFromLegacy(totalTimeDir); err != nil {
					log.Printf("[storage] migration warning: %v", err)
				}

				store, err := storage.Open(totalTimeDir)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Read total listening time failed:", err)
					os.Exit(1)
				}

				totalSeconds := int64(0)
				if value := store.GetSetting(storage.TotalListeningTimeSettingKey); value != "" {
					totalSeconds, err = strconv.ParseInt(value, 10, 64)
					if err != nil || totalSeconds < 0 {
						_ = store.Close()
						fmt.Fprintln(os.Stderr, "Read total listening time failed: invalid value in SQLite settings")
						os.Exit(1)
					}
				}

				if err := store.Close(); err != nil {
					log.Printf("[storage] close warning: %v", err)
				}

				fmt.Printf("Total listening time: %02dh %02dm %02ds\n",
					totalSeconds/3600,
					(totalSeconds/60)%60,
					totalSeconds%60,
				)
				return
			}

			if libraryStatsFlag {
				if err := runLibraryStats(downloadDir); err != nil {
					fmt.Fprintln(os.Stderr, "Read library statistics failed:", err)
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
				if apiKey == "" {
					apiKey = os.Getenv("SOM_API_KEY")
				}
				if apiKey == "" {
					fmt.Fprintln(os.Stderr, "warning: --server set without --api-key or SOM_API_KEY env; the backend will reject every request with 401")
				}
				provider = api.NewHTTPProvider(serverURL, apiKey)
			}

			if downloadDir == "" {
				if env := os.Getenv("SOM_DOWNLOAD_DIR"); env != "" {
					downloadDir = env
				} else {
					downloadDir = storage.DefaultDir()
				}
			}
			if err := storage.MigrateFromLegacy(downloadDir); err != nil {
				log.Printf("[storage] migration warning: %v", err)
			}

			app := ui.NewApp(provider, downloadDir)
			app.SetVersion(Version)
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

			d := app.SessionDuration()
			h := int(d.Hours())
			m := int(d.Minutes()) % 60
			s := int(d.Seconds()) % 60

			session := fmt.Sprintf("%02dh %02dm %02ds", h, m, s)
			fmt.Printf("\n%s\n", ui.StyleHint("Session", session))
		},
	}

	// Đăng ký toàn bộ cờ vào Cobra (tự động ăn khớp completion)
	rootCmd.Flags().StringVar(&serverURL, "server", "", "URL of Google Cloud backend (leave empty to run locally)")
	rootCmd.Flags().StringVar(&apiKey, "api-key", "", "API key for --server remote mode (or set SOM_API_KEY env var)")
	rootCmd.Flags().StringVar(&downloadDir, "download-dir", "", "Directory to store downloaded tracks (default: ~/.local/share/som)")
	rootCmd.Flags().BoolVar(&upgradeFlag, "upgrade", false, "download and install the latest SOM release from GitHub")
	rootCmd.Flags().BoolVar(&installFlag, "install", false, "install this binary to a user-local bin directory")
	rootCmd.Flags().BoolVar(&versionFlag, "version", false, "print the current version and exit")
	rootCmd.Flags().BoolVar(&checkUpdateFlag, "check-update", false, "check whether a newer SOM release exists without installing")
	rootCmd.Flags().BoolVar(&uninstallFlag, "uninstall", false, "remove the installed som binary from your machine")
	rootCmd.Flags().BoolVar(&updateYtdlpFlag, "update-ytdlp", false, "update the yt-dlp binary to the latest version")
	rootCmd.Flags().BoolVar(&changelogFlag, "changelog", false, "print the commits of the current version")
	rootCmd.Flags().BoolVar(
		&syncAssetsFlag,
		"sync-assets",
		false,
		"internal: sync embedded desktop assets",
	)
	rootCmd.Flags().BoolVar(
		&totalTimeFlag,
		"tt",
		false,
		"show total time spent listening to music and exit",
	)
	rootCmd.Flags().BoolVar(
		&libraryStatsFlag,
		"stats",
		false,
		"show total tracks, playlists, and disk usage; then exit",
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// runChangelog in changelog đã được sinh sẵn khi build release TUI.
func runChangelog(_ string) {
	if embeddedChangelog == "" {
		fmt.Println("changelog: no changelog embedded in this build")
		return
	}
	fmt.Print(embeddedChangelog)
}
