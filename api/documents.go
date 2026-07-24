package api

import (
	"encoding/json"
	"net/http"

	"github.com/brandall2021/matematicarag/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Document struct {
	ID           string `json:"id"`
	Filename     string `json:"filename"`
	OriginalName string `json:"originalName"`
	Type         string `json:"type"`
	Size         int64  `json:"size"`
	Status       string `json:"status"`
	CreatedAt    string `json:"createdAt"`
}

func DocumentRoutes(db *pgxpool.Pool, cfg *config.Config) func(r chi.Router) {
	return func(r chi.Router) {
		r.Use(AuthMiddleware(cfg.JWTSecret))
		r.Post("/upload", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"error":"not implemented"}`, http.StatusNotImplemented)
		})
		r.Post("/youtube", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"error":"not implemented"}`, http.StatusNotImplemented)
		})
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]Document{})
		})
		r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		})
		r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		})
	}
}
