package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"som/internal/domain"
	"som/internal/local"
	"som/internal/scraper"
	"som/internal/tui/api"
	"som/internal/tui/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-chi/chi/v5/middleware"
)

var Version = "dev"

func main() {
	serverURL := flag.String("server", "", "URL of Google Cloud backend (leave empty to run locally)")
	upgrade := flag.Bool("upgrade", false, "download and install the latest SOM release from GitHub")
	install := flag.Bool("install", false, "copy this binary to /usr/local/bin (or platform equivalent)")
	showVersion := flag.Bool("version", false, "print the current version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("som", Version)
		return
	}
	if *install {
		if err := runInstall(); err != nil {
			fmt.Fprintln(os.Stderr, "Install failed:", err)
			os.Exit(1)
		}
		return
	}
	if *upgrade {
		if err := runSelfUpdate(Version); err != nil {
			fmt.Fprintln(os.Stderr, "Upgrade failed:", err)
			os.Exit(1)
		}
		return
	}

	middleware.DefaultLogger = middleware.RequestLogger(
		&middleware.DefaultLogFormatter{
			Logger:  log.New(io.Discard, "", 0),
			NoColor: true,
		},
	)

	var provider domain.MusicProvider

	if *serverURL == "" {
		ytdlpPath := os.Getenv("YTDLP_PATH")
		if ytdlpPath == "" {
			ytdlpPath = "yt-dlp"
		}
		sc := scraper.NewYtdlpScraper(ytdlpPath)
		provider = &local.DirectProvider{Scraper: sc}
	} else {
		provider = api.NewHTTPProvider(*serverURL)
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

	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

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
}
