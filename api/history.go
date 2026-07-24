package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HistoryEntry struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Messages  int       `json:"messages"`
	CreatedAt time.Time `json:"createdAt"`
}

func HistoryRoutes(db *pgxpool.Pool) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			userID := r.Context().Value(UserIDKey).(string)
			rows, err := db.Query(r.Context(),
				`SELECT s.id, s.title, COUNT(m.id), s.created_at
				 FROM chat_sessions s LEFT JOIN chat_messages m ON m.session_id = s.id
				 WHERE s.user_id = $1 GROUP BY s.id ORDER BY s.updated_at DESC LIMIT 50`, userID,
			)
			if err != nil {
				http.Error(w, `{"error":"failed to get history"}`, http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			entries := make([]HistoryEntry, 0)
			for rows.Next() {
				var e HistoryEntry
				if err := rows.Scan(&e.ID, &e.Title, &e.Messages, &e.CreatedAt); err == nil {
					entries = append(entries, e)
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(entries)
		})
	}
}
