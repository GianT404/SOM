package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"som/internal/backend"
	"som/internal/handler"
	"som/internal/scraper"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	apiKey := os.Getenv("SOM_API_KEY")
	if apiKey == "" {
		slog.Error("Missing environment variable SOM_API_KEY, server refuses to start.")
		os.Exit(1)
	}
	go func() {
		slog.Info("Starting pprof monitoring", "port", "6060")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			slog.Error("pprof server failed", "error", err.Error())
		}
	}()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	host := os.Getenv("HOST")

	ytdlpPath := os.Getenv("YTDLP_PATH")
	if ytdlpPath == "" {
		ytdlpPath = "yt-dlp"
	}

	sc := scraper.NewYtdlpScraper(ytdlpPath)

	searchH := &handler.SearchHandler{Scraper: sc}
	streamH := handler.NewStreamHandler(sc)
	lyricsH := &handler.LyricsHandler{Scraper: sc}
	resolveH := &handler.ResolveHandler{Scraper: sc}

	generalLimiter := backend.NewIPRateLimiter(2, 20)
	heavyLimiter := backend.NewIPRateLimiter(0.2, 5)
	// Build the chi router
	r := chi.NewRouter()

	// Middleware stack
	// r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(3 * time.Minute))
	r.Use(corsMiddleware)

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"dopus"}`))
	})

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(backend.APIKeyMiddleware(apiKey))
		r.Use(generalLimiter.Middleware)

		r.Get("/search", searchH.ServeHTTP)
		r.Get("/lyrics", lyricsH.ServeHTTP)

		r.With(heavyLimiter.Middleware).Get("/stream", streamH.ServeHTTP)
		r.With(heavyLimiter.Middleware).Get("/resolve", resolveH.ServeHTTP)
	})

	// Create the HTTP server
	srv := &http.Server{
		Addr:         serverAddr(host, port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 5 * time.Minute, // long enough for audio proxy streaming
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		purples := []lipgloss.Color{
			"#FFE8DF",
			"#FFB9A7",
			"#E8593C",
			"#C84328",
			"#9D311A",
			"#6B1F0E",
		}
		art := []string{
			"     ███████╗   ██████╗   ███╗   ███╗",
			"     ██╔════╝  ██╔═══██╗  ████╗ ████║",
			"     ███████╗  ██║   ██║  ██╔████╔██║",
			"     ╚════██║  ██║   ██║  ██║╚██╔╝██║",
			"     ███████║  ╚██████╔╝  ██║ ╚═╝ ██║",
			"     ╚══════╝   ╚═════╝   ╚═╝     ╚═╝ v.2.3",
		}

		// In ra ASCII art áp dụng màu lipgloss theo từng dòng
		for i, line := range art {
			fmt.Println(lipgloss.NewStyle().Foreground(purples[i]).Render(line))
		}

		log.Printf("Server starting on %s", srv.Addr)
		log.Printf("   yt-dlp binary: %s", ytdlpPath)
		log.Println("   Endpoints:")
		log.Println("     GET /api/v1/search?q={keyword}")
		log.Println("     GET /api/v1/stream?id={video_id}")
		log.Println("     GET /api/v1/lyrics?id={video_id}")
		log.Println("     GET /api/v1/resolve?id={video_id}")
		log.Println("     GET /health")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err.Error())
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("forced shutdown", "error", err.Error())
		os.Exit(1)
	}

	slog.Info("Server stopped gracefully")

}

func serverAddr(host string, port string) string {
	if host == "" {
		return ":" + port
	}
	return net.JoinHostPort(host, port)
}

// corsMiddleware adds CORS headers for the mobile app
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
