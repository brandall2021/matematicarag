package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminStats struct {
	TotalUsers    int64 `json:"totalUsers"`
	TotalSessions int64 `json:"totalSessions"`
	TotalMessages int64 `json:"totalMessages"`
	TotalDocs     int64 `json:"totalDocuments"`
}

func StatsRoutes(db *pgxpool.Pool) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
			stats := AdminStats{}
			db.QueryRow(r.Context(), `SELECT COUNT(*) FROM users`).Scan(&stats.TotalUsers)
			db.QueryRow(r.Context(), `SELECT COUNT(*) FROM chat_sessions`).Scan(&stats.TotalSessions)
			db.QueryRow(r.Context(), `SELECT COUNT(*) FROM chat_messages`).Scan(&stats.TotalMessages)
			db.QueryRow(r.Context(), `SELECT COUNT(*) FROM documents`).Scan(&stats.TotalDocs)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(stats)
		})
	}
}
