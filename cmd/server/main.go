package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"charm.land/lipgloss/v2"
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

	sc := scraper.NewYtdlpScraperFastPath(ytdlpPath)
	resilientSc := scraper.NewResilientScraper(sc)

	// Device registry: per-device token auth cho mobile app.
	deviceReg := backend.NewDeviceRegistry()
	deviceReg.BanFromEnv(os.Getenv("BANNED_DEVICES"))

	r := newRouter(resilientSc, apiKey, deviceReg)

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
		purples := []color.Color{
			lipgloss.Color("#FFE8DF"),
			lipgloss.Color("#FFB9A7"),
			lipgloss.Color("#E8593C"),
			lipgloss.Color("#C84328"),
			lipgloss.Color("#9D311A"),
			lipgloss.Color("#6B1F0E"),
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

// newRouter dựng toàn bộ chi router (tách riêng để test wiring mà không cần
// khởi động server). Lưu ý: chi yêu cầu Use() phải đặt TRƯỚC khi khai route
// trên cùng một mux — các route cần auth phải nằm trong Group().
func newRouter(sc *scraper.ResilientScraper, apiKey string, deviceReg *backend.DeviceRegistry) http.Handler {
	searchH := &handler.SearchHandler{Scraper: sc}
	streamH := handler.NewStreamHandler(sc)
	lyricsH := &handler.LyricsHandler{Scraper: sc}
	resolveH := &handler.ResolveHandler{Scraper: sc}

	generalLimiter := backend.NewIPRateLimiter(2, 20)
	heavyLimiter := backend.NewIPRateLimiter(0.2, 5)
	registerLimiter := backend.NewIPRateLimiter(1, 5)

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(3 * time.Minute))
	r.Use(corsMiddleware)
	// gzip cho payload JSON nhỏ (search/lyrics/resolve/register). Không liệt
	// kê audio/ogg nên endpoint /stream không bị nén (tránh buffer/độ trễ).
	r.Use(middleware.Compress(5, "application/json", "text/html", "text/plain"))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"dopus"}`))
	})

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Register là endpoint công khai để app xin token per-device.
		// Rate limit riêng theo IP chống spam đăng ký.
		r.With(registerLimiter.Middleware).Post("/auth/register", registerHandler(deviceReg))

		// Mọi route còn lại yêu cầu X-API-Key (key tĩnh trên server) hoặc
		// X-Device-Token hợp lệ. Rate limit theo device_id khi đã xác thực.
		r.Group(func(r chi.Router) {
			r.Use(backend.AuthMiddleware(apiKey, deviceReg))
			r.Use(generalLimiter.Middleware)

			r.Get("/search", searchH.ServeHTTP)
			r.Get("/lyrics", lyricsH.ServeHTTP)

			r.With(heavyLimiter.Middleware).Get("/stream", streamH.ServeHTTP)
			r.With(heavyLimiter.Middleware).Get("/resolve", resolveH.ServeHTTP)
		})
	})

	return r
}

// registerHandler cấp token per-device. Body: {"device_id": "..."}.
func registerHandler(reg *backend.DeviceRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			DeviceID string `json:"device_id"`
		}
		defer r.Body.Close()
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		token, err := reg.Register(strings.TrimSpace(req.DeviceID))
		if err != nil {
			if errors.Is(err, backend.ErrDeviceBanned) {
				writeJSONError(w, http.StatusForbidden, "device is banned")
				return
			}
			writeJSONError(w, http.StatusBadRequest, "invalid device_id")
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"token": token})
	}
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// corsMiddleware adds CORS headers for the web/OpenAPI docs on GitHub Pages
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-API-Key, X-Device-Token")
		w.Header().Set("Access-Control-Max-Age", "3600")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
