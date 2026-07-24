package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/brandall2021/matematicarag/api"
	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/brandall2021/matematicarag/internal/database"
	"github.com/brandall2021/matematicarag/internal/middleware"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(60 * time.Second))
	r.Use(middleware.CORS(cfg.CORSOriginsList()))
	r.Use(middleware.RateLimit(30))

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", api.AuthRoutes(db, cfg))
		r.Route("/chat", api.ChatRoutes(db, cfg))
		r.Route("/rag", api.RagRoutes(db, cfg))
		r.Route("/math", api.MathRoutes())
		r.Route("/documents", api.DocumentRoutes(db, cfg))
		r.Route("/history", api.HistoryRoutes(db))
		r.Route("/settings", api.SettingsRoutes(db))
		r.Route("/stats", api.StatsRoutes(db))
		r.Route("/analytics", api.AnalyticsRoutes(db))
		r.Route("/indexer", api.IndexerRoutes(db, cfg))
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	staticDir := "./static"
	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		staticDir = dir
	}

	fileServer := http.FileServer(http.Dir(staticDir))
	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/health" {
			http.NotFound(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	addr := ":" + cfg.Port
	log.Printf("Server starting on %s", addr)
	log.Printf("Serving static files from %s", staticDir)

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
