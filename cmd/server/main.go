package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"dm4a/internal/handler"
	"dm4a/internal/scraper"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ytdlpPath := os.Getenv("YTDLP_PATH")
	if ytdlpPath == "" {
		ytdlpPath = "yt-dlp"
	}

	// Create the scraper implementation.
	sc := scraper.NewYtdlpScraper(ytdlpPath)

	// Wire up handlers.
	searchH := &handler.SearchHandler{Scraper: sc}
	streamH := handler.NewStreamHandler(sc)
	lyricsH := &handler.LyricsHandler{Scraper: sc}
	resolveH := &handler.ResolveHandler{Scraper: sc}

	// Build the chi router.
	r := chi.NewRouter()

	// Middleware stack.
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(3 * time.Minute))
	r.Use(corsMiddleware)

	// Health check.
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","service":"dm4a"}`))
	})

	// API v1 routes.
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/search", searchH.ServeHTTP)
		r.Get("/stream", streamH.ServeHTTP)
		r.Get("/lyrics", lyricsH.ServeHTTP)
		r.Get("/resolve", resolveH.ServeHTTP)
	})

	// Create the HTTP server.
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 5 * time.Minute, // long enough for audio proxy streaming
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println(`
 ________       ________      _____ ______      
|\   ____\     |\   __  \    |\   _ \  _   \    
\ \  \___|_    \ \  \|\  \   \ \  \\\__\ \  \   
 \ \_____  \    \ \  \\\  \   \ \  \\|__| \  \  
  \|____|\  \    \ \  \\\  \   \ \  \    \ \  \ 
    ____\_\  \    \ \_______\   \ \__\    \ \__\
   |\_________\    \|_______|    \|__|     \|__|
   \|_________|                                                                  
`)
		log.Printf("Dm4a server starting on :%s", port)
		log.Printf("   yt-dlp binary: %s", ytdlpPath)
		log.Println("   Endpoints:")
		log.Println("     GET /api/v1/search?q={keyword}")
		log.Println("     GET /api/v1/stream?id={video_id}")
		log.Println("     GET /api/v1/lyrics?id={video_id}")
		log.Println("     GET /api/v1/resolve?id={video_id}")
		log.Println("     GET /health")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
}

// corsMiddleware adds CORS headers for the mobile app.
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
