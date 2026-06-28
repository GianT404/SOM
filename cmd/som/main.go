package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"som/internal/backend"
	"som/internal/tui/ui"
)

func main() {
	port := "8080"
	ytdlpPath := os.Getenv("YTDLP_PATH")
	if ytdlpPath == "" {
		ytdlpPath = "yt-dlp"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := backend.StartServer(ctx, port, ytdlpPath); err != nil {
			fmt.Fprintf(os.Stderr, "Lỗi khởi chạy backend: %v\n", err)
		}
	}()
	time.Sleep(200 * time.Millisecond)
	serverURL := fmt.Sprintf("http://localhost:%s", port)
	app := ui.NewApp(serverURL)

	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Lỗi giao diện TUI: %v\n", err)
		os.Exit(1)
	}

}
