package backend

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"som/internal/handler"
	"som/internal/scraper"
)

func StartServer(ctx context.Context, port string, ytdlpPath string) error {
	sc := scraper.NewYtdlpScraper(ytdlpPath)

	searchH := &handler.SearchHandler{Scraper: sc}
	streamH := handler.NewStreamHandler(sc)
	lyricsH := &handler.LyricsHandler{Scraper: sc}
	resolveH := &handler.ResolveHandler{Scraper: sc}

	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer)
	r.Use(middleware.Timeout(3 * time.Minute))

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/search", searchH.ServeHTTP)
		r.Get("/stream", streamH.ServeHTTP)
		r.Get("/lyrics", lyricsH.ServeHTTP)
		r.Get("/resolve", resolveH.ServeHTTP)
	})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	errChan := make(chan error, 1)

	go func() {
		log.Printf("Backend khởi chạy tại cổng %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
